package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"github.com/sevaergdm/httpfromtcp/internal/request"
	"github.com/sevaergdm/httpfromtcp/internal/response"
)

type Handler func(w *response.Writer, req *request.Request)

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

type Server struct {
	Listener net.Listener
	Closed   atomic.Bool
	Handler  Handler
}

func (s *Server) listen() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if s.Closed.Load() {
				return
			}
			log.Printf("accept error: %v", err)
			continue
		}

		go s.handle(conn)
	}
}

func (s *Server) Close() error {
	s.Closed.Store(true)
	if s.Listener != nil {
		return s.Listener.Close()
	}
	return nil
}

func (s *Server) handle(c net.Conn) {
	defer c.Close()
	r, err := request.RequestFromReader(c)
	if err != nil {
		return
	}

	w := response.NewWriter(c)
	s.Handler(w, r)
}

func Serve(handler Handler, port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	server := &Server{
		Listener: listener,
		Handler:  handler,
	}

	go server.listen()

	return server, nil
}
