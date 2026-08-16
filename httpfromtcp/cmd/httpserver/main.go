package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
    
	"github.com/lichengf/httpfromtcp/internal/request"
	"github.com/lichengf/httpfromtcp/internal/response"
	"github.com/lichengf/httpfromtcp/internal/server"
)

const port = 42069

func main() {
    server, err := server.Serve(port, func(w *response.Writer, req *request.Request) {
        h := response.GetDefaultHeaders(0)
        body := []byte{}
        switch req.RequestLine.RequestTarget {
        case "/yourproblem":
            w.WriteStatusLine(response.Status400)
            body = []byte("Your problem is not my problem\n")
        case "/myproblem":
            w.WriteStatusLine(response.Status500)
            body = []byte("Woopsie, my bad\n")
        default:
            w.WriteStatusLine(response.Status200)
            body = []byte("All good frfr\n")
        }
        h.Replace("content-length", fmt.Sprintf("%d", body))
        w.WriteHeaders(h)
        w.WriteBody([]byte("Your problem is not my problem\n"))

    })



    if err != nil {
        log.Fatalf("Error starting server: %v", err)
    }
    defer server.Close()
    log.Println("Server started on port", port)

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    log.Println("Server gracefully stopped")
}
