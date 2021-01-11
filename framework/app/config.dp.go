package app

type autoBase struct {
	Port             int64
	LogLevel         string
	LogSize          int64
	Group            string
	Project          string
	DisableWebServer bool
	Type             string
}
type autoBoxfoxGrpcServer struct {
	A int64
	B []int64
	C string
	D []string
	E bool
}
type Config struct {
	BaseVal             autoBase
	BoxfoxGrpcServerVal autoBoxfoxGrpcServer
}
