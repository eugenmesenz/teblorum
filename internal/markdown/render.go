package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// RenderToHTML конвертирует Markdown в безопасный HTML.
// Использует goldmark для парсинга и bluemonday для санитизации.
func RenderToHTML(md string) (string, error) {
	if strings.TrimSpace(md) == "" {
		return "", nil
	}

	// Настройка goldmark
	gm := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,         // Tables, strikethrough, task lists, etc.
			extension.Typographer, // Smartypants: кавычки, тире, многоточия
			extension.Linkify,     // Автоссылки
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Авто-ID для заголовков
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Разрешаем HTML-теги (они будут отфильтрованы bluemonday)
		),
	)

	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		return "", fmt.Errorf("convert markdown: %w", err)
	}

	// Санитизация HTML через bluemonday
	policy := bluemonday.UGCPolicy()

	// Разрешаем полезные теги для контента
	policy.AllowAttrs("href", "title").OnElements("a")
	policy.AllowAttrs("src", "alt", "title").OnElements("img")
	policy.AllowAttrs("class", "id").Globally()
	policy.AllowAttrs("target", "rel").OnElements("a")
	policy.AllowStandardURLs()
	policy.AllowRelativeURLs(true)

	// Разрешаем заголовки
	policy.AllowElements("h1", "h2", "h3", "h4", "h5", "h6")

	// Разрешаем форматирование текста
	policy.AllowElements("p", "br", "strong", "em", "del", "ins", "sub", "sup",
		"code", "pre", "kbd", "samp", "mark", "small", "abbr", "dfn", "cite", "q")

	// Разрешаем списки
	policy.AllowElements("ul", "ol", "li", "dl", "dt", "dd")

	// Разрешаем таблицы
	policy.AllowElements("table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption", "colgroup", "col")

	// Разрешаем блоки
	policy.AllowElements("blockquote", "figure", "figcaption", "hr", "div", "span",
		"details", "summary")

	// Разрешаем ссылки и изображения
	policy.AllowElements("a", "img") // уже разрешено выше с аттрибутами

	// Разрешаем HTML5 теги для семантики
	policy.AllowElements("article", "section", "header", "footer", "nav", "aside")

	html := policy.SanitizeBytes(buf.Bytes())
	return string(html), nil
}

// RenderToHTMLSimple — упрощённый рендеринг без GFM-расширений (для комментариев).
func RenderToHTMLSimple(md string) (string, error) {
	if strings.TrimSpace(md) == "" {
		return "", nil
	}

	gm := goldmark.New(
		goldmark.WithExtensions(
			extension.Linkify,
		),
	)

	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		return "", fmt.Errorf("convert markdown: %w", err)
	}

	// Более строгий политика для комментариев
	policy := bluemonday.StrictPolicy()
	policy.AllowElements("p", "br", "strong", "em", "code", "a", "del", "pre")
	policy.AllowAttrs("href", "title").OnElements("a")
	policy.AllowStandardURLs()
	policy.AllowRelativeURLs(true)

	html := policy.SanitizeBytes(buf.Bytes())
	return string(html), nil
}

// StripMarkdown удаляет Markdown-разметку, возвращая plain text (для превью).
func StripMarkdown(md string) string {
	html, err := RenderToHTML(md)
	if err != nil {
		return md
	}

	// Простое удаление HTML-тегов
	var result bytes.Buffer
	inTag := false
	for _, r := range html {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}

	// Заменяем HTML-entities
	text := result.String()
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return text
}
