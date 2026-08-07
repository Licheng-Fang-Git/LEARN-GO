//go:build !cgosqlite

package store

// Default driver: modernc.org/sqlite is a pure-Go SQLite implementation.
// It needs no C compiler and no CGO, so `go build` works everywhere.
import _ "modernc.org/sqlite"

const driverName = "sqlite"
