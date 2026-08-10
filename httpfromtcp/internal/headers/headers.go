package headers

import(
	"fmt"
	"bytes"
	"strings"
	"unicode"
	"slices"
)
// https://www.boot.dev/lessons/100872fe-eb70-4822-a513-5f1ecde980f4

var InvalidHeader = fmt.Errorf("error: Header Invalid")
var HeaderDone = fmt.Errorf("error: Header Done")
var CRLF = []byte("\r\n")

type Headers struct{
	headers map[string] string
}

func NewHeaders() *Headers {
    return &Headers{
		headers: map[string]string{},
	}
}

func checkFieldNameSpacing(before int, after int) bool{
	return before == after
}

func checkFieldNameMismatch(fieldName string) bool{
	noMismatch := false
	validSpecialChars := []rune{'!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~'}
	outer:
	for _,c := range fieldName{
		switch{
		case unicode.IsDigit(c): 
		case unicode.IsLetter(c): 
		case slices.Index(validSpecialChars, c) != -1:
		default:
			noMismatch = true
			break outer
		}
	}
	return noMismatch
}
func (h *Headers) Parse(data []byte) (int, bool, error){
	n := 0
	nextHeaderIdx := 0
	done := false
	idx := bytes.Index(data[nextHeaderIdx:], CRLF)
	if idx == 0{
		return n, true, nil
	}
	
	for{
		if nextHeaderIdx >= len(data){ 
			return n, true, nil}

		idx = bytes.Index(data[nextHeaderIdx:], CRLF)
		if idx == -1{
			return n, false, InvalidHeader
		}

		if idx == 0{
			n += len(CRLF)
			done = true
			break
		}
		nextHeaderLine := data[nextHeaderIdx:nextHeaderIdx+idx]
		colonIdx := bytes.Index(nextHeaderLine, []byte(":"))
		if colonIdx == -1{
			return n, false, InvalidHeader
		}

		fieldName := strings.TrimSpace(string(nextHeaderLine[:colonIdx]))
		if !checkFieldNameSpacing(colonIdx, len(fieldName)){
			return 0, false, InvalidHeader
		}

		if checkFieldNameMismatch(fieldName){
			return 0, false, InvalidHeader
		}
		fieldName = strings.ToLower(fieldName)

		fieldValue := strings.TrimSpace(string(nextHeaderLine[colonIdx + 1:idx]))
		_,ok := h.headers[fieldName]
		if ok{
			h.headers[fieldName] += ", " + fieldValue
		}else{
			h.headers[fieldName] = fieldValue}

		nextHeaderIdx += idx+len(CRLF)
		n += idx + len(CRLF)
	}

	return n, done, nil

}