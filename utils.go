package main

import "time"

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

// generateFixedWindow generates 38 hourly slots from 11:00 previous day to 23:00 same day (shifted 7 hours left)
func generateFixedWindow(currentTime time.Time) []time.Time {
	utc := currentTime.UTC()
	// Truncate to hour boundary (remove minutes, seconds, nanoseconds)
	truncated := time.Date(utc.Year(), utc.Month(), utc.Day(), utc.Hour(), 0, 0, 0, utc.Location())
	// Go back (13 + current_hour) hours to reach 11:00 (11 AM) of the previous day
	start := truncated.Add(-time.Duration(13+truncated.Hour()) * time.Hour)

	var slots []time.Time
	for i := 0; i < 38; i++ {
		slots = append(slots, start.Add(time.Duration(i)*time.Hour))
	}
	return slots
}
