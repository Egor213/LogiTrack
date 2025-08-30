package grpcserver

import (
	"net"
	"time"

	errorsUtils "github.com/Egor213/LogiTrack/pkg/errors"

	"google.golang.org/grpc"
)

const (
	defaultAddr            = ":50051"
	defaultShutdownTimeout = 3 * time.Second
)

type Server struct {
	Server          *grpc.Server
	listener        net.Listener
	notify          chan error
	shutdownTimeout time.Duration
}

func New(register func(*grpc.Server), opts ...Option) (*Server, error) {
	listener, err := net.Listen("tcp", defaultAddr)

	if err != nil {
		return nil, errorsUtils.WrapPathErr(err)
	}

	grpcServer := grpc.NewServer()

	register(grpcServer)

	s := &Server{
		Server:          grpcServer,
		listener:        listener,
		notify:          make(chan error, 1),
		shutdownTimeout: defaultShutdownTimeout,
	}

	for _, opt := range opts {
		opt(s)
	}

	s.start()

	return s, nil
}

func (s *Server) start() {
	go func() {
		s.notify <- s.Server.Serve(s.listener)
		close(s.notify)
	}()
}

func (s *Server) Notify() <-chan error {
	return s.notify
}

func (s *Server) Shutdown() {
	done := make(chan any)
	go func() {
		s.Server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(s.shutdownTimeout):
		s.Server.Stop()
	}
}
