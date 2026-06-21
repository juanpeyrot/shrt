package web

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"time"
)

type TemplateRegistry struct {
	pages    map[string]*template.Template
	partials map[string]*template.Template
	funcMap  template.FuncMap
	baseURL  string
}

func NewTemplateRegistry(templatesDir, baseURL string) (*TemplateRegistry, error) {
	tr := &TemplateRegistry{
		pages:    make(map[string]*template.Template),
		partials: make(map[string]*template.Template),
		baseURL:  baseURL,
	}
	tr.funcMap = template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("Jan 02, 2006")
		},
		"formatDateTime": func(t any) string {
			switch v := t.(type) {
			case time.Time:
				return v.Format("Jan 02, 2006 15:04")
			case *time.Time:
				if v == nil {
					return ""
				}
				return v.Format("Jan 02, 2006 15:04")
			default:
				return ""
			}
		},
		"shortURL": func(code string) string {
			return baseURL + "/" + code
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"seq": func(start, end int) []int {
			s := make([]int, 0, end-start+1)
			for i := start; i <= end; i++ {
				s = append(s, i)
			}
			return s
		},
		"pctOf": func(value, max int64) int {
			if max == 0 {
				return 0
			}
			return int(value * 100 / max)
		},
		"dict": func(pairs ...any) map[string]any {
			m := make(map[string]any, len(pairs)/2)
			for i := 0; i < len(pairs)-1; i += 2 {
				key, _ := pairs[i].(string)
				m[key] = pairs[i+1]
			}
			return m
		},
	}

	if err := tr.parsePages(templatesDir); err != nil {
		return nil, fmt.Errorf("parse pages: %w", err)
	}
	if err := tr.parsePartials(templatesDir); err != nil {
		return nil, fmt.Errorf("parse partials: %w", err)
	}
	return tr, nil
}

func (tr *TemplateRegistry) parsePages(dir string) error {
	layoutFile := filepath.Join(dir, "layouts", "base.html")
	partialFiles, _ := filepath.Glob(filepath.Join(dir, "partials", "*.html"))

	pageFiles, err := filepath.Glob(filepath.Join(dir, "pages", "*.html"))
	if err != nil {
		return err
	}

	for _, pageFile := range pageFiles {
		name := fileNameWithoutExt(pageFile)

		files := []string{layoutFile}
		files = append(files, partialFiles...)
		files = append(files, pageFile)

		tmpl, err := template.New(filepath.Base(layoutFile)).Funcs(tr.funcMap).ParseFiles(files...)
		if err != nil {
			return fmt.Errorf("parse page %s: %w", name, err)
		}
		tr.pages[name] = tmpl
	}
	return nil
}

func (tr *TemplateRegistry) parsePartials(dir string) error {
	partialFiles, err := filepath.Glob(filepath.Join(dir, "partials", "*.html"))
	if err != nil {
		return err
	}
	if len(partialFiles) == 0 {
		return nil
	}

	combined, err := template.New("partials").Funcs(tr.funcMap).ParseFiles(partialFiles...)
	if err != nil {
		return fmt.Errorf("parse partials: %w", err)
	}
	tr.partials["_combined"] = combined
	return nil
}

func (tr *TemplateRegistry) Render(w http.ResponseWriter, page string, data any) {
	tmpl, ok := tr.pages[page]
	if !ok {
		http.Error(w, "template not found: "+page, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (tr *TemplateRegistry) RenderPartial(w http.ResponseWriter, partial string, data any) {
	combined, ok := tr.partials["_combined"]
	if !ok {
		http.Error(w, "no partials loaded", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := combined.ExecuteTemplate(w, partial, data); err != nil {
		http.Error(w, "partial not found: "+partial+": "+err.Error(), http.StatusInternalServerError)
	}
}

func fileNameWithoutExt(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}

