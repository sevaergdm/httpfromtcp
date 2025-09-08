package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"github.com/sevaergdm/httpfromtcp/internal/response"
)

type Server struct {
	Listener net.Listener
	Closed   atomic.Bool
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
	err := response.WriteStatusLine(c, response.OK)
	if err != nil {
		return 
	}
	headers := response.GetDefaultHeaders(0)
	err = response.WriteHeaders(c, headers)
	if err != nil {
		return
	}
}

func Serve(port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	server := &Server{
		Listener: listener,
	}

	go server.listen()

	return server, nil
}
