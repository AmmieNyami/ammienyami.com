package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var port = 6969
var pagesDir = "pages"
var templatesDir = "templates"
var defaultTemplate = "default-template.html"
var staticDir = "static"

func removeFirstPathComponent(p string) string {
	p = path.Clean(p)
	p = strings.TrimPrefix(p, "/")

	parts := strings.SplitN(p, "/", 2)
	if len(parts) < 2 {
		return "/"
	}

	return "/" + parts[1]
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}

func removeExtension(filename string) string {
	ext := path.Ext(filename)
	return strings.TrimSuffix(filename, ext)
}

func serveTemplateFile(templatePath string, w http.ResponseWriter) {
	fmt.Printf("Trying to serve template %q...\n", templatePath)

	if !fileExists(templatePath) {
		http.Error(w, "Not found (404)", http.StatusNotFound)
		return
	}

	templateInput, err := NewTemplateInputFromFile(templatePath, staticDir)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Internal server error (500)", http.StatusInternalServerError)
		return
	}

	templateInputContent, err := templateInput.Render()
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Internal server error (500)", http.StatusInternalServerError)
		return
	}

	template, err := NewTemplateFromFile(path.Join(templatesDir, defaultTemplate), TemplateContext{
		StaticDir: staticDir,
		Content:   templateInputContent,
		Variables: templateInput.variables,
	})
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Internal server error (500)", http.StatusInternalServerError)
		return
	}

	renderedTemplate, err := template.Render()
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Internal server error (500)", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, renderedTemplate)
}

func main() {
	flag.IntVar(&port, "port", 6969, "The port the HTTP server will run on")
	flag.StringVar(&pagesDir, "pages-dir", "pages", "The directory where page template inputs are located")
	flag.StringVar(&templatesDir, "templates-dir", "templates", "The directory where plates are located")
	flag.StringVar(&defaultTemplate, "default-template", "default-template.html", "The default template used to render pages")
	flag.StringVar(&staticDir, "static-dir", "static", "The directory containing the files that are served from \"/static\"")
	flag.Parse()

	fmt.Printf("Starting HTTP server on port %d...\n", port)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "Not found (404)", http.StatusNotFound)
			return
		}
		indexTemplatePath := path.Join(pagesDir, "index.template.html")
		serveTemplateFile(indexTemplatePath, w)
	})

	http.HandleFunc("/pages/", func(w http.ResponseWriter, r *http.Request) {
		requestedTemplatePath := filepath.Clean(filepath.Join(pagesDir, removeExtension(removeFirstPathComponent(r.URL.Path))+".template.html"))
		if !strings.HasPrefix(requestedTemplatePath, filepath.Clean(pagesDir + "/")) {
			http.Error(w, "Not found (404)", http.StatusNotFound)
			return
		}
		serveTemplateFile(requestedTemplatePath, w)
	})

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil); err != nil {
		fmt.Printf("ERROR: failed to start HTTP server: %v\n", err)
		os.Exit(1)
	}
}
