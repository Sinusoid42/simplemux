package main

import (
	"net/http"
	"simplemux"
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
