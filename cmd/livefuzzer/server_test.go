package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestServerEndpoints(t *testing.T) {
	serverMutex = sync.Mutex{} // Ensure clean mutex for tests
	running = false
	cancelFunc = nil

	mux := http.NewServeMux()
	serverCommand.Action(nil) // Initialize server with endpoints
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("Health Check", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to make GET request: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 OK, got %v", resp.StatusCode)
		}
		if string(body) != "200 OK" {
			t.Errorf("Expected body '200 OK', got %v", string(body))
		}
	})

	t.Run("Start Spam", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/spam/start", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make POST request: %v", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 OK, got %v", resp.StatusCode)
		}
		if string(body) != "Spam started" {
			t.Errorf("Expected body 'Spam started', got %v", string(body))
		}
	})

	t.Run("Start Spam Again", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/spam/start", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make POST request: %v", err)
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("Expected status 409 Conflict, got %v", resp.StatusCode)
		}
		if string(body) != "Spam already running" {
			t.Errorf("Expected body 'Spam already running', got %v", string(body))
		}
	})

	t.Run("Stop Spam", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/spam/stop", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make POST request: %v", err)
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200 OK, got %v", resp.StatusCode)
		}
		if string(body) != "Spam stopped" {
			t.Errorf("Expected body 'Spam stopped', got %v", string(body))
		}
	})

	t.Run("Stop Spam Again", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/spam/stop", "application/json", nil)
		if err != nil {
			t.Fatalf("Failed to make POST request: %v", err)
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected status 400 Bad Request, got %v", resp.StatusCode)
		}
		if string(body) != "No spam running" {
			t.Errorf("Expected body 'No spam running', got %v", string(body))
		}
	})
}
