package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

func toWeatherResponse(src GoogleWeatherResponse) WeatherResponse {
	result := WeatherResponse{
		ForecastHours: make([]ForecastHour, 0, len(src.ForecastHours)),
	}

	for _, hour := range src.ForecastHours {
		result.ForecastHours = append(result.ForecastHours, ForecastHour{
			StartTime: hour.Interval.StartTime,
			DisplayDateTime: DisplayDateTime{
				Year:      hour.DisplayDateTime.Year,
				Month:     hour.DisplayDateTime.Month,
				Day:       hour.DisplayDateTime.Day,
				Hour:      hour.DisplayDateTime.Hours,
				Minute:    hour.DisplayDateTime.Minutes,
				UTCOffset: hour.DisplayDateTime.UTCOffset,
			},
			Temperature: TemperatureValue{
				Celsius: hour.Temperature.Degrees,
			},
			FeelsLikeTemperature: TemperatureValue{
				Celsius: hour.FeelsLikeTemperature.Degrees,
			},
			Humidity: PercentValue{
				Percent: hour.RelativeHumidity,
			},
			RainChance: PercentValue{
				Percent: hour.Precipitation.Probability.Percent,
			},
		})
	}

	return result
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	apiKey := os.Getenv("GOOGLE_WEATHER_API_KEY")
	latitude := os.Getenv("LATITUDE")
	longitude := os.Getenv("LONGITUDE")
	password := os.Getenv("PASSWORD")

	http.HandleFunc("/weather", func(w http.ResponseWriter, r *http.Request) {
		// Check password query parameter
		pass := r.URL.Query().Get("pass")
		if pass != password {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Create a new HTTP client with a timeout
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		// Make a request to the Google Weather API
		resp, err := client.Get(fmt.Sprintf("https://weather.googleapis.com/v1/forecast/hours:lookup?key=%s&location.latitude=%s&location.longitude=%s&hours=%d", apiKey, latitude, longitude, 24))
		if err != nil {
			http.Error(w, "Failed to fetch weather data", http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read weather data", http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			http.Error(w, string(body), resp.StatusCode)
			return
		}

		var sourceWeather GoogleWeatherResponse
		if err := json.Unmarshal(body, &sourceWeather); err != nil {
			http.Error(w, "Failed to parse weather data", http.StatusInternalServerError)
			return
		}

		weather := toWeatherResponse(sourceWeather)

		// Return the parsed WeatherResponse as JSON.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(weather); err != nil {
			http.Error(w, "Failed to encode weather data", http.StatusInternalServerError)
			return
		}

	})

	http.HandleFunc("/aqi", func(w http.ResponseWriter, r *http.Request) {
		// Check password query parameter
		pass := r.URL.Query().Get("pass")
		if pass != password {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Create a new HTTP client with a timeout
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resBody := GoogleAqiRequestBody{}
		lat, err := strconv.ParseFloat(latitude, 64)
		if err != nil {
			http.Error(w, "Invalid latitude value", http.StatusBadRequest)
			return
		}
		resBody.Location.Latitude = lat
		long, err := strconv.ParseFloat(longitude, 64)
		if err != nil {
			http.Error(w, "Invalid longitude value", http.StatusBadRequest)
			return
		}
		resBody.Location.Longitude = long

		jsonBody, err := json.Marshal(resBody)
		if err != nil {
			http.Error(w, "Failed to marshal AQI request body", http.StatusInternalServerError)
			return
		}

		resp, err := client.Post(fmt.Sprintf("https://airquality.googleapis.com/v1/currentConditions:lookup?key=%s", apiKey), "application/json", bytes.NewBuffer(jsonBody))
		if err != nil {
			http.Error(w, "Failed to fetch AQI data", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read AQI data", http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			http.Error(w, string(body), resp.StatusCode)
			return
		}

		var sourceAqi GoogleAqi
		if err := json.Unmarshal(body, &sourceAqi); err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse AQI data: %v", err), http.StatusInternalServerError)
			return
		}

		if len(sourceAqi.Indexes) == 0 {
			http.Error(w, "AQI data missing indexes", http.StatusInternalServerError)
			return
		}

		// Return the parsed AQI data as JSON.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(sourceAqi.Indexes[0].Aqi); err != nil {
			http.Error(w, "Failed to encode AQI data", http.StatusInternalServerError)
			return
		}
	})

	// Define a simple route
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Status: OK")
	})

	fmt.Println("🚀 Server starting on http://localhost:8080")

	// Start the server
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
