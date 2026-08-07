# Building FormServer — a template-driven form server in Go

This tutorial walks through a complete, working Go web server that hosts forms,
collects submissions, stores them in a database with CRUD operations, and reports
the outcome back to the client. We lean on the **standard library** — `net/http`,
`html/template`, `database/sql`, `encoding/json` — so you can see the machinery
without a framework hiding it.

By the end you'll understand every file in the project, the full lifecycle of a
request, and where to extend it (email notifications, auth, new field types).

> **Prerequisites:** Go 1.22 or newer (we use the method-aware `ServeMux` added
> in 1.22) and basic familiarity with Go syntax and HTTP.

---

## 1. The core idea: forms are data, not code

Most tutorials hard-code one form and one handler. That doesn't scale: every new
form means new HTML and new Go. Instead we treat **a form as a piece of data** —
a title plus an ordered list of typed fields — described in JSON:

```json
{
  "id": "survey",
  "title": "Customer satisfaction survey",
  "fields": [
    { "name": "rating", "label": "Overall rating", "type": "select", "required": true,
      "options": ["5","4","3","2","1"] },
    { "name": "comments", "label": "Anything else?", "type": "textarea" }
  ]
}
```

One generic HTML template renders *any* such definition, and one database table
stores responses for *any* form. Adding a form is dropping a JSON file — no code,
no recompile. This single decision shapes the whole architecture:

```
JSON form defs ──► forms.Registry ──► generic HTML template ──► browser
                                          │
                          submission ─────┘
                                          ▼
                                    store (SQLite)  ◄──► JSON/HTML responses
```

The project is split into three internal packages, each with one job:

| Package | Responsibility |
|---|---|
| `internal/forms` | What a form *is*; load definitions; validate submissions |
| `internal/store` | Where responses *live*; CRUD over SQLite |
| `internal/web`   | The HTTP layer: routing, parsing, rendering, content negotiation |

`main.go` wires them together. Let's build each piece.

---

## 2. Modeling a form (`internal/forms/forms.go`)

A form is two structs. A `Field` is one input; a `Definition` is the whole form:

```go
type Field struct {
    Name        string   `json:"name"`        // the POST key
    Label       string   `json:"label"`       // shown to the user
    Type        string   `json:"type"`        // text, email, number, select, ...
    Required    bool     `json:"required"`
    Placeholder string   `json:"placeholder,omitempty"`
    Help        string   `json:"help,omitempty"`
    Options     []string `json:"options,omitempty"` // for select/radio
}

type Definition struct {
    ID          string  `json:"id"`
    Title       string  `json:"title"`
    Description string  `json:"description"`
    SubmitLabel string  `json:"submitLabel,omitempty"`
    Fields      []Field `json:"fields"`
}
```

The struct tags map JSON keys to fields — that's all it takes for
`encoding/json` to parse our definition files.

### Loading definitions from disk

`LoadDir` reads every `*.json` file in a directory into a `Registry` (a map keyed
by form ID). Two details matter for robustness:

```go
dec := json.NewDecoder(strings.NewReader(string(b)))
dec.DisallowUnknownFields()  // a typo'd key is an error, not a silent no-op
...
if err := d.validate(); err != nil { ... }  // fail fast at startup
```

`DisallowUnknownFields` turns `"lable"` (a typo) into a startup error instead of
a field that silently never appears. And `validate()` runs once at boot — a
missing `id`, a duplicate field name, or a `select` with no `options` stops the
server from starting rather than blowing up on the first request. **Push errors
as early as possible.**

### Validating what users submit

The same package owns submission validation, because the form definition is the
authority on what's acceptable. `Clean` takes the raw submitted values and
returns a cleaned map plus a list of human-readable errors:

