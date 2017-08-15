package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spreadshirt/go-feature"
)

var features struct {
	Scream feature.Flag
}

func main() {
	featureSet := feature.NewSet()
	features.Scream, _ = featureSet.NewFlag("scream")

	http.HandleFunc("/hello", helloHandler)
	http.Handle("/features/", featureSet)

	log.Fatal(http.ListenAndServe(":22022", nil))
}

func helloHandler(w http.ResponseWriter, req *http.Request) {
	var msg string
	if features.Scream.IsEnabled() {
		msg = "HELLO!!!!"
	} else {
		msg = "hello."
	}

	fmt.Fprintf(w, msg)
}
