package handler

import (
	pb "boxfox_grpc_server/proto"
	"context"
)

type BoxfoxGrpcServerServiceHandler struct{}

func (s *BoxfoxGrpcServerServiceHandler) Monitor(ctx context.Context, in *pb.MonitorReq) (*pb.MonitorResp, error) {
	if err := in.Validate(); err != nil {
		// 返回err，直接交给框架处理
		// 1. http: 调用方收到500，和错误消息体
		// 2. grpc-go: ...
		return nil, err
	}

	return &pb.MonitorResp{
		Errno:  0,
		Errmsg: in.Name,
	}, nil
}
