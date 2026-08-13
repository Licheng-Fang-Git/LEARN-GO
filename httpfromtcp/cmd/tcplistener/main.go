package main

import (
	"fmt"
	"log"
	"github.com/lichengf/httpfromtcp/internal/request"
	"net"
)

func main(){
	listener, err := net.Listen("tcp", ":42069")

	if err != nil{
		log.Fatal(err)
	}

	defer listener.Close()
	for{
		conn, err := listener.Accept()
		
		if err != nil{
			log.Fatal(err)
		}

		r, err := request.RequestFromReader(conn)

		fmt.Println("Request Line:")
		fmt.Println("- Method:", r.RequestLine.Method)
		fmt.Println("- Target:", r.RequestLine.RequestTarget)
		fmt.Println("- Version:", r.RequestLine.HttpVersion)
		fmt.Println("Headers: ")
		
		for key,val := range r.Headers.Headers{
			fmt.Printf("- %s: %s\n", key, val)
		}
		fmt.Println("Body:")
		fmt.Println(string(r.Body))

	}
	
}