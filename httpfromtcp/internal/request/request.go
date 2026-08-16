package request

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"github.com/lichengf/httpfromtcp/internal/headers"
)

type parserState int

const (
    Initialized parserState = iota //0
	requestStateParsingHeaders //1
	requestStateParsingBody //2
    Done                    //3
	
)

type Request struct {
    RequestLine RequestLine
	state parserState
	Headers headers.Headers
	Body []byte

}

func (r *Request) parse(data []byte) (int, error){
	read := 0
	// fmt.Println(data, r.state)
	outer:
	for{
		currentData := data[read:]
		// fmt.Println(string(currentData), r.state)
		switch r.state{
		case Initialized:
			rl, n, err := parseRequestLine(currentData)
			// fmt.Println(rl, n, err)
			if err != nil{
				break outer
			}
			r.RequestLine = *rl
			read += n
			r.state = requestStateParsingHeaders

			// return n, nil
		case requestStateParsingHeaders:
			// fmt.Println("Headers Hit")
			n, ok, err := r.Headers.Parse(currentData)
			// fmt.Println(n,ok,err)
			if err != nil{
				return n, nil
			}
			read += n
			if ok{
				r.state = requestStateParsingBody
			}
			// return read, nil
			
		case requestStateParsingBody:
			lengthBody, err := r.Headers.Get("content-length")
			if err != nil{
				r.state = Done
				r.Body = nil
				return read, nil
			}

			lenb, err := strconv.Atoi(lengthBody)
			if err != nil{
				r.state = Done
				return 0, nil
			}
			
			if  len(currentData) > lenb{
				r.state = Done
				return 0, BODY_GREATER_THAN_CONTENT_LEN
			}else if lenb == len(currentData){
				r.Body = currentData
				r.state = Done
				return lenb, nil
			}else{
				return read, nil
			}

		case Done:
			return read, nil
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
var BODY_GREATER_THAN_CONTENT_LEN = fmt.Errorf("error: BODY_GREATER_THAN_CONTENT_LEN")
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
	buf := make([]byte, 1024)
	bufLen := 0
	request.Headers = *headers.NewHeaders()
	for request.state != Done{
		// fmt.Println(bufLen, readN, string(buf[:bufLen]))
		// fmt.Println(request.state)
		
		n, err := reader.Read(buf[bufLen:])
		
		if err != nil{
			// _, errH := request.Headers.Get("content-length")
			// if err == io.EOF && errH != nil{
			// 	request.Body = buf[:bufLen]
			// 	request.state = Done
			// 	return &request, nil
			// }
			return nil, err 
		}
		bufLen += n
		
		readN, err := request.parse(buf[:bufLen])

		if err != nil{
			return nil, err
		}

		copy(buf, buf[readN:bufLen])
		bufLen -= readN

		if request.state == Done {
        	break
    	}

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