```go
func (d *Definition) Clean(values url.Values) (map[string]any, []string) {
    out := map[string]any{}
    var errs []string
    for _, f := range d.Fields {
        if f.Type == "checkbox" {
            out[f.Name] = values.Has(f.Name) // present == checked
            continue
        }
        v := strings.TrimSpace(values.Get(f.Name))
        if v == "" {
            if f.Required { errs = append(errs, f.Label+" is required.") }
            out[f.Name] = ""
            continue
        }
        switch f.Type {
        case "number":
            n, err := strconv.ParseFloat(v, 64)
            if err != nil { errs = append(errs, f.Label+" must be a number."); continue }
            out[f.Name] = n            // stored as a real number, not "40.7"
        case "select", "radio":
            // reject anything not in the declared options
        ...
        }
    }
    return out, errs
}
```

Two things worth noting: numbers are coerced to real numeric types (so they're
stored and returned as JSON numbers, not strings), and any submitted key that
*isn't* a defined field is dropped — a client can't smuggle extra columns in.

---

## 3. Persisting responses (`internal/store/store.go`)

Here's the one place we step outside the standard library. `database/sql` is an
*interface*; it needs a concrete driver. We use a SQLite driver (more on the
two choices in §7). The interface, though, is pure stdlib.

### One table for every form

Because forms are dynamic, we don't create a column per field. We store each
submission's values as a **JSON blob** in a single `data` column:

```sql
CREATE TABLE IF NOT EXISTS submissions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    form_id    TEXT NOT NULL,
    data       TEXT NOT NULL,   -- JSON: {"username":"jane","newsletter":true}
    created_at TEXT NOT NULL,   -- RFC3339 text (portable across drivers)
    updated_at TEXT NOT NULL
);
```

This "schemaless column in a relational table" pattern lets one table serve any
form. If you later need to query *inside* the data, SQLite can index JSON
expressions — but for collect-and-review this is ideal.

> **Why timestamps as TEXT?** Different drivers marshal `time.Time` differently.
> Formatting to RFC3339 text ourselves makes the round-trip identical no matter
> which driver is compiled in.

### The CRUD methods

Each CRUD operation is one method with a parameterized query — **always use `?`
placeholders, never string concatenation**, which is what keeps SQL injection
off the table:

```go
func (s *Store) Create(ctx context.Context, formID string, data map[string]any) (*Submission, error) {
    blob, _ := json.Marshal(data)
    now := time.Now().UTC()
    res, err := s.db.ExecContext(ctx,
        `INSERT INTO submissions(form_id,data,created_at,updated_at) VALUES(?,?,?,?)`,
        formID, string(blob), now.Format(tsLayout), now.Format(tsLayout))
    if err != nil { return nil, err }
    id, _ := res.LastInsertId()
    return &Submission{ID: id, FormID: formID, Data: data, CreatedAt: now, UpdatedAt: now}, nil
}
```

`Get`, `ListByForm`, `Update`, and `Delete` follow the same shape. `Update` and
`Delete` check `RowsAffected()` and return a sentinel `ErrNotFound` when nothing
matched, so the web layer can turn that into a clean 404. Reads run through a
tiny `scan` helper (satisfied by both `*sql.Row` and `*sql.Rows`) that
unmarshals the JSON blob back into a `map[string]any` and parses the timestamps.

We also cap the connection pool at one:

```go
db.SetMaxOpenConns(1) // SQLite has a single writer; serialize to avoid "database is locked"
```

---

## 4. The HTTP layer (`internal/web/server.go`)

This is where forms and storage meet the network.

### Routing with the method-aware mux (Go 1.22+)

Since Go 1.22, `http.ServeMux` understands method and path-parameter patterns —
no third-party router needed:

```go
mux.HandleFunc("GET /forms/{id}",             a.handleShowForm)
mux.HandleFunc("POST /forms/{id}/submit",     a.handleSubmit)
mux.HandleFunc("GET /forms/{id}/submissions", a.handleList)
mux.HandleFunc("GET /submissions/{id}",       a.handleShowSubmission)
mux.HandleFunc("PUT /submissions/{id}",       a.handleUpdate)
mux.HandleFunc("DELETE /submissions/{id}",    a.handleDelete)
```

