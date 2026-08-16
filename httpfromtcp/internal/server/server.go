package server

import (
	"fmt"
	"io"
	"net"
	
	"github.com/lichengf/httpfromtcp/internal/request"
	"github.com/lichengf/httpfromtcp/internal/response"
)

type Handler func(w *response.Writer, req *request.Request)

type Server struct{
	closed bool
	handler Handler
}
type HandlerError struct{
	Status response.StatusCode
	Message string
}


func runServer(listener net.Listener, server *Server){
	for{
		conn, err := listener.Accept()
		if server.closed{
			return
		}
		if err != nil{
			return 
		}
		server.handle(conn)
	}

}

func Serve(port int, handler Handler) (*Server, error){
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) 
	if err != nil{
		return nil, err
	}
	server := &Server{ closed: false, handler: handler }
	
	go runServer(listener, server)
	return server, nil
}

func (s *Server) Close(){
	s.closed = true
}

func (s *Server) handle(conn io.ReadWriteCloser){
	defer conn.Close()
	request, err := request.RequestFromReader(conn)
	responseWriter := response.NewWriter(conn)

	if err != nil{
		responseWriter.WriteStatusLine(response.Status400)
		headers := response.GetDefaultHeaders(0)
		responseWriter.WriteHeaders(headers)
		return 
	}
	
	s.handler(responseWriter, request)
}