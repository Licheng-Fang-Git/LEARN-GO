package headers

import (

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadersParse(t *testing.T){
	// Test: Valid single header
	headers := NewHeaders()
	data := []byte("Host: localhost:42069\r\n\r\n")
	n, done, err := headers.Parse(data)
	require.NoError(t, err)
	require.NotNil(t, headers)
	assert.Equal(t, 25, n)
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
	data = []byte("Host: localhost:42069\r\nHi: everyone\r\n\r\n")
	n, done, err = headers.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "everyone", headers.headers["hi"])
	assert.Equal(t, 39, n)
	assert.True(t, done)


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