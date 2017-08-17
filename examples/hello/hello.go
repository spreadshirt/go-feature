package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/spreadshirt/go-feature"
)

var features struct {
	Scream   feature.Flag
	Surprise feature.Flag
}

func main() {
	// initialize RNG so that ratio flags are actually random
	rand.Seed(time.Now().UnixNano())

	featureSet := feature.NewSet()
	features.Scream, _ = featureSet.NewFlag("scream")
	features.Surprise = feature.NewRatioFlag("surprise", 0.1)
	featureSet.Add(features.Surprise)

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

	if features.Surprise.IsEnabled() {
		fmt.Fprintf(w, "\n\nsurpriiiise!!!")
	}
}
