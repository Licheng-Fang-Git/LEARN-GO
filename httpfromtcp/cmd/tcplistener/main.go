package main

import (
	"fmt"
	"io"
	"log"

	"strings"
	"net"
)

func getLinesChannel(f io.ReadCloser) <-chan string{
	out := make(chan string, 1)
	
	go func(){
		defer f.Close()
		defer close(out)

		currLine := ""
		for{
			data := make([]byte, 8)
			n, err := f.Read(data)
	
			if err != nil{
				break
			}
			
			data_string := string(data[:n])
			newLine := strings.Index(data_string, "\n")
			if newLine == -1{
				currLine = currLine + data_string
			}else{
				currLine += data_string[:newLine]
				out <- currLine
				currLine = data_string[newLine+1:]
			}
			
		}
		out <- currLine
		
	}()
	return out
}

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

		output := getLinesChannel(conn)
		for line := range output{
			fmt.Println(line)
		}
		
	}
	
}