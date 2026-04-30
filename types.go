package main

import "time"

type WeatherResponse struct {
	ForecastHours []ForecastHour `json:"forecastHours"`
}

type ForecastHour struct {
	StartTime            time.Time        `json:"startTime"`
	DisplayDateTime      DisplayDateTime  `json:"displayDateTime"`
	Temperature          TemperatureValue `json:"temperature"`
	FeelsLikeTemperature TemperatureValue `json:"feelsLikeTemperature"`
	Humidity             PercentValue     `json:"humidity"`
	RainChance           PercentValue     `json:"rainChance"`
}

type DisplayDateTime struct {
	Year      int    `json:"year"`
	Month     int    `json:"month"`
	Day       int    `json:"day"`
	Hour      int    `json:"hour"`
	Minute    int    `json:"minute"`
	UTCOffset string `json:"utcOffset"`
}

type TemperatureValue struct {
	Celsius float64 `json:"celsius"`
}

type PercentValue struct {
	Percent int `json:"percent"`
}

type GoogleWeatherResponse struct {
	ForecastHours []GoogleForecastHour `json:"forecastHours"`
}

type GoogleForecastHour struct {
	Interval struct {
		StartTime time.Time `json:"startTime"`
	} `json:"interval"`
	DisplayDateTime struct {
		Year      int    `json:"year"`
		Month     int    `json:"month"`
		Day       int    `json:"day"`
		Hours     int    `json:"hours"`
		Minutes   int    `json:"minutes"`
		UTCOffset string `json:"utcOffset"`
	} `json:"displayDateTime"`
	Temperature struct {
		Degrees float64 `json:"degrees"`
	} `json:"temperature"`
	FeelsLikeTemperature struct {
		Degrees float64 `json:"degrees"`
	} `json:"feelsLikeTemperature"`
	RelativeHumidity int `json:"relativeHumidity"`
	Precipitation    struct {
		Probability struct {
			Percent int `json:"percent"`
		} `json:"probability"`
	} `json:"precipitation"`
}

type GoogleAqiRequestBody struct {
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
}

type GoogleAqi struct {
	Indexes []struct {
		Aqi int `json:"aqi"`
	} `json:"indexes"`
}
