package main

import (
	"fmt"
	"net/http"
	"os"
)

// main is the entrypoint for the healthcheck CLI/utility, verifying the API's local health status.
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := checkHealth(port); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

// checkHealth queries the local /health endpoint to ensure the API service is up and running.
func checkHealth(port string) error {
	// We check the local API health endpoint
	resp, err := http.Get("http://localhost:" + port + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
