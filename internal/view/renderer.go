package view

import (
	"bytes"
	"html/template"
	"log"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gogf/gf/v2/net/ghttp"
)

type Renderer struct {
	templates *template.Template
}

func NewRenderer(pattern string) (*Renderer, error) {
	tpl, err := template.New("base").Funcs(template.FuncMap{
		"year": func(t time.Time) int { return t.Year() },
		"since": func(t time.Time) string {
			d := time.Since(t)
			switch {
			case d < time.Hour:
				return strconv.Itoa(max(1, int(d.Minutes()))) + " min ago"
			case d < 24*time.Hour:
				return strconv.Itoa(int(d.Hours())) + " hours ago"
			case d < 30*24*time.Hour:
				return strconv.Itoa(int(d.Hours()/24)) + " days ago"
			default:
				return t.Format("2006-01-02")
			}
		},
	}).ParseGlob(pattern)
	if err != nil {
		return nil, err
	}
	return &Renderer{templates: tpl}, nil
}

func NewDefaultRenderer() (*Renderer, error) {
	return NewRenderer(filepath.Join("web", "templates", "*.tmpl"))
}

func (r *Renderer) HTML(req *ghttp.Request, name string, data any) {
	req.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := r.templates.ExecuteTemplate(req.Response.Writer, name, data); err != nil {
		log.Println(err)
	}
}

func (r *Renderer) HTMLStatus(req *ghttp.Request, status int, name string, data any) {
	var buffer bytes.Buffer
	if err := r.templates.ExecuteTemplate(&buffer, name, data); err != nil {
		log.Println(err)
		req.Response.WriteStatus(status)
		return
	}
	req.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	req.Response.WriteStatus(status, buffer.Bytes())
}
