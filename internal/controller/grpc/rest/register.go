package grpcrest

import (
	"context"

	loggrpc "github.com/Egor213/LogiTrack/internal/controller/grpc/v1/loggrpc_gen"
	gw "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func RegisterServices(ctx context.Context, handler *gw.ServeMux, grpcPort string) error {
	err := loggrpc.RegisterLogServiceHandlerFromEndpoint(
		ctx,
		handler,
		"localhost:"+grpcPort,
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	return err
}
