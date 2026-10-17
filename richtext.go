package main

import (
	"bytes"
	"html/template"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

var markdownConverter = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(gmhtml.WithHardWraps()),
)

// richTextHTML renders product/page copy that may be HTML or Markdown.
// Content that already looks like HTML is returned as-is; otherwise Markdown is converted.
func richTextHTML(src string) template.HTML {
	src = strings.TrimSpace(src)
	if src == "" {
		return ""
	}
	if looksLikeHTML(src) {
		return template.HTML(src)
	}
	var buf bytes.Buffer
	if err := markdownConverter.Convert([]byte(src), &buf); err != nil {
		escaped := template.HTMLEscapeString(src)
		return template.HTML(strings.ReplaceAll(escaped, "\n", "<br>\n"))
	}
	return template.HTML(buf.String())
}

func looksLikeHTML(src string) bool {
	s := strings.TrimSpace(src)
	if !strings.HasPrefix(s, "<") {
		return false
	}
	lower := strings.ToLower(s)
	for _, tag := range []string{
		"<p", "<div", "<ul", "<ol", "<li", "<h1", "<h2", "<h3", "<h4",
		"<br", "<strong", "<em", "<a ", "<span", "<blockquote", "<table",
	} {
		if strings.Contains(lower, tag) {
			return true
		}
	}
	return false
}
