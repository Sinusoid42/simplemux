package main

import (
	"fmt"
	"net/http"
	"simplemux"
)

func handleFunc(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("haha"))

}

func main() {
	mux := simplemux.Generate_mulitplexer()

	config := simplemux.Mux_config{
		Addr: ":9000",
	}

	mux.Start(&config)

	mux.AddRoute(" /test", handleFunc)

	fmt.Println("Do Something")

	mux.Wait()

}
