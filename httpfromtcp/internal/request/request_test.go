package request

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
    data            string
    numBytesPerRead int
    pos             int
}
var Thinking = fmt.Errorf("Hi")
// Read reads up to len(p) or numBytesPerRead bytes from the string per call
// its useful for simulating reading a variable number of bytes per chunk from a network connection
func (cr *chunkReader) Read(p []byte) (n int, err error) {
    if cr.pos >= len(cr.data) {
        return 0, io.EOF
    }
    endIndex := cr.pos + cr.numBytesPerRead
    if endIndex > len(cr.data) {
        endIndex = len(cr.data)
    }
    n = copy(p, cr.data[cr.pos:endIndex])
    cr.pos += n
    return n, nil
}

func TestRequestLineParse(t *testing.T) {
	// Test: Standard Body
	reader := &chunkReader{
		data: "POST /coffee HTTP/1.1\r\n"+
		"Host: localhost:42069\r\n"+
		"User-Agent: curl/8.4.0\r\n" +
		"Accept: */*\r\n" +
		"Content-Type: application/json\r\n"+
		"Content-Length: 31\r\n\r\n"+
		"{type: dark mode, size: medium}",
		numBytesPerRead: 168,
	}

	r, err := RequestFromReader(reader)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, "{type: dark mode, size: medium}", string(r.Body))

	// // Test: Body shorter than reported content length
	// reader = &chunkReader{
	// 	data: "POST /submit HTTP/1.1\r\n" +
	// 		"Host: localhost:42069\r\n" +
	// 		"Content-Length: 20\r\n" +
	// 		"\r\n" +
	// 		"partial content",
	// 	numBytesPerRead: 3,
	// }
	// r, err = RequestFromReader(reader)
	// fmt.Println(err)
	// require.Error(t, err)
}