Inside a handler, `r.PathValue("id")` pulls the matched segment. The RESTful
verbs (`PUT`, `DELETE`) serve API clients; a `POST /submissions/{id}/delete`
alias exists because HTML forms can only `GET` or `POST`.

### One server, two audiences: content negotiation

The same endpoints serve a browser (wants HTML) and a script (wants JSON). A
tiny helper decides which:

```go
func wantsJSON(r *http.Request) bool {
    if r.URL.Query().Get("format") == "json" { return true }
    return strings.Contains(r.Header.Get("Accept"), "application/json")
}
```

So `handleSubmit`, after storing a submission, branches on it — this is the
"**notify the user of the outcome**" requirement in action:

```go
sub, err := a.store.Create(r.Context(), def.ID, data)
if err != nil { a.fail(w, r, 500, "Could not save your response."); return }

a.log.Printf("stored submission #%d for form %q", sub.ID, def.ID) // server-side notice
if wantsJSON(r) {
    respondJSON(w, http.StatusCreated, map[string]any{"status": "ok", "submission": sub})
    return
}
// browser: render a friendly result page showing exactly what was stored
a.renderStatus(w, http.StatusCreated, "result.html", resultView{ ... })
```

A browser gets a `201` result page; an API client gets `201 {"status":"ok",...}`;
the server logs the outcome either way.

### Accepting both form posts and JSON bodies

`parseValues` normalizes both encodings into `url.Values` so validation doesn't
care where the data came from:

```go
func parseValues(r *http.Request) (url.Values, error) {
    if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
        var m map[string]any
        json.NewDecoder(r.Body).Decode(&m)
        v := url.Values{}
        for k, raw := range m { /* stringify; bool true => "on" for checkboxes */ }
        return v, nil
    }
    r.ParseForm()
    return r.PostForm, nil
}
```

### Rendering safely

Templates are parsed once at startup with `ParseGlob`. Every page is rendered
**into a buffer first**, so if a template errors we send a clean `500` instead of
a half-written page:

```go
func (a *App) renderStatus(w http.ResponseWriter, code int, name string, data any) {
    var buf bytes.Buffer
    if err := a.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
        a.log.Printf("template %s: %v", name, err)
        http.Error(w, "Internal template error", 500)
        return
    }
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(code)
    buf.WriteTo(w)
}
```

Because we use `html/template` (not `text/template`), all interpolated values are
**context-aware auto-escaped** — user input rendered into a page can't inject
script. That's XSS protection for free.

### Request logging middleware

A small wrapper records method, path, status, and duration for every request. A
`statusRecorder` embeds `http.ResponseWriter` and intercepts `WriteHeader` to
capture the status code:

```go
type statusRecorder struct { http.ResponseWriter; code int }
func (s *statusRecorder) WriteHeader(c int) { s.code = c; s.ResponseWriter.WriteHeader(c) }
```

---

## 5. The generic template (`templates/form.html`)

This one template renders every form. It ranges over the fields and switches on
type. The trick is capturing the field in a variable (`$field`) before entering
the inner `range` over options, where the dot (`.`) would otherwise change:

```html
{{range .Def.Fields}}
{{$field := .}}
{{$val := index $.Values $field.Name}}   {{/* prior value, for re-rendering */}}
<div class="field">
  {{if eq $field.Type "textarea"}}
    <textarea name="{{$field.Name}}">{{$val}}</textarea>
  {{else if eq $field.Type "select"}}
    <select name="{{$field.Name}}">
      {{range $field.Options}}<option{{if eq . $val}} selected{{end}}>{{.}}</option>{{end}}
    </select>
  {{else if eq $field.Type "checkbox"}}
    <input type="checkbox" name="{{$field.Name}}"{{if $val}} checked{{end}}>
  {{else}}
    <input type="{{$field.Type}}" name="{{$field.Name}}" value="{{$val}}"
           {{if $field.Required}}required{{end}}>
  {{end}}
</div>
{{end}}
```

