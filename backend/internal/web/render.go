package web

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
)

type pageData struct {
	Title    string
	CSRF     string
	LoggedIn bool
	Data     any
	Body     template.HTML
}

func (h *Handler) renderView(w http.ResponseWriter, r *http.Request, title, view string, data any) {
	pageModel := pageData{Title: title, CSRF: h.csrf(w, r), LoggedIn: h.authorized(r), Data: data}
	var body bytes.Buffer
	if err := page.ExecuteTemplate(&body, view, pageModel); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		fmt.Println(err)
		return
	}
	pageModel.Body = template.HTML(body.String())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := page.ExecuteTemplate(w, "page", pageModel); err != nil {
		fmt.Println(err)
	}
}
