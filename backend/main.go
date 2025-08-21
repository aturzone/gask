package main

import (
	"log"
	"net/http"
)

func main() {

	addr := ":7887"
	fs := http.FileServer(http.Dir("/.public"))
	http.Handle("/", fs)

	log.Printf("serving ./public at http://localhost%s/", addr)
	log.Fatal(http.ListenAndServe(addr, nil))

}
