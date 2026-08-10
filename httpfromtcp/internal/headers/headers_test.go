package headers

import (

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadersParse(t *testing.T){
	// Test: Valid single header
	headers := NewHeaders()
	data := []byte("Host: localhost:42069   \r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, 28, n)
	assert.Equal(t, "localhost:42069", headers.headers["host"])
	assert.True(t, done)

	// Test: Invalid spacing header
	headers = NewHeaders()
	data = []byte("       Host: localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)

	// Test: Valid 2 headers with existing headers
	headers = NewHeaders()
	data = []byte("Host: localhost:42069\r\nHi: everyone\r\nSet-Person: lane-loves-go\r\nSet-Person: prime-loves-zig  \r\nSet-Person:   tj-loves-ocaml\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "lane-loves-go, prime-loves-zig, tj-loves-ocaml", headers.headers["set-person"])
	// assert.Equal(t, 39, n)
	assert.True(t, done)

	headers = NewHeaders()
	data = []byte("H©st: localhost:42069\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.Error(t, err)
	assert.Equal(t, 0, n)
	assert.False(t, done)


	// // Test: Valid done
	// headers = NewHeaders()
	// data = []byte("\r\nHost: localhost:42069\r\n\r\n")
	// n, done, err = headers.Parse(data)
	// require.NoError(t, err)
	// assert.Equal(t, 0, n)
	// assert.True(t, done)

	// //Invalid spacing header
	// headers = NewHeaders()
	// data = []byte("  Host  : localhost:42069\r\n\r\n")
	// n, done, err = headers.Parse(data)
	// require.Error(t, err)
	// assert.Equal(t, 0, n)
	// assert.False(t, done)
}