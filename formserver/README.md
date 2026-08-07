# FormServer

A small, standalone web server written in Go that:

- **Hosts forms generated from templates** — each form is a JSON file in `forms/`, rendered by one generic HTML template.
- **Collects submissions** over HTTP from browsers *or* API clients (JSON).
- **Stores them in SQLite** via full CRUD (Create, Read, Update, Delete).
- **Reports the outcome** back to the caller — an HTML result page for browsers, a JSON response for API clients — and logs every request server-side.

Built almost entirely on the Go **standard library** (`net/http`, `html/template`, `database/sql`, `encoding/json`). The only external dependency is a SQLite driver, because Go's `database/sql` is an interface that needs a concrete driver plugged in.

---

## Quick start

```bash
cd formserver

# Fetch the pure-Go SQLite driver (first run only).
go mod download          # or: go mod tidy

# Run it.
go run .
```

Then open <http://localhost:8080>.

Flags:

```bash
go run . -addr :9000 -db my.db -forms forms -templates templates -static static
```

| Flag         | Default        | Meaning                              |
|--------------|----------------|--------------------------------------|
| `-addr`      | `:8080`        | HTTP listen address                  |
| `-db`        | `formserver.db`| SQLite file (created if missing)     |
| `-forms`     | `forms`        | Directory of form-definition JSON    |
| `-templates` | `templates`    | Directory of HTML templates          |
| `-static`    | `static`       | Directory of static assets           |

---

## The two SQLite drivers

`database/sql` needs a driver. This project supports two, chosen at build time:

| Driver | Build command | CGO / gcc? | Notes |
|--------|---------------|------------|-------|
| **modernc.org/sqlite** (default) | `go build` | **No** | Pure Go. Nothing to install, builds anywhere. |
| **github.com/mattn/go-sqlite3** | `go build -tags cgosqlite` | **Yes** | Wraps C SQLite; needs a C compiler. Most widely used. |

The default (`go build`) uses the pure-Go driver — recommended. If you want the
CGO driver, make sure it's available and build with the tag:

```bash
go get github.com/mattn/go-sqlite3   # if not already in go.mod
CGO_ENABLED=1 go build -tags cgosqlite -o formserver .
```

Only the blank import and driver-name string differ between the two
(`internal/store/driver_modernc.go` vs `internal/store/driver_mattn.go`); all
the CRUD logic is shared.

---

## HTTP API

Every page also speaks JSON. Ask for it with `Accept: application/json` or `?format=json`.

| Method & path | Purpose | CRUD |
|---|---|---|
| `GET /` | List available forms | — |
| `GET /forms/{id}` | Render a form | — |
| `POST /forms/{id}/submit` | Submit a form | **C** |
| `GET /forms/{id}/submissions` | List responses for a form | **R** |
| `GET /submissions/{id}` | View one response | **R** |
| `PUT /submissions/{id}` | Replace a response (JSON) | **U** |
| `DELETE /submissions/{id}` | Delete a response | **D** |
| `POST /submissions/{id}/delete` | Delete (browser-form fallback) | **D** |
| `GET /healthz` | Health check | — |

### Examples

```bash
# Create (API / JSON)
curl -X POST localhost:8080/forms/geo/submit \
  -H 'Content-Type: application/json' \
  -d '{"label":"Warehouse A","latitude":40.7128,"longitude":-74.006,"note":"hub"}'

# Read
curl 'localhost:8080/forms/geo/submissions?format=json'
curl 'localhost:8080/submissions/1?format=json'

# Update
curl -X PUT localhost:8080/submissions/1 \
  -H 'Content-Type: application/json' \
  -d '{"label":"Warehouse B","latitude":41.0,"longitude":-73.0,"note":"moved"}'

# Delete (send the JSON Accept header to get a JSON reply instead of a redirect)
curl -X DELETE -H 'Accept: application/json' localhost:8080/submissions/1
```

---

## Add a new form (no code, no rebuild)

Drop a JSON file in `forms/` and restart:

```json
{
  "id": "rsvp",
  "title": "Event RSVP",
  "description": "Let us know if you're coming.",
  "submitLabel": "RSVP",
  "fields": [
    { "name": "name",    "label": "Name",     "type": "text",   "required": true },
    { "name": "guests",  "label": "Guests",   "type": "number" },
    { "name": "attending","label": "Attending?","type": "radio", "required": true, "options": ["Yes","No"] }
  ]
}
```

Supported field types: `text`, `email`, `password`, `number`, `tel`, `url`,
`date`, `textarea`, `select`, `checkbox`, `radio`. `select`/`radio` require an
`options` array.

The four bundled forms cover the example use cases: `signup` (login
credentials), `survey` (survey results), `geo` (geographic coordinates), and
`contact` (asynchronous messages).

---

## Project layout

```
formserver/
├── main.go                      # wiring: load forms → open DB → serve
├── forms/                       # form definitions (one JSON per form)
├── templates/                   # html/template files (shared header/footer + pages)
├── static/style.css             # styling
└── internal/
    ├── forms/forms.go           # form model, loader, submission validation
    ├── store/store.go           # SQLite CRUD via database/sql
    ├── store/driver_modernc.go  # default pure-Go driver (build: default)
    ├── store/driver_mattn.go    # optional CGO driver (build: -tags cgosqlite)
    └── web/server.go            # routing, handlers, content negotiation, rendering
```

See **TUTORIAL.md** for a full walkthrough of how it all fits together and how to extend it.
