// Package forms defines the data model for a "form template" and loads those
// templates from a directory of JSON files. A form is just a title plus an
// ordered list of typed fields, so the same generic HTML template can render
// ANY form the user drops into the forms/ directory — no recompilation needed.
package forms

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Field is one input in a form template.
type Field struct {
	Name        string   `json:"name"`     // form field name (the POST key)
	Label       string   `json:"label"`    // human label shown next to the input
	Type        string   `json:"type"`     // text, email, password, number, tel, url, date, textarea, select, checkbox, radio
	Required    bool     `json:"required"` // must be present/non-empty
	Placeholder string   `json:"placeholder,omitempty"`
	Help        string   `json:"help,omitempty"`    // small helper text under the input
	Options     []string `json:"options,omitempty"` // choices for select/radio
}

// Definition is a whole form template.
type Definition struct {
	ID          string  `json:"id"` // URL-safe identifier, e.g. "contact"
	Title       string  `json:"title"`
	Description string  `json:"description"`
	SubmitLabel string  `json:"submitLabel,omitempty"`
	Fields      []Field `json:"fields"`
}

// Registry holds every loaded form, keyed by ID.
type Registry struct {
	byID map[string]*Definition
}

// Get returns a form by ID.
func (r *Registry) Get(id string) (*Definition, bool) {
	d, ok := r.byID[id]
	return d, ok
}

// Len is the number of loaded forms.
func (r *Registry) Len() int { return len(r.byID) }

// All returns every form, sorted by title (stable order for the index page).
func (r *Registry) All() []*Definition {
	out := make([]*Definition, 0, len(r.byID))
	for _, d := range r.byID {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

// LoadDir reads every *.json file in dir and builds a Registry. Each file is one
// form template. Definitions are validated up front so a malformed form is a
// startup error rather than a runtime surprise.
func LoadDir(dir string) (*Registry, error) {
	reg := &Registry{byID: map[string]*Definition{}}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		var d Definition
		dec := json.NewDecoder(strings.NewReader(string(b)))
		dec.DisallowUnknownFields() // typos in the JSON become errors, not silent no-ops
		if err := dec.Decode(&d); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if err := d.validate(); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		if _, dup := reg.byID[d.ID]; dup {
			return nil, fmt.Errorf("%s: duplicate form id %q", path, d.ID)
		}
		reg.byID[d.ID] = &d
	}
	return reg, nil
}

var validType = map[string]bool{
	"text": true, "email": true, "password": true, "number": true, "tel": true,
	"url": true, "date": true, "textarea": true, "select": true, "checkbox": true, "radio": true,
}

func (d *Definition) validate() error {
	if d.ID == "" {
		return fmt.Errorf("missing id")
	}
	if d.Title == "" {
		return fmt.Errorf("form %q: missing title", d.ID)
	}
	if len(d.Fields) == 0 {
		return fmt.Errorf("form %q: has no fields", d.ID)
	}
	seen := map[string]bool{}
	for i, f := range d.Fields {
		if f.Name == "" {
			return fmt.Errorf("form %q: field %d missing name", d.ID, i)
		}
		if seen[f.Name] {
			return fmt.Errorf("form %q: duplicate field name %q", d.ID, f.Name)
		}
		seen[f.Name] = true
		if !validType[f.Type] {
			return fmt.Errorf("form %q: field %q has invalid type %q", d.ID, f.Name, f.Type)
		}
		if (f.Type == "select" || f.Type == "radio") && len(f.Options) == 0 {
			return fmt.Errorf("form %q: field %q of type %q needs options", d.ID, f.Name, f.Type)
		}
	}
	if d.SubmitLabel == "" {
		d.SubmitLabel = "Submit"
	}
	return nil
}

// Clean validates and coerces submitted values against this form's fields.
// It returns a cleaned map (ready to store) plus a list of human-readable
// validation errors. Values that don't correspond to a defined field are
// dropped — the form definition is the source of truth for what we accept.
func (d *Definition) Clean(values url.Values) (map[string]any, []string) {
	out := map[string]any{}
	var errs []string
	for _, f := range d.Fields {
		if f.Type == "checkbox" {
			// A checkbox is "on" when the key is present at all.
			out[f.Name] = values.Has(f.Name)
			continue
		}
		v := strings.TrimSpace(values.Get(f.Name))
		if v == "" {
			if f.Required {
				errs = append(errs, f.Label+" is required.")
			}
			out[f.Name] = ""
			continue
		}
		switch f.Type {
		case "number":
			n, err := strconv.ParseFloat(v, 64)
			if err != nil {
				errs = append(errs, f.Label+" must be a number.")
				out[f.Name] = v
				continue
			}
			out[f.Name] = n
		case "email":
			if !strings.Contains(v, "@") || !strings.Contains(v, ".") {
				errs = append(errs, f.Label+" must be a valid email address.")
			}
			out[f.Name] = v
		case "select", "radio":
			ok := false
			for _, o := range f.Options {
				if o == v {
					ok = true
					break
				}
			}
			if !ok {
				errs = append(errs, f.Label+" has an invalid choice.")
			}
			out[f.Name] = v
		default:
			out[f.Name] = v
		}
	}
	return out, errs
}
