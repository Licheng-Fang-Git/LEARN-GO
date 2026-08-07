// Package web is the HTTP layer: routing, request parsing, content negotiation
// (HTML for browsers, JSON for API clients), and rendering. It ties the form
// registry (what forms exist) to the store (where responses live).
package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"formserver/internal/forms"
	"formserver/internal/store"
)

// App holds the shared dependencies for all handlers.
type App struct {
	reg    *forms.Registry
	store  *store.Store
	tmpl   *template.Template
	static string
	log    *log.Logger
}

// New builds the App, parsing every template in tmplDir up front.
func New(reg *forms.Registry, st *store.Store, tmplDir, staticDir string, logger *log.Logger) (*App, error) {
	tmpl, err := template.New("").ParseGlob(filepath.Join(tmplDir, "*.html"))
	if err != nil {
		return nil, err
	}
	return &App{reg: reg, store: st, tmpl: tmpl, static: staticDir, log: logger}, nil
}

// Routes wires URLs to handlers using the method-aware ServeMux (Go 1.22+).
func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", a.handleIndex)
	mux.HandleFunc("GET /healthz", a.handleHealth)

	// Form rendering + submission (Create).
	mux.HandleFunc("GET /forms/{id}", a.handleShowForm)
	mux.HandleFunc("POST /forms/{id}/submit", a.handleSubmit)

	// Reading responses.
	mux.HandleFunc("GET /forms/{id}/submissions", a.handleList)
	mux.HandleFunc("GET /submissions/{id}", a.handleShowSubmission)

	// Update + Delete (REST verbs for API clients; POST fallback for browsers).
	mux.HandleFunc("PUT /submissions/{id}", a.handleUpdate)
	mux.HandleFunc("DELETE /submissions/{id}", a.handleDelete)
	mux.HandleFunc("POST /submissions/{id}/delete", a.handleDelete)

	// Static assets.
	fs := http.FileServer(http.Dir(a.static))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	return a.logging(mux)
}

// ---- view models -----------------------------------------------------------

// page carries the fields every template's shared header/footer needs.
type page struct {
	Title string
	Year  int
}

func newPage(title string) page { return page{Title: title, Year: time.Now().Year()} }

type indexView struct {
	page
	Forms []*forms.Definition
}

type formView struct {
	page
	Def    *forms.Definition
	Values map[string]string // for re-populating the form after a validation error
	Errors []string
}

type resultView struct {
	page
	FormID     string
	FormTitle  string
	Submission *store.Submission
	Pretty     string
}

type listView struct {
	page
	Def         *forms.Definition
	Submissions []*store.Submission
}

type kv struct {
	Label string
	Value any
}

type detailView struct {
	page
	Submission *store.Submission
	Rows       []kv
}

type errView struct {
	page
	Code    int
	Message string
}

// ---- handlers --------------------------------------------------------------

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "forms": a.reg.Len()})
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	// "GET /" is the catch-all; reject anything that isn't exactly the root.
	if r.URL.Path != "/" {
		a.fail(w, r, http.StatusNotFound, "Page not found.")
		return
	}
	if wantsJSON(r) {
		respondJSON(w, http.StatusOK, map[string]any{"forms": a.reg.All()})
		return
	}
	a.render(w, r, "index.html", indexView{page: newPage("Forms"), Forms: a.reg.All()})
}

func (a *App) handleShowForm(w http.ResponseWriter, r *http.Request) {
	def, ok := a.reg.Get(r.PathValue("id"))
	if !ok {
		a.fail(w, r, http.StatusNotFound, "No such form.")
		return
	}
	a.render(w, r, "form.html", formView{
		page:   newPage(def.Title),
		Def:    def,
		Values: map[string]string{},
	})
}

