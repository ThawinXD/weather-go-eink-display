package main

import (
	"fmt"
	"net/http"
)

// RegisterRoutes registers the HTTP handlers for the server.
func RegisterRoutes(apiKey, latitude, longitude, password string) {
	http.HandleFunc("/weather", WeatherHandler(apiKey, latitude, longitude, password))
	http.HandleFunc("/aqi", AqiHandler(apiKey, latitude, longitude, password))

	// Define a simple route
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Status: OK")
	})
}
