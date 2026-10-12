package web

import (
	"embed"
	"html/template"
	"net/url"
)

//go:embed templates/*.html
var templateFiles embed.FS

var page = template.Must(template.New("page.html").Funcs(template.FuncMap{
	"inc":      func(value int) int { return value + 1 },
	"longDate": formatLongDate,
	"urlquery": url.QueryEscape,
}).ParseFS(templateFiles, "templates/*.html"))
