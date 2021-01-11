package demo

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	//"github.com/zer0131/toolbox"
	"github.com/zer0131/toolbox/accesslog"
	"github.com/zer0131/toolbox/cron"
	"github.com/zer0131/toolbox/httplib"
	"github.com/zer0131/toolbox/log"

	"google.golang.org/grpc/metadata"
	//pfc "gitlab.xxxxx.com/golang-lib/github-com_niean_goperfcounter_6552ca"
)

var (
	httpBoxfoxGrpcServerServiceServer BoxfoxGrpcServerServiceServer
)

func RegisterHTTPBoxfoxGrpcServerServiceServer(srv BoxfoxGrpcServerServiceServer) {
	httpBoxfoxGrpcServerServiceServer = srv
}

type httpMethodHandler func(ctx context.Context, req interface{}) (interface{}, error)

var _HTTP_Path_Handler = map[string]httpMethodHandler{

	"/demo.BoxfoxGrpcServerService/Monitor": _HTTP_BoxfoxGrpcServerService_Monitor_Handler,
}

type HTTPHandler struct{}

func (h *HTTPHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	//metrix := toolbox.StatProtoMetrix(r.URL.Path)

	// qps统计：60s上传一次，可以观察75% 95% 99%三个指标
	//pfc.Meter(metrix, 1)

	// 耗时统计
	//start := time.Now()
	//defer func() {
	//	pfc.Histogram(metrix, time.Since(start).Nanoseconds()/(1000*1000))
	//}()

	ctx := httplib.Header2IncomingContext(r.Header)
	logId, _ := log.LogIdFromContext(ctx)
	if logId == "" {
		ctx = log.NewGrpcContextWithLogID(ctx)
	} else {
		ctx = metadata.AppendToOutgoingContext(ctx, log.LogIDKey, logId)
	}

	var (
		b         []byte
		berr      error
		isFormReq bool
	)
	switch r.Method {
	case http.MethodGet:
		isFormReq = true
	case http.MethodPost:
		ct := r.Header.Get("Content-Type")
		if ct == "application/x-www-form-urlencoded" {
			isFormReq = true
		}
	}
	if isFormReq {
		if err := r.ParseForm(); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
			return
		}

		// Form到proto数据结构的直接转化，要求proto在设计时，属性都是数组
		b, berr = json.Marshal(r.Form)
		if berr != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(berr.Error()))
			return
		}
	} else {
		// application/json
		b, berr = ioutil.ReadAll(r.Body)
		if berr != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(berr.Error()))
			return
		}
	}

	// 定时任务
	schedulePrefix := "/schedule/"
	if strings.HasPrefix(r.URL.Path, schedulePrefix) {
		name := strings.TrimPrefix(r.URL.Path, schedulePrefix)
		if err := cron.Run(ctx, name, b); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
		} else {
			logId, _ := log.LogIdFromContext(ctx)
			rw.Write([]byte(fmt.Sprintf("log-id=%s 请去自己的服务根据logid查看日志", logId)))
		}
		return
	}
	accesslog.PrintLog(ctx, r)
	accessTime := time.Now()
	handleFunc, ok := _HTTP_Path_Handler[r.URL.Path]
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	resp, err := handleFunc(ctx, b)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	respb, err := json.Marshal(resp)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		return
	}
	rw.Write(respb)
	endTime := time.Now()
	spendTime := float32(endTime.UnixNano()-accessTime.UnixNano()) / 1e9
	endTimeFormat := endTime.Format("06-01-02 15:04:05.999")
	accessTimeFormat := accessTime.Format("06-01-02 15:04:05.999")
	log.Infof(ctx, "spendTime=%fs accessTime=%s endTime=%s", spendTime, accessTimeFormat, endTimeFormat)
}

func _HTTP_BoxfoxGrpcServerService_Monitor_Handler(ctx context.Context, req interface{}) (interface{}, error) {
	var protoReq MonitorReq
	if err := json.Unmarshal(req.([]byte), &protoReq); err != nil {
		return nil, err
	}
	return httpBoxfoxGrpcServerServiceServer.Monitor(ctx, &protoReq)
}
