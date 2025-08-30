package grpcv1

import loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"

type LogController struct {
	loggrpc.UnimplementedLogServiceServer
}

func NewLogController() *LogController {
	return &LogController{}
}
