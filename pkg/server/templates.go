package server

import (
	"embed"
	"html/template"
	"io"
)

//go:embed static/*
var templateFS embed.FS

var templates = template.Must(template.New("").ParseFS(templateFS, "static/*.html"))

func render(w io.Writer, name string, data any) error {
	return templates.ExecuteTemplate(w, name, data)
}
