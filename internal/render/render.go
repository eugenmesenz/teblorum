package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
)

// TemplateFS — минимальный интерфейс для чтения файлов шаблонов из embed.FS
// или тестовой файловой системы.
type TemplateFS interface {
	ReadFile(name string) ([]byte, error)
}

// TemplateRenderer — кешированный парсер и рендерер шаблонов.
type TemplateRenderer struct {
	pages    map[string]*template.Template
	partials *template.Template
	funcMap  template.FuncMap
}

// NewTemplateRenderer создаёт рендерер, парсит все шаблоны при старте.
func NewTemplateRenderer(tfs TemplateFS) (*TemplateRenderer, error) {
	r := &TemplateRenderer{
		funcMap: template.FuncMap{
			"add": func(a, b int) int { return a + b },
			"sub": func(a, b int) int { return a - b },
		},
	}

	if err := r.parseAll(tfs); err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return r, nil
}

// pageTemplates — список известных страниц.
var pageTemplates = []string{"feed", "article", "thread", "user"}

// partialTemplates — список известных partial-шаблонов (без расширения).
var partialTemplates = []string{"comment", "comment_form", "pagination"}

func (r *TemplateRenderer) parseAll(tfs TemplateFS) error {
	// Парсим partials
	partials := template.New("").Funcs(r.funcMap)

	for _, name := range partialTemplates {
		data, err := tfs.ReadFile("web/templates/partials/" + name + ".html")
		if err != nil {
			return fmt.Errorf("read partial %s: %w", name, err)
		}
		if _, err := partials.New(name + ".html").Parse(string(data)); err != nil {
			return fmt.Errorf("parse partial %s: %w", name, err)
		}
	}
	r.partials = partials

	// Парсим страницы
	r.pages = make(map[string]*template.Template, len(pageTemplates))

	for _, name := range pageTemplates {
		layoutData, err := tfs.ReadFile("web/templates/layout.html")
		if err != nil {
			return fmt.Errorf("read layout: %w", err)
		}

		pageData, err := tfs.ReadFile("web/templates/pages/" + name + ".html")
		if err != nil {
			return fmt.Errorf("read page %s: %w", name, err)
		}

		combined := string(layoutData) + "\n" + string(pageData)

		tmpl, err := template.New("layout.html").Funcs(r.funcMap).Parse(combined)
		if err != nil {
			return fmt.Errorf("parse combined %s: %w", name, err)
		}
		r.pages[name] = tmpl
	}

	return nil
}

// PageHTTP рендерит полную страницу с layout в http.ResponseWriter.
// Если isHTMX — рендерит только блок content.
func (r *TemplateRenderer) PageHTTP(w http.ResponseWriter, name string, data any, isHTMX bool) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	tmpl, ok := r.pages[name]
	if !ok {
		r.WriteError(w, http.StatusInternalServerError, "unknown page: "+name)
		return
	}

	if isHTMX {
		if err := tmpl.ExecuteTemplate(w, "content", data); err != nil {
			http.Error(w, "render error", http.StatusInternalServerError)
		}
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// PageToString рендерит страницу в строку (для тестов).
func (r *TemplateRenderer) PageToString(name string, data any, isHTMX bool) (string, error) {
	tmpl, ok := r.pages[name]
	if !ok {
		return "", fmt.Errorf("unknown page: %s", name)
	}

	var buf bytes.Buffer
	if isHTMX {
		if err := tmpl.ExecuteTemplate(&buf, "content", data); err != nil {
			return "", err
		}
	} else {
		if err := tmpl.ExecuteTemplate(&buf, "layout.html", data); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

// Partial рендерит HTMX-фрагмент (без layout).
func (r *TemplateRenderer) Partial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := r.partials.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// PartialString рендерит partial в строку (для тестов).
func (r *TemplateRenderer) PartialString(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := r.partials.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// WriteError отправляет HTML-страницу ошибки.
func (r *TemplateRenderer) WriteError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	var title string
	switch status {
	case http.StatusNotFound:
		title = "404 — Страница не найдена"
	case http.StatusForbidden:
		title = "403 — Доступ запрещён"
	default:
		title = fmt.Sprintf("%d — Ошибка", status)
	}

	fmt.Fprintf(w, "<h1>%s</h1><p>%s</p>", title, template.HTMLEscapeString(message))
}

// Error — псевдоним для WriteError (совместимость).
func (r *TemplateRenderer) Error(w http.ResponseWriter, status int, message string) {
	r.WriteError(w, status, message)
}

// DetectHTMX проверяет, является ли запрос HTMX-запросом.
func DetectHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// Compile-time check.
var _ io.Writer = new(bytes.Buffer)
