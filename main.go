package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables (optional .env)
	_ = godotenv.Load()

	apiKey := os.Getenv("API_KEY")
	latitude := os.Getenv("LATITUDE")
	longitude := os.Getenv("LONGITUDE")
	password := os.Getenv("PASSWORD")

	// Register handlers from routes.go
	RegisterRoutes(apiKey, latitude, longitude, password)

	// Simple health route (can be moved into RegisterRoutes)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Status: OK")
	})

	// Determine port (Render and many platforms set PORT environment variable)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	fmt.Printf("🚀 Server starting on http://localhost:%s\n", port)

	// Start the server
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
