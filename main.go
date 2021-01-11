package main

import (
	"context"
	"fmt"
	glog "log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	//"runtime"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/codegangsta/inject"
	"github.com/facebookgo/grace/gracenet"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	_ "google.golang.org/grpc/resolver/passthrough"

	//pfc "gitlab.xxxxx.com/golang-lib/github-com_niean_goperfcounter_6552ca"
	"github.com/zer0131/toolbox/log"
	//"github.com/zer0131/toolbox"
	"github.com/zer0131/toolbox/cron"
	"github.com/zer0131/toolbox/layer"
	//_ "github.com/zer0131/toolbox/resolver/list"
	//_ "github.com/zer0131/toolbox/resolver/sfns"
	//_ "github.com/zer0131/soa/toolbox/resolver/sfnsall"
	"github.com/zer0131/toolbox/interceptor/header"
	"github.com/zer0131/toolbox/interceptor/perfcounter"

	"boxfox_grpc_server/framework/app"
	"boxfox_grpc_server/handler"
	"boxfox_grpc_server/hook"
	pb "boxfox_grpc_server/proto"
)

const (
	defaultPushTimeInterval = 10
	configPath              = "./conf/app.ini"
)

var (
	grpcSrv *grpc.Server
	httpSrv *http.Server
)

func main() {
	grpclog.SetLogger(glog.New(os.Stdout, "", glog.LstdFlags))

	ctx := log.NewContextWithLogID(context.Background())

	startLoadingConfig(ctx)

	var err error
	if reflect.ValueOf(app.ConfigVal.BaseVal).FieldByName("LogExpireDay").IsValid() {
		err = log.InitV4(log.WithProject("boxfox_grpc_server"), log.WithMaxLength(app.ConfigVal.BaseVal.LogSize), log.WithExpireDay(reflect.ValueOf(app.ConfigVal.BaseVal).FieldByName("LogExpireDay").Int()))
	} else {
		err = log.InitV4(log.WithProject("boxfox_grpc_server"), log.WithMaxLength(app.ConfigVal.BaseVal.LogSize))
	}
	// 等待log写入文件完毕再退出
	defer log.Close()

	log.SetLogLevel(app.ConfigVal.BaseVal.LogLevel)

	hook := &hook.DpHook{}
	rpcBoxfoxGrpcServerServiceHandler := &handler.BoxfoxGrpcServerServiceHandler{}
	// handler -> service -> model 依赖注入
	injectHandler(hook, rpcBoxfoxGrpcServerServiceHandler)

	if err = hook.OnAfterLoadConfig(ctx); err != nil {
		log.Errorf(ctx, "err=%+v", err)
		return
	}

	var gn gracenet.Net
	if !app.ConfigVal.BaseVal.DisableWebServer {
		addr := fmt.Sprintf(":%d", app.ConfigVal.BaseVal.Port)
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			panic(err)
		}
		var gn gracenet.Net
		lis, err := gn.ListenTCP("tcp", tcpAddr)
		if err != nil {
			log.Errorf(ctx, "err=%s", err.Error())
			return
		}

		// 引入cmux合并协议
		tcpMux := cmux.New(lis)

		grpcL := tcpMux.MatchWithWriters(cmux.HTTP2MatchHeaderFieldPrefixSendSettings("content-type", "application/grpc"))
		httpL := tcpMux.Match(cmux.HTTP1Fast())

		startGrpc(ctx, grpcL, rpcBoxfoxGrpcServerServiceHandler)
		startHttp(ctx, addr, httpL, rpcBoxfoxGrpcServerServiceHandler)

		go tcpMux.Serve()
	}

	//go runtimeStats(ctx)

	// 热升级
	stopChan := make(chan bool)
	go handleSigs(ctx, stopChan)
	select {
	case graceful := <-stopChan:
		if graceful && !app.ConfigVal.BaseVal.DisableWebServer {
			pid, err := gn.StartProcess()
			if err != nil {
				log.Errorf(ctx, "err=%s", err.Error())
			}
			log.Infof(ctx, "Start new process, pid=%d", pid)

			grpcSrv.GracefulStop()
			if err := httpSrv.Shutdown(ctx); err != nil {
				log.Errorf(ctx, "err=%s", err.Error())
			}
			log.Info(ctx, "graceful stop, pid=%", os.Getpid())
		}
	}

	hook.OnShutdown(ctx)

	time.Sleep(10 * time.Millisecond)
}

