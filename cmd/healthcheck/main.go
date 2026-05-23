package main

import (
	"fmt"
	"net/http"
	"os"
)

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
