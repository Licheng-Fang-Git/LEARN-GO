package server

import (
	"fmt"
	"io"
	"net"
	"bytes"
	"github.com/lichengf/httpfromtcp/internal/response"
	"github.com/lichengf/httpfromtcp/internal/request"
)

type Server struct{
	closed bool
	handler Handler
}
type HandlerError struct{
	status response.StatusCode
	message string
}
type Handler func(w io.Writer, req *request.Request) *HandlerError{

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
		server.handle(conn, handler)
	}

}

func Serve(port int, handler *Handler) (*Server, error){
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) 
	if err != nil{
		return nil, err
	}
	server := &Server{ closed: false}
	
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
	}
	writer := bytes.NewBuffer([]byte{})
	s.handler(writer,request)
	response.WriteStatusLine(conn, 200)
	h := response.GetDefaultHeaders(0)
	response.WriteHeaders(conn, h)
	// out := []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello World!\n")
	// conn.Write(out)
	conn.Close()

}