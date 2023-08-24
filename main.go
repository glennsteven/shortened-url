package main

import (
	"fmt"
	"log"
	"net/http"
	"shortened_url/shortened"
	"strings"
)

func main() {
	s := "https://mail.yandex.com/?uid=1130000056464814#message/182114309931795933"
	shortTest1, _ := shortened.Shorten(s)
	shortTest2, _ := shortened.Shorten("google.com")
	fmt.Println("Shorten-1 = ", fmt.Sprintf("http://localhost:8000/%v", shortTest1))
	fmt.Println("Shorten-2 = ", fmt.Sprintf("http://localhost:8000/%v", shortTest2))
	originalURL, err := shortened.Extend(shortTest1)
	if err != nil {
		log.Printf("something went wrong")
		return
	}
	fmt.Println("OriginalURL: ", originalURL)
	newServer()
}

func newServer() {
	http.ListenAndServe(":8000", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		p := strings.TrimPrefix(request.URL.Path, "/")
		url, err := shortened.Extend(p)
		if err != nil {
			writer.WriteHeader(http.StatusNotFound)
			return
		}

		http.Redirect(writer, request, url, http.StatusPermanentRedirect)
	}))
}
