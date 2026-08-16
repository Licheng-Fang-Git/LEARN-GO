package response

import(
	"io"
	"strconv"
	"github.com/lichengf/httpfromtcp/internal/headers"
)

type StatusCode int

const(
	Status200 StatusCode = 200
	Status400 StatusCode = 400
	Status500 StatusCode = 500
)

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	switch statusCode{
	case Status200:
		out := []byte("HTTP/1.1 200 OK\r\n")
		w.Write(out)
	case Status400:
		out := []byte("HTTP/1.1 400 Bad Request\r\n")
		w.Write(out)
	case Status500:
		out := []byte("HTTP/1.1 500 Internal Server Error\r\n")
		w.Write(out)
	default:
		w.Write([]byte(""))
	}
	return nil
}

func GetDefaultHeaders(contentLen int) headers.Headers{
	h := headers.NewHeaders()
	h.Headers["content-length"] = strconv.Itoa(contentLen)
	h.Headers["connection"] = "close"
	h.Headers["content-type"] = "text/plain"
	return *h
}

func WriteHeaders(w io.Writer, headers headers.Headers) error{
	for key,val := range headers.Headers{
		out := []byte(key + ": " + val + "\r\n")
		w.Write(out)
	}
	w.Write([]byte("\r\n"))
	return nil
}

