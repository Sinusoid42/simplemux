package main

import (
	"fmt"
	"net/http"
	"simplemux"
)

func handleFunc(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func main() {
	mux := simplemux.Generate_mulitplexer() //Create a new multiplexer

	// Adding a route without content type specification
	mux.AddRoute("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	}, "")

	mux.Redirect("/index", "/")

	// Adding a route with content type specification
	mux.AddRoute("POST /submit", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Form submitted!"))
	}, "application/x-www-form-urlencoded")

	// Adding a JSON-specific route
	mux.AddRoute("POST /submit-json", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("JSON submitted!"))
	}, "application/json")

	// Adding middleware
	mux.Use(LoggingMiddleware)

	config := &simplemux.Mux_config{
		Addr: ":8080",
	}

	mux.Start(config)

	// Wait for the server to shutdown gracefully
	mux.Wait()
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Received request: %s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
