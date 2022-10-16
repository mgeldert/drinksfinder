package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var (
	// Valid fields for results ordering
	drinkfinderV1OrderFields = []string{
		"beer", "atmosphere", "amenities", "value",
	}

	// Mask for valid UK postcodes
	postcodeValidation = regexp.MustCompile(
		"^[a-zA-Z]{1,2}[0-9]{1,2}\\s*[0-9][a-zA-Z]{2}$",
	)

	// Hard-coded long/lat of the X-Lab Leeds office
	xLabOfficeLocation = location{
		latitude: 53.801181,
		longitude: -1.556016,
	}
)

/*
	Handler function for drinksfinder v1 API
*/
func drinksfinderV1Handler(w http.ResponseWriter, r *http.Request) {
	apiPath := strings.TrimPrefix(r.URL.Path, drinksfinderBasePath)
	qsParams := r.URL.Query()
	paginationParams := getPaginationParameters(qsParams)

	// Get "order_by" querystring param and validate against expected values
	order := qsParams.Get("order_by")
	if order != "" && !StringArrayContains(drinkfinderV1OrderFields, order) {
		setErrorResponse(&w,
			http.StatusBadRequest,
			fmt.Sprintf(
				"Unexpected value '%s' for \"order_by\": expected one of %v",
				order, drinkfinderV1OrderFields,
			),
		)
		return
	}

	// Get all "tag" querystring param values
	// Enhancement - validate against expected values
	tags := qsParams["tag"]

	// Build the search string to send to the data access layer
	dbLookupParams := make(map[string]interface{})
	if apiPath == "/pubs" {
		// Get a list of pubs
		dbLookupParams["topic"] = "pubs"

		// Optionally request a sort order
		if order != "" {
			dbLookupParams["order"] = order
		}

	} else if apiPath == "/pubs/near" {
		// If employee is at the office, they can request proximity to the
		// hard-coded long/lat
		dbLookupParams["longlat"] = []float64{
			xLabOfficeLocation.longitude, xLabOfficeLocation.latitude,
		}

	} else if strings.HasPrefix(apiPath, "/pubs/near/postcode/") {
		// Employee might not be at the office, so let them search for proximity
		// to another location using a postcode.
		// N.B. This will only work with a valid Google Geocoding API key.
		if googleApiKey == "" {
			setErrorResponse(
				&w, http.StatusNotImplemented, "Postcode search not available",
			)
			return
		}

		// Get the postcode from the URL path
		postcode := strings.TrimPrefix(apiPath, "/pubs/near/postcode/")
		if pc, err := url.PathUnescape(postcode); err != nil {
			setErrorResponse(&w,
				http.StatusBadRequest,
				fmt.Sprintf("Invalid postcode '%s'", postcode),
			)
			return
		} else {
			postcode = strings.TrimSpace(pc)
		}

		// Validate postcode
		if ! postcodeValidation.Match([]byte(postcode)) {
			setErrorResponse(&w,
				http.StatusBadRequest,
				fmt.Sprintf("Invalid postcode '%s'", postcode),
			)
			return
		}

		// Call out to Google Geocoding API to convert postcode to long/lat
		long, lat, err := getLongLatFromPostcode(postcode)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		dbLookupParams["longlat"] = []float64{long, lat}
	} else {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if len(tags) > 0 {
		dbLookupParams["tags"] = tags
	}

	dbLookupString, err := json.Marshal(dbLookupParams)
	if err != nil {
		// Don't send an error message as we don't want to leak any information
		// that could be used as an attack vector
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Request data from the data access layer
	request, err := http.NewRequest(
		"PUT", dataAccessUrl, bytes.NewBuffer(dbLookupString),
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	// Read and process response from data access layer
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	records := []map[string]interface{}{}
	if err = json.Unmarshal(body, &records); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	count := len(records)
	total := count

	if count > 0 {
		// Apply pagination settings
		startIndex := 0
		endIndex := len(records) - 1
		if start, found := (*paginationParams)["start"]; found {
			startIndex = start - 1
		}
		if limit, found := (*paginationParams)["limit"]; found {
			if end := startIndex + limit;  end < endIndex {
				endIndex = end
			}
		}
		records = records[startIndex:endIndex]
		count = endIndex - startIndex
	}

	// Generate response body
	responseBody, err := json.Marshal(map[string]interface{}{
		"results": records,
		"count": count,
		"total": total,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Calculate checksum to facilitate caching via ETags
	md5sum := md5.Sum(responseBody)
	checksum := hex.EncodeToString(md5sum[:])
	if (*r).Header.Get("If-None-Match") == checksum {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	// Write successful response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", checksum)
	fmt.Fprintf(w, string(responseBody))
}
