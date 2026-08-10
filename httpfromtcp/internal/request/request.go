package request

import (
	"fmt"
	"io"
	"bytes"
	"github.com/lichengf/httpfromtcp/internal/headers"
)

type parserState int

const (
    Initialized parserState = iota 
	requestStateParsingHeaders
    Done                    
	
)

type Request struct {
    RequestLine RequestLine
	state parserState
	headers headers.Headers

}

func (r *Request) parse(data []byte) (int, error){
	read := 0
	outer:
	for{
		switch r.state{
		case Initialized:
			rl, n, err := parseRequestLine(data)
			if err != nil{
				break outer
			}
			r.RequestLine = *rl
			read += n
			r.state = requestStateParsingHeaders
			return n, nil
		case requestStateParsingHeaders:
			n, ok, err := r.headers.Parse(data)
			if err != nil{
				break outer 
			}
			if ok{
				r.state = Done
				return n, nil
			}
			read += n

			return read, nil

		case Done:
			return 0, nil
		}
	}
	return read, nil
}

type RequestLine struct {
    HttpVersion   string
    RequestTarget string
    Method        string
}

var ERROR_MALFORMED_REQUEST_LINE = fmt.Errorf("malformed request-line")
var UNSUPPORTED_HTTP_VERSION = fmt.Errorf("HTTP Version unsupported")
var DONE_STATE = fmt.Errorf("error: trying to read data in a done state")
var UNKNOWN_STATE = fmt.Errorf("error: unknown state")
var NO_SEPERATOR = fmt.Errorf("error: No Seperator")
var SEPERATOR = []byte("\r\n")

func validHTTP(version string) bool{
	return version == "1.1" 
}

func parseRequestLine( b []byte)(*RequestLine, int, error){
	idx := bytes.Index(b, SEPERATOR)
	if idx == -1{
		return nil, 0, NO_SEPERATOR
	}
	startLine := b[:idx]
	read := idx + len(SEPERATOR)
	r := bytes.Split(startLine, []byte(" "))

	if len(r) != 3{
		return nil, 0, ERROR_MALFORMED_REQUEST_LINE
	}
	
	if !bytes.HasPrefix(r[2], []byte("HTTP/")) {
		return nil, 0, ERROR_MALFORMED_REQUEST_LINE
	}
	
	version := string(r[2][5:])
	if !validHTTP(version) {
		return nil, 0, UNSUPPORTED_HTTP_VERSION
	}

	rl := &RequestLine{
		Method: string(r[0]),
		RequestTarget: string(r[1]),
		HttpVersion: string(r[2][5:]),
	}

	return rl, read, nil
}

func RequestFromReader(reader io.Reader) (*Request, error){
	var request Request
	buf := make([]byte, 32)
	bufLen := 0
	bytesRead := 0
	readN := 0
	// request.state = requestStateParsingHeaders
	for request.state != Done{
		fmt.Println(bufLen, readN, bytesRead, "'" + string(buf) + "'", "State:", request.state, "He")
		n, err := reader.Read(buf[bufLen:])
		if err != nil{
			return nil, err
		}

		bufLen += n
		readN, err := request.parse(buf[:bufLen])

		if err != nil{
			return nil, err
		}

		copy(buf, buf[readN:bufLen])
		bytesRead += readN
		bufLen -= readN

	}
	
	return &request, nil
	
}


func main(){
	   // Test: Good GET Request line
	//    reader := &chunkReader{
	// 	data:            "GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n",
	// 	numBytesPerRead: 3,
	// }
	// r, err := RequestFromReader(reader)
	// fmt.Println(r, err)
}