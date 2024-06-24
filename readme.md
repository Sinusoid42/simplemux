# Simple Golang Multiplexer Implementation

This is a simple implementation of the already existing ServeHTTP for a given Websocket.
This multiplexer just simply runs HTTP, allows very fast Restful API implementations in Go.

---
 
 
```go
package main

import (
	"net/http"
	"github.com/Sinusoid42/simplemux"
)

func handleFunc(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func main() {
	mux := simplemux.Generate_mulitplexer() //Create a new multiplexer

	config := simplemux.Mux_config{
		Addr: ":9000",
	}

	mux.Start(&config) //Start the server

	mux.AddRoute("GET /index", handleFunc) //Add the callback function to the route

	mux.Wait() //Wait for the server to stop or on SIGITNT
}
```

---

Either a request method exactly matching the http method in the definition or if nothing is provided a wildcard is used.
This software adds only a small and fast parsing and better autoresponse to the already existing ServeHTTP.