package grpcv1

import (
	loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"
	"github.com/Egor213/LogiTrack/internal/service"
	"google.golang.org/grpc"
)

func RegisterServices(services *service.Services) func(s *grpc.Server) {
	return func(s *grpc.Server) {
		loggrpc.RegisterLogServiceServer(s, NewLogController())
	}
}
