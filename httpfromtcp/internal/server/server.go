package server

import (
	"fmt"
	"net"
)

type Server struct{
	
}

func runServer(listener net.Listener, server *Server){
	for{
		conn, err := listener.Accept()
		if err != nil{
			return 
		}
	}

}

func Serve(port int) (*Server, error){
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) 
	if err != nil{
		return nil, err
	}
	server := &Server{}
	
	go runServer()
	return server, nil
}

func (s *Server) Close() error{
	return nil
}

func (s *Server) listen(){}

func (s *Server) handle(conn net.Conn){}