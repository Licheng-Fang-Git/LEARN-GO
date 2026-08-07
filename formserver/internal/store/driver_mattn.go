//go:build cgosqlite

package store

// Optional driver, selected with `-tags cgosqlite`:
// github.com/mattn/go-sqlite3 wraps the C SQLite library. It requires CGO and a
// C compiler (gcc/clang), but it's the most battle-tested driver in the wild.
import _ "github.com/mattn/go-sqlite3"

const driverName = "sqlite3"
