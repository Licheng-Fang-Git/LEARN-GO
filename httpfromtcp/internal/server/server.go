package server

import (
	"fmt"
	"io"
	"net"
	"bytes"
	"github.com/lichengf/httpfromtcp/internal/response"
	"github.com/lichengf/httpfromtcp/internal/request"
)
type Handler func(w io.Writer, req *request.Request) *HandlerError

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
	h := response.GetDefaultHeaders(0)
	request, err := request.RequestFromReader(conn)
	
	if err != nil{
		response.WriteStatusLine(conn, 400)
		response.WriteHeaders(conn, h)
		return 
	}
	writer := bytes.NewBuffer([]byte{})
	handlerError := s.handler(writer, request)
	body := writer.Bytes()
	if handlerError != nil{
		response.WriteStatusLine(conn, handlerError.Status)
		h.Replace("content-length", fmt.Sprintf("%d", len(body)))
		response.WriteHeaders(conn, h)
		conn.Write([]byte(handlerError.Message))
		return 
	}
	fmt.Print(string(body))
	response.WriteStatusLine(conn, 200)
	h.Replace("content-length", fmt.Sprintf("%d", len(body)))
	response.WriteHeaders(conn, h)
	conn.Write(body)
}