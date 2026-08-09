package main

import (
	"fmt"
	"log"

	"net"
	"request/tests"
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

		output := (conn)
		for line := range output{
			fmt.Println(line)
		}
		
	}
	
}