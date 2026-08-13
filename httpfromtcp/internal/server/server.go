package server

import(
	"net"

)

type Server struct{
	
}

func Serve(port int) (*Server, error){
	server := &Server{}
	
	return server, nil
}

func (s *Server) Close() error{
	return nil
}

func (s *Server) listen(){}

func (s *Server) handle(conn net.Conn){}