Passing `.Values` in means that when validation fails we re-render the *same*
form with the user's input preserved and the error list on top — no retyping.
Shared chrome lives in `layout.html` as `{{define "header"}}` / `{{define
"footer"}}` blocks that every page includes.

---

## 6. Wiring it together (`main.go`)

`main` is a linear, readable startup sequence:

```go
registry, err := forms.LoadDir(*formsDir)          // 1. load form templates
st, err := store.Open(*dbPath); st.Migrate(ctx)    // 2. open + migrate DB
app, err := web.New(registry, st, *tmplDir, ...)   // 3. build the HTTP app
srv := &http.Server{Addr: *addr, Handler: app.Routes()}
go srv.ListenAndServe()                             // 4. serve...
<-stop; srv.Shutdown(ctx)                           // ...with graceful shutdown
```

Everything is a `flag`, so ports, the DB path, and the asset directories are
configurable without editing code. The graceful shutdown (listening for
`SIGINT`/`SIGTERM`, then `srv.Shutdown`) lets in-flight requests finish before
the process exits — important once real clients depend on you.

---

## 7. The one dependency: choosing a SQLite driver

`database/sql` needs a driver registered via a blank import. There's no SQLite
driver *in* the standard library, so this is the single external piece — and we
make it swappable with a **build tag** so the choice is a build-time flag, not a
code change.

`driver_modernc.go` (compiled by default):

```go
//go:build !cgosqlite
package store
import _ "modernc.org/sqlite"
const driverName = "sqlite"
```

`driver_mattn.go` (compiled with `-tags cgosqlite`):

```go
//go:build cgosqlite
package store
import _ "github.com/mattn/go-sqlite3"
const driverName = "sqlite3"
```

`store.go` just calls `sql.Open(driverName, path)`. The default is
`modernc.org/sqlite` — **pure Go, no CGO, no C compiler**, so `go build` works
anywhere. The `mattn` driver wraps C SQLite (needs gcc) and is the most
battle-tested option; build it with `go build -tags cgosqlite`. Same CRUD code
either way.

---

## 8. Try it

```bash
go mod download && go run .
```

```bash
# Create (JSON API)
curl -X POST localhost:8080/forms/geo/submit -H 'Content-Type: application/json' \
  -d '{"label":"Warehouse A","latitude":40.7128,"longitude":-74.006}'
# → 201 {"status":"ok","submission":{"id":1,...}}

# Read, update, delete
curl 'localhost:8080/submissions/1?format=json'
curl -X PUT localhost:8080/submissions/1 -H 'Content-Type: application/json' \
  -d '{"label":"Warehouse B","latitude":41,"longitude":-73}'
curl -X DELETE -H 'Accept: application/json' localhost:8080/submissions/1
```

Or just open <http://localhost:8080> and click through the four bundled forms.

---

## 9. Where to take it next

The architecture leaves clean seams for each of these:

- **Email / async notification.** In `handleSubmit`, after `store.Create`
  succeeds, fire an email with `net/smtp` (do it in a goroutine, or push to a
  channel a worker drains, so the response isn't blocked on SMTP). This realizes
  the "asynchronous communication" use case end-to-end.
- **Authentication.** Add a middleware in `Routes()` that checks a session cookie
  or API token before the submissions-management endpoints.
- **New field types.** Add a `case` in `Clean` (validation) and a branch in
  `form.html` (rendering). Nothing else changes.
- **CSV export.** A `GET /forms/{id}/export` handler that streams
  `ListByForm` results through `encoding/csv`.
- **Rate limiting / CSRF.** Wrap the mux with more middleware; add a hidden token
  field the template emits and the handler checks.
- **Postgres.** Swap the driver import and DSN; the `database/sql` code barely
  changes (mind the `?` vs `$1` placeholder style).

The throughline: keep *what a form is* (data), *where responses live* (store),
and *how the web talks* (handlers) in separate packages, and each of these
extensions touches only one of them.
```
