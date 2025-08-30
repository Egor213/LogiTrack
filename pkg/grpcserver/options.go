package grpcserver

import (
	"net"
	"time"
)

type Option func(*Server)

func WithPort(port string) Option {
	return func(s *Server) {
		listener, err := net.Listen("tcp", ":"+port)
		if err == nil {
			s.listener = listener
		}
	}
}

func WithShutdownTimeout(d time.Duration) Option {
	return func(s *Server) {
		s.shutdownTimeout = d
	}
}
