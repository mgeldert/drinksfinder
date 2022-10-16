package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type location struct {
	longitude float64
	latitude float64
}

const (
	drinksfinderBasePath = "/api/drinksfinder/v1"
)

var (
	// URL of the data access service
	dataAccessUrl = os.Getenv("DATA_ACCESS_URL")

	// Whether the data access service is available
	dataAccessLayerAvailable = true
)

/*
	Test if a value is present in a string array.
*/
func StringArrayContains(haystack []string, needle string) bool {
	for _, value := range haystack {
		if value == needle {
			return true
		}
	}
	return false
}

/*
	Parse the request's querystring arguments for pagination directives.
*/
func getPaginationParameters(qs url.Values) *map[string]int {
	result := make(map[string]int)
	if start := qs.Get("start"); start != "" {
		if val, err := strconv.Atoi(start); err == nil {
			result["start"] = val
		}
	}
	if limit := qs.Get("limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			result["limit"] = val
		}
	}
	return &result
}

/*
	Send an HTTP error response.
*/
func setErrorResponse(w *http.ResponseWriter, errorType int, errorMsg string) {
	(*w).Header().Set("Content-Type", "application/json")
	(*w).WriteHeader(errorType)
	fmt.Fprintf(*w, "{\"error\": \"%s\"}", errorMsg)
}

func livenessHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	if dataAccessLayerAvailable {
		fmt.Fprintf(w, "OK")
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func dataAccessAvailabilityChecker() {
	for ;; {
		response, err := http.Get(dataAccessUrl + "ready")
		if err != nil || response.StatusCode != 200 {
			dataAccessLayerAvailable = false
		} else {
			dataAccessLayerAvailable = true
		}
		time.Sleep(5 * time.Second)
	}
}

func main() {
	http.HandleFunc(drinksfinderBasePath + "/", drinksfinderV1Handler)
	http.HandleFunc("/live", livenessHandler)
	http.HandleFunc("/ready", readinessHandler)

	// Run data access layer healthcheck in a go-routine
	go dataAccessAvailabilityChecker()

	fmt.Printf(
		"Starting server at port 8080 with data access at %s\n", dataAccessUrl,
	)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
