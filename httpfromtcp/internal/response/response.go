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

type WriterState int

const(
	WriteStatusLine = iota
	WriteHeaders
	WriteBody
)

type Writer struct{
	writer io.Writer
}

func NewWriter(writer io.Writer) *Writer{
	return &Writer{writer: writer}
}
func (w *Writer) WriteStatusLine(statusCode StatusCode) error{
	statusLine := []byte{}
	switch statusCode{
	case Status200:
		statusLine = []byte("HTTP/1.1 200 OK\r\n")
	case Status400:
		statusLine = []byte("HTTP/1.1 400 Bad Request\r\n")
	case Status500:
		statusLine = []byte("HTTP/1.1 500 Internal Server Error\r\n")
	default:
		statusLine = []byte("")
	}
	_, err := w.writer.Write(statusLine)
	return err
}
func (w *Writer) WriteHeaders(headers headers.Headers) error{
	b := []byte{}
	for key,val := range headers.Headers{
		b = append(b, []byte(key + ": " + val + "\r\n")...)
	}
	b = append(b, ([]byte("\r\n"))...)
	_, err := w.writer.Write(b)

	return err
}
func (w *Writer) WriteBody(p []byte) (int, error){
	n, err := w.writer.Write(p)
	return n, err
}

// func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
// 	switch statusCode{
// 	case Status200:
// 		out := []byte("HTTP/1.1 200 OK\r\n")
// 		w.Write(out)
// 	case Status400:
// 		out := []byte("HTTP/1.1 400 Bad Request\r\n")
// 		w.Write(out)
// 	case Status500:
// 		out := []byte("HTTP/1.1 500 Internal Server Error\r\n")
// 		w.Write(out)
// 	default:
// 		w.Write([]byte(""))
// 	}
// 	return nil
// }

func GetDefaultHeaders(contentLen int) headers.Headers{
	h := headers.NewHeaders()
	h.Headers["content-length"] = strconv.Itoa(contentLen)
	h.Headers["connection"] = "close"
	h.Headers["content-type"] = "text/plain"
	return *h
}

// func WriteHeaders(w io.Writer, headers headers.Headers) error{
// 	for key,val := range headers.Headers{
// 		out := []byte(key + ": " + val + "\r\n")
// 		w.Write(out)
// 	}
// 	w.Write([]byte("\r\n"))
// 	return nil
// }