//统计runtime的指标
//func runtimeStats(ctx context.Context) {
//	for {
//		select {
//		case <-ctx.Done():
//			break
//		default:
//			time.Sleep(defaultPushTimeInterval * time.Second)
//			s := &runtime.MemStats{}
//			runtime.ReadMemStats(s)
//			//服务goroutine的数量
//			pfc.Gauge(toolbox.StatMetrix("NumGoroutine"), int64(runtime.NumGoroutine()))
//			//服务现在使用的内存
//			pfc.Gauge(toolbox.StatMetrix("Sys"), int64(s.Sys))
//			//垃圾回收占用服务CPU工作的时间总和。如果有100个goroutine，垃圾回收的时间为1S,那么久占用了100S
//			pfc.Gauge(toolbox.StatMetrix("GCCPUFraction"), int64(s.GCCPUFraction))
//			//NumGC is the number of completed GC cycles.
//			pfc.Gauge(toolbox.StatMetrix("NumGC"), int64(s.NumGC))
//			//垃圾回收或者其他信息收集导致服务暂停的次数
//			pfc.Gauge(toolbox.StatMetrix("PauseTotalNs"), int64(s.PauseTotalNs))
//			//下次回收的垃圾内存空间
//			pfc.Gauge(toolbox.StatMetrix("NextGC"), int64(s.NextGC))
//		}
//	}
//}

func handleSigs(ctx context.Context, stopChan chan bool) {
	sigChan := make(chan os.Signal)

	hookableSignals := []os.Signal{
		syscall.SIGHUP,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGTSTP,
	}

	signal.Notify(
		sigChan,
		hookableSignals...,
	)

	for {
		sig := <-sigChan
		pid := syscall.Getpid()
		switch sig {
		case syscall.SIGHUP:
			log.Infof(ctx, "Received SIGHUP. forking. Old pid=%d", pid)
			stopChan <- true
		case syscall.SIGINT, syscall.SIGTERM:
			log.Infof(ctx, "Received %v. pid=%d", sig, pid)
			stopChan <- false
		case syscall.SIGUSR1:
			log.Infof(ctx, "Received SIGUSR1  %v. pid=%d", sig, pid)
			for {
				if err := app.InitProjectConfig(ctx, configPath); err != nil {
					log.Errorf(ctx, "err=%s", err.Error())
					time.Sleep(1 * time.Second)
					continue
				}
				break
			}
		default:
			log.Infof(ctx, "Received %v: nothing i care about...\n", sig)
		}
	}
}

func startLoadingConfig(ctx context.Context) {
	for {
		if err := app.LoadConfig(ctx, configPath); err != nil {
			fmt.Printf("Load Config failed. err = %s", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
	fmt.Println("Load config completed.")
}

func injectHandler(hook *hook.DpHook, rpcBoxfoxGrpcServerServiceHandler *handler.BoxfoxGrpcServerServiceHandler) {
	injector := inject.New()
	for inf, m := range layer.ModelList() {
		injector.MapTo(m, inf)
	}
	for modelWrapperType, modelWrapper := range layer.ModelWrapperList() {
		if err := injector.Apply(modelWrapper); err != nil {
			panic(err)
		}
		injector.MapTo(modelWrapper, modelWrapperType)
	}
	for pluginsType, plugins := range layer.PluginsList() {
		injector.MapTo(plugins, pluginsType)
	}
	for inf, s := range layer.ServiceList() {
		if err := injector.Apply(s); err != nil {
			// 参考spring，直接panic
			panic(err)
		}
		injector.MapTo(s, inf)
	}
	for serviceWrapperType, serviceWrapper := range layer.ServiceWrapperList() {
		if err := injector.Apply(serviceWrapper); err != nil {
			panic(err)
		}
		injector.MapTo(serviceWrapper, serviceWrapperType)
	}

	if err := injector.Apply(&rpcBoxfoxGrpcServerServiceHandler); err != nil {
		// 参考spring，直接panic
		panic(err)
	}

	for _, worker := range cron.WorkerList() {
		if err := injector.Apply(worker); err != nil {
			panic(err)
		}
	}
	// 支持hook struct的依赖注入
	if err := injector.Apply(hook); err != nil {
		panic(err)
	}
}

func startGrpc(ctx context.Context, grpcL net.Listener, rpcBoxfoxGrpcServerServiceHandler *handler.BoxfoxGrpcServerServiceHandler) {
	grpcSrv = grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			header.UnaryServerInterceptor(),
			perfcounter.UnaryServerInterceptor(),
		)),
		grpc.MaxRecvMsgSize(1024*1024*128),
	)

	pb.RegisterBoxfoxGrpcServerServiceServer(grpcSrv, rpcBoxfoxGrpcServerServiceHandler)

	go func() {
		if err := grpcSrv.Serve(grpcL); err != nil {
			log.Errorf(ctx, "err=%s", err.Error())
		}
	}()
}

func startHttp(ctx context.Context, addr string, httpL net.Listener, rpcBoxfoxGrpcServerServiceHandler *handler.BoxfoxGrpcServerServiceHandler) {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.Handle("/", &pb.HTTPHandler{})

	pb.RegisterHTTPBoxfoxGrpcServerServiceServer(rpcBoxfoxGrpcServerServiceHandler)

	httpSrv = &http.Server{Addr: addr}
	go func() {
		if err := httpSrv.Serve(httpL); err != nil {
			log.Errorf(ctx, "err=%s", err.Error())
		}
	}()
}
