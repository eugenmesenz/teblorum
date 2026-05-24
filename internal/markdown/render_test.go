package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderToHTML(t *testing.T) {
	t.Run("basic markdown", func(t *testing.T) {
		md := `# Hello

This is **bold** text.`

		html, err := RenderToHTML(md)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(html, "<h1") {
			t.Error("expected <h1> tag in output")
		}
		if !strings.Contains(html, "<strong>") {
			t.Error("expected <strong> tag in output")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		html, err := RenderToHTML("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if html != "" {
			t.Errorf("expected empty string, got %q", html)
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		html, err := RenderToHTML("   ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if html != "" {
			t.Errorf("expected empty string, got %q", html)
		}
	})

	t.Run("XSS prevention", func(t *testing.T) {
		md := `<script>alert('xss')</script>`
		html, err := RenderToHTML(md)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(html, "<script>") {
			t.Error("script tag should be sanitized")
		}
	})

	t.Run("linkify", func(t *testing.T) {
		md := "https://example.com"
		html, err := RenderToHTML(md)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(html, "<a") {
			t.Error("expected link in output")
		}
		if !strings.Contains(html, "rel=") {
			t.Error("expected rel attribute on link")
		}
	})

	t.Run("GFM table", func(t *testing.T) {
		md := "| A | B |\n|---|---|\n| 1 | 2 |"
		html, err := RenderToHTML(md)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(html, "<table") {
			t.Error("expected <table> tag for GFM table")
		}
	})

	t.Run("GFM strikethrough", func(t *testing.T) {
		md := "~~strikethrough~~"
		html, err := RenderToHTML(md)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(html, "<del>") {
			t.Error("expected <del> tag for strikethrough")
		}
	})

	t.Run("relative URL allowed", func(t *testing.T) {
		md := "[relative](/page)"
		html, err := RenderToHTML(md)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(html, `href="/page"`) {
			t.Error("expected relative URL to be preserved")
		}
	})

	t.Run("golden file", func(t *testing.T) {
		mdBytes, err := os.ReadFile(filepath.Join("testdata", "basic.md"))
		if err != nil {
			t.Skip("testdata not found:", err)
		}

		html, err := RenderToHTML(string(mdBytes))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		goldenPath := filepath.Join("testdata", "basic_html.golden")
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			// Записываем golden файл при первом запуске
			os.WriteFile(goldenPath, []byte(html), 0644)
		}

		golden, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("read golden file: %v", err)
		}

		// Нормализуем whitespace для сравнения
		expected := strings.TrimSpace(string(golden))
		got := strings.TrimSpace(html)

		if expected != got {
			t.Errorf("golden file mismatch\nexpected:\n%s\n\ngot:\n%s", expected, got)
		}
	})
}

func TestRenderToHTMLSimple(t *testing.T) {
	t.Run("simple markdown", func(t *testing.T) {
		md := "Hello **world**"
		html, err := RenderToHTMLSimple(md)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(html, "<strong>") {
			t.Error("expected <strong> tag")
		}
	})

	t.Run("no advanced features", func(t *testing.T) {
		md := "| A | B |\n|---|---|\n| 1 | 2 |"
		html, err := RenderToHTMLSimple(md)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Simple mode не должен рендерить таблицы
		if strings.Contains(html, "<table") {
			t.Error("expected no <table> in simple mode")
		}
	})

	t.Run("XSS prevention in simple mode", func(t *testing.T) {
		md := `<script>alert('xss')</script>`
		html, err := RenderToHTMLSimple(md)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(html, "<script>") {
			t.Error("script tag should be sanitized")
		}
	})
}

func TestStripMarkdown(t *testing.T) {
	t.Run("basic stripping", func(t *testing.T) {
		md := "Hello **world**"
		text := StripMarkdown(md)
		if !strings.Contains(text, "Hello world") {
			t.Errorf("expected 'Hello world', got %q", text)
		}
	})

	t.Run("heading stripping", func(t *testing.T) {
		md := "# Title\n\nParagraph."
		text := StripMarkdown(md)
		if !strings.Contains(text, "Title") {
			t.Errorf("expected 'Title', got %q", text)
		}
		if !strings.Contains(text, "Paragraph") {
			t.Errorf("expected 'Paragraph', got %q", text)
		}
	})

	t.Run("empty input", func(t *testing.T) {
		text := StripMarkdown("")
		if text != "" {
			t.Errorf("expected empty string, got %q", text)
		}
	})

	t.Run("XSS stripped from plain text", func(t *testing.T) {
		md := "<script>alert('xss')</script>"
		text := StripMarkdown(md)
		if strings.Contains(text, "<script>") {
			t.Error("script tag should not appear in plain text")
		}
	})
}
