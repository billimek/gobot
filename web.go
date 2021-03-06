package main

import (
	"fmt"
	"html/template"
	"net/http"
)

type Page struct {
	Title   string
	Date    string
	Results []string
}

var templates = template.Must(template.ParseFiles(
	"html/index.html",
	"html/advanced.html",
	"html/list.html",
))

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func eqHandler(w http.ResponseWriter, r *http.Request) {
	p := &Page{Title: "title", Date: "2013-10-21"}
	switch r.URL.Path[9:] {
	case "", "index.php":
		p.Title = "TorilMUD Equipment Database"
		if r.Method == "POST" {
			results := FindItem(r.PostFormValue("itemName"), "short_stats")
			p.Results = results
		}
		renderTemplate(w, "index.html", p)
	case "advanced.php":
		p.Title = "Advanced Search"
		if r.Method == "POST" {
			//p.Results = "success!"
		}
		renderTemplate(w, "advanced.html", p)
	case "list.php":
		p.Title = "Copy/Paste Statter"
		if r.Method == "POST" {
			//p.Results = "success!"
		}
		renderTemplate(w, "list.html", p)
	}
}

func todHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Todrael's Lair, %s", r.URL.Path[1:])
}
