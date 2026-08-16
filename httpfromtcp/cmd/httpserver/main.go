package main

import (
	"io"
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
    server, err := server.Serve(port, func(w io.Writer, req *request.Request) *server.HandlerError{
        handleError := &server.HandlerError{Status: response.Status200, Message: ""}
        switch req.RequestLine.RequestTarget {
        case "/yourproblem":
            w.Write([]byte("Your problem is not my problem\n"))
            handleError.Status = response.Status400
            handleError.Message = "Your problem is not my problem\n"
            return handleError
        case "/myproblem":
            w.Write([]byte("Woopsie, my bad\n"))
            handleError.Status = response.Status500
            handleError.Message = "Woopsie, my bad\n"
            return handleError
        default:
            w.Write([]byte("All good, frfr\n"))
            return nil
        }
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
