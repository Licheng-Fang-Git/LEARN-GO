package request

import (
	"errors"
	"fmt"
	"io"

	"strings"
)

type Request struct {
    RequestLine RequestLine
}

type RequestLine struct {
    HttpVersion   string
    RequestTarget string
    Method        string
}

var ERROR_MALFORMED_REQUEST_LINE = fmt.Errorf("malformed request-line")
var UNSUPPORTED_HTTP_VERSION = fmt.Errorf("HTTP Version unsupported")

func validHTTP(version *RequestLine) bool{
	return version.HttpVersion == "1.1" 
}

func RequestFromReader(reader io.Reader) (*Request, error){
	b, err := io.ReadAll(reader)
	var request Request
	if err != nil{
		return nil, errors.Join(fmt.Errorf("unable to read io.ReadAll"), err)
	}
	requestLine := strings.Split(string(b), "\r\n")[0]
	err = parseRequestLine(requestLine, &request)

	return &request, err
	
}

func parseRequestLine( reuqestLine string, request *Request)(error){
	r := strings.Split(reuqestLine, " ")

	if len(r) != 3{
		return ERROR_MALFORMED_REQUEST_LINE
	}

	request.RequestLine.Method = r[0]
	request.RequestLine.RequestTarget = r[1]
	request.RequestLine.HttpVersion = r[2][5:]

	if !validHTTP(&request.RequestLine){
		return UNSUPPORTED_HTTP_VERSION
	}

	return nil
}

func main(){
	// r, err := RequestFromReader(strings.NewReader("GET / HTTP/1.1\r\nHost: localhost:42069\r\nUser-Agent: curl/7.81.0\r\nAccept: */*\r\n\r\n"))
	// if err != nil{
	// 	log.Fatal(err)
	// }
	// fmt.Println(r.RequestLine.HttpVersion)
}