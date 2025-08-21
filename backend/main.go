package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	addr := ":7887"

	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/files/", http.StripPrefix("/files/", fs)) 

	http.HandleFunc("/md/", func(w http.ResponseWriter, r *http.Request) {
		path := "./public" + r.URL.Path[len("/md/")-1:] 
		if !strings.HasSuffix(path, ".md") {
			http.Error(w, "Not a markdown file", http.StatusBadRequest)
			return
		}

		f, err := os.Open(path)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<html><body><pre>%s</pre></body></html>", content)
	})

	log.Printf("Serving files at http://localhost%s/", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