func (a *App) handleSubmit(w http.ResponseWriter, r *http.Request) {
	def, ok := a.reg.Get(r.PathValue("id"))
	if !ok {
		a.fail(w, r, http.StatusNotFound, "No such form.")
		return
	}

	values, err := parseValues(r)
	if err != nil {
		a.fail(w, r, http.StatusBadRequest, "Could not read submitted data.")
		return
	}

	data, verrs := def.Clean(values)
	if len(verrs) > 0 {
		// Validation failed: 422 with errors (JSON) or the form re-rendered (HTML).
		if wantsJSON(r) {
			respondJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"status": "error", "errors": verrs,
			})
			return
		}
		a.renderStatus(w, http.StatusUnprocessableEntity, "form.html", formView{
			page:   newPage(def.Title),
			Def:    def,
			Values: rawStrings(values),
			Errors: verrs,
		})
		return
	}

	sub, err := a.store.Create(r.Context(), def.ID, data)
	if err != nil {
		a.log.Printf("create submission for %q failed: %v", def.ID, err)
		a.fail(w, r, http.StatusInternalServerError, "Could not save your response.")
		return
	}

	// Notify the client of the outcome.
	a.log.Printf("stored submission #%d for form %q", sub.ID, def.ID)
	if wantsJSON(r) {
		respondJSON(w, http.StatusCreated, map[string]any{"status": "ok", "submission": sub})
		return
	}
	pretty, _ := json.MarshalIndent(sub.Data, "", "  ")
	a.renderStatus(w, http.StatusCreated, "result.html", resultView{
		page:       newPage("Submitted"),
		FormID:     def.ID,
		FormTitle:  def.Title,
		Submission: sub,
		Pretty:     string(pretty),
	})
}

func (a *App) handleList(w http.ResponseWriter, r *http.Request) {
	def, ok := a.reg.Get(r.PathValue("id"))
	if !ok {
		a.fail(w, r, http.StatusNotFound, "No such form.")
		return
	}
	subs, err := a.store.ListByForm(r.Context(), def.ID)
	if err != nil {
		a.log.Printf("list submissions for %q: %v", def.ID, err)
		a.fail(w, r, http.StatusInternalServerError, "Could not load responses.")
		return
	}
	if wantsJSON(r) {
		respondJSON(w, http.StatusOK, map[string]any{"form": def.ID, "submissions": subs})
		return
	}
	a.render(w, r, "list.html", listView{
		page:        newPage("Responses · " + def.Title),
		Def:         def,
		Submissions: subs,
	})
}

func (a *App) handleShowSubmission(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		a.fail(w, r, http.StatusBadRequest, "Invalid submission id.")
		return
	}
	sub, err := a.store.Get(r.Context(), id)
	if err == store.ErrNotFound {
		a.fail(w, r, http.StatusNotFound, "No such submission.")
		return
	} else if err != nil {
		a.fail(w, r, http.StatusInternalServerError, "Could not load submission.")
		return
	}
	if wantsJSON(r) {
		respondJSON(w, http.StatusOK, sub)
		return
	}
	a.render(w, r, "detail.html", detailView{
		page:       newPage(fmt.Sprintf("Response #%d", sub.ID)),
		Submission: sub,
		Rows:       a.orderedRows(sub),
	})
}

// handleUpdate is JSON-only (a REST PUT). It replaces a submission's data.
func (a *App) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		a.fail(w, r, http.StatusBadRequest, "Invalid submission id.")
		return
	}
	existing, err := a.store.Get(r.Context(), id)
	if err == store.ErrNotFound {
		a.fail(w, r, http.StatusNotFound, "No such submission.")
		return
	} else if err != nil {
		a.fail(w, r, http.StatusInternalServerError, "Could not load submission.")
		return
	}
	def, ok := a.reg.Get(existing.FormID)
	if !ok {
		a.fail(w, r, http.StatusConflict, "Form definition no longer exists.")
		return
	}
	values, err := parseValues(r)
	if err != nil {
		a.fail(w, r, http.StatusBadRequest, "Could not read submitted data.")
		return
	}
	data, verrs := def.Clean(values)
	if len(verrs) > 0 {
		respondJSON(w, http.StatusUnprocessableEntity, map[string]any{"status": "error", "errors": verrs})
		return
	}
	sub, err := a.store.Update(r.Context(), id, data)
	if err != nil {
		a.fail(w, r, http.StatusInternalServerError, "Could not update submission.")
		return
	}
	a.log.Printf("updated submission #%d", sub.ID)
	respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "submission": sub})
}

