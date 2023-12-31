package main

import (
	"github.com/single-proxy/api"
	"log"
	"net/http"
	"strconv"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		api.ProxyAPI(w, r)
	})

	log.Printf("Starting server on port %d\n", api.PORT)
	if err := http.ListenAndServe(":"+strconv.Itoa(api.PORT), nil); err != nil {
		log.Fatal(err)
	}
}
