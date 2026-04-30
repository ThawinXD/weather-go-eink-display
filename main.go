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

	"github.com/beevik/ntp"
	"github.com/joho/godotenv"
)

func generateFixedWindow(currentTime time.Time) []time.Time {
	// Generate 38 hourly slots from 11:00 previous day to 23:00 same day (shifted 7 hours left)
	utc := currentTime.UTC()
	// Truncate to hour boundary (remove minutes, seconds, nanoseconds)
	truncated := time.Date(utc.Year(), utc.Month(), utc.Day(), utc.Hour(), 0, 0, 0, utc.Location())
	// Go back (13 + current_hour) hours to reach 11:00 (11 AM) of the previous day (6 PM - 7 hours = 11 AM)
	start := truncated.Add(-time.Duration(13+truncated.Hour()) * time.Hour)

	var slots []time.Time
	for i := 0; i < 38; i++ {
		slots = append(slots, start.Add(time.Duration(i)*time.Hour))
	}
	return slots
}

func toWeatherResponse(hours []GoogleForecastHour) WeatherResponse {
	result := WeatherResponse{
		ForecastHours: make([]ForecastHour, 0, len(hours)),
	}

	for _, hour := range hours {
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

		t, err := ntp.Time("time.google.com")
		if err != nil {
			http.Error(w, "Failed to synchronize time with NTP server", http.StatusInternalServerError)
			return
		}

		currHour := t.Hour()

		// Create a new HTTP client with a timeout
		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		client2 := &http.Client{
			Timeout: 10 * time.Second,
		}

		// Make a request to the Google Weather API
		resp, err := client.Get(fmt.Sprintf("https://weather.googleapis.com/v1/forecast/hours:lookup?key=%s&location.latitude=%s&location.longitude=%s&hours=%d", apiKey, latitude, longitude, 31-currHour))
		if err != nil {
			log.Printf("Error fetching weather data: %v", err)
			http.Error(w, "Failed to fetch weather data", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		resp2, err := client2.Get(fmt.Sprintf("https://weather.googleapis.com/v1/history/hours:lookup?key=%s&location.latitude=%s&location.longitude=%s&hours=%d", apiKey, latitude, longitude, min(currHour+7, 24)))
		if err != nil {
			log.Printf("Error fetching weather data for past 7 hours: %v", err)
			http.Error(w, "Failed to fetch weather data for past 7 hours", http.StatusInternalServerError)
			return
		}
		defer resp2.Body.Close()

		// Read the response body
		body1, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading weather data: %v", err)
			http.Error(w, "Failed to read weather data", http.StatusInternalServerError)
			return
		}
		body2, err := io.ReadAll(resp2.Body)
		if err != nil {
			log.Printf("Error reading weather data for past 7 hours: %v", err)
			http.Error(w, "Failed to read weather data for past 7 hours", http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error fetching weather data: %v", resp.StatusCode)
			http.Error(w, string(body1), resp.StatusCode)
			return
		}
		if resp2.StatusCode != http.StatusOK {
			log.Printf("Error fetching weather data for past 7 hours: %v", resp2.StatusCode)
			http.Error(w, string(body2), resp2.StatusCode)
			return
		}

		var sourceWeather1 GoogleWeatherResponse
		if err := json.Unmarshal(body1, &sourceWeather1); err != nil {
			log.Printf("Error parsing weather data: %v", err)
			http.Error(w, "Failed to parse weather data", http.StatusInternalServerError)
			return
		}
		var sourceWeather2 GoogleHistoryResponse
		if err := json.Unmarshal(body2, &sourceWeather2); err != nil {
			log.Printf("Error parsing weather data for past 7 hours: %v", err)
			http.Error(w, "Failed to parse weather data", http.StatusInternalServerError)
			return
		}

		// Generate fixed 38-hour window from 18:00 previous day to 06:00 next day
		fixedWindow := generateFixedWindow(t)

		// Merge forecast and history data into the fixed window
		mergedHours := make(map[time.Time]ForecastHour)
		var utcOffsetStr string
		for _, hour := range sourceWeather1.ForecastHours {
			utcOffsetStr = hour.DisplayDateTime.UTCOffset
			break
		}
		if utcOffsetStr == "" {
			for _, hour := range sourceWeather2.HistoryHours {
				utcOffsetStr = hour.DisplayDateTime.UTCOffset
				break
			}
		}

		for _, hour := range toWeatherResponse(sourceWeather1.ForecastHours).ForecastHours {
			mergedHours[hour.StartTime] = hour
		}
		for _, hour := range toWeatherResponse(sourceWeather2.HistoryHours).ForecastHours {
			mergedHours[hour.StartTime] = hour
		}

		// Parse UTC offset
		utcOffset := time.Duration(0)
		if utcOffsetStr != "" {
			var offsetSeconds int64
			fmt.Sscanf(utcOffsetStr, "%ds", &offsetSeconds)
			utcOffset = time.Duration(offsetSeconds) * time.Second
		}

		// Fill the fixed window with data or -1 placeholders
		var filledHours []ForecastHour
		for _, slot := range fixedWindow {
			if data, exists := mergedHours[slot]; exists {
				filledHours = append(filledHours, data)
			} else {
				// Create placeholder with -1 for missing data
				localTime := slot.Add(utcOffset)
				filledHours = append(filledHours, ForecastHour{
					StartTime: slot,
					DisplayDateTime: DisplayDateTime{
						Year:      localTime.Year(),
						Month:     int(localTime.Month()),
						Day:       localTime.Day(),
						Hour:      localTime.Hour(),
						Minute:    0,
						UTCOffset: utcOffsetStr,
					},
					Temperature: TemperatureValue{
						Celsius: -1,
					},
					FeelsLikeTemperature: TemperatureValue{
						Celsius: -1,
					},
					Humidity: PercentValue{
						Percent: -1,
					},
					RainChance: PercentValue{
						Percent: -1,
					},
				})
			}
		}

		weather := WeatherResponse{
			ForecastHours: filledHours,
		}

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
