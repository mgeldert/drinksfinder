package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	googleApiRequest = "https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s"
	googleApiKey = ""
)

func init() {
	apiKey, _ := ioutil.ReadFile("/settings/google_api_key")
	googleApiKey = string(apiKey)
}

func getLongLatFromPostcode(postcode string) (float64, float64, error) {

	response, err := http.Get(fmt.Sprintf(googleApiRequest, postcode, googleApiKey))
	if err != nil {
		return 0.0, 0.0, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0.0, 0.0, err
	}

	data := map[string]interface{}{}
	if err = json.Unmarshal(body, &data); err != nil {
		return 0.0, 0.0, err
	}

	if status, found := data["status"]; !found || status != "OK" {
		return 0.0, 0.0, fmt.Errorf("Postcode lookup failed")
	}

	results := data["results"].([]interface{})
	if len(results) == 0 {
		return 0.0, 0.0, fmt.Errorf("Postcode lookup failed")
	}

	firstResult := results[0].(map[string]interface{})
	geometry := firstResult["geometry"].(map[string]interface{})
	location := geometry["location"].(map[string]interface{})
	longitude := location["lng"].(float64)
	latitude := location["lat"].(float64)

	return longitude, latitude, nil
}
