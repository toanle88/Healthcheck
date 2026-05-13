package main

import (
	"net/http"
	"os"
)

func main() {
	// We check the local API health endpoint
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}