func (a *App) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		a.fail(w, r, http.StatusBadRequest, "Invalid submission id.")
		return
	}
	// Remember which form this belonged to so a browser can be sent back to its list.
	formID := ""
	if sub, err := a.store.Get(r.Context(), id); err == nil {
		formID = sub.FormID
	}
	if err := a.store.Delete(r.Context(), id); err == store.ErrNotFound {
		a.fail(w, r, http.StatusNotFound, "No such submission.")
		return
	} else if err != nil {
		a.fail(w, r, http.StatusInternalServerError, "Could not delete submission.")
		return
	}
	a.log.Printf("deleted submission #%d", id)
	if wantsJSON(r) {
		respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": id})
		return
	}
	if formID != "" {
		http.Redirect(w, r, "/forms/"+formID+"/submissions", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ---- helpers ---------------------------------------------------------------

// orderedRows returns a submission's values in the form's field order (maps are
// unordered), pairing each stored value with its human label.
func (a *App) orderedRows(sub *store.Submission) []kv {
	rows := make([]kv, 0)
	if def, ok := a.reg.Get(sub.FormID); ok {
		for _, f := range def.Fields {
			rows = append(rows, kv{Label: f.Label, Value: sub.Data[f.Name]})
		}
		return rows
	}
	// Fallback if the form definition is gone: show raw keys.
	for k, v := range sub.Data {
		rows = append(rows, kv{Label: k, Value: v})
	}
	return rows
}

// parseValues extracts submitted values from either an HTML form POST
// (application/x-www-form-urlencoded) or a JSON body, normalising both into
// url.Values so def.Clean can treat them identically.
func parseValues(r *http.Request) (url.Values, error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		var m map[string]any
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			return nil, err
		}
		v := url.Values{}
		for k, raw := range m {
			switch t := raw.(type) {
			case bool:
				if t {
					v.Set(k, "on") // checkbox semantics: present == true
				}
			case nil:
				// omit
			default:
				v.Set(k, fmt.Sprint(t))
			}
		}
		return v, nil
	}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	return r.PostForm, nil
}

// rawStrings flattens url.Values to the first value per key, for re-rendering.
func rawStrings(v url.Values) map[string]string {
	out := make(map[string]string, len(v))
	for k := range v {
		out[k] = v.Get(k)
	}
	return out
}

func wantsJSON(r *http.Request) bool {
	if r.URL.Query().Get("format") == "json" {
		return true
	}
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func respondJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *App) render(w http.ResponseWriter, r *http.Request, name string, data any) {
	a.renderStatus(w, http.StatusOK, name, data)
}

func (a *App) renderStatus(w http.ResponseWriter, code int, name string, data any) {
	// Render into a buffer first so a template error becomes a clean 500
	// instead of a half-written response.
	var buf bytes.Buffer
	if err := a.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		a.log.Printf("template %s: %v", name, err)
		http.Error(w, "Internal template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, _ = buf.WriteTo(w)
}

func (a *App) fail(w http.ResponseWriter, r *http.Request, code int, msg string) {
	if wantsJSON(r) {
		respondJSON(w, code, map[string]any{"status": "error", "error": msg})
		return
	}
	a.renderStatus(w, code, "error.html", errView{page: newPage("Error"), Code: code, Message: msg})
}

// logging records method, path, status and duration for every request.
type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (s *statusRecorder) WriteHeader(c int) {
	s.code = c
	s.ResponseWriter.WriteHeader(c)
}

func (a *App) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)
		a.log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.code, time.Since(start).Round(time.Millisecond))
	})
}
