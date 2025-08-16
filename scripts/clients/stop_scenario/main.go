package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type StopScenarioResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run stop_scenario_client.go <api_url> <scenario_id>")
		fmt.Println("Example: go run stop_scenario_client.go http://localhost:8000 scn-123")
		os.Exit(1)
	}

	apiURL := os.Args[1]
	scenarioID := os.Args[2]

	// Create the request URL
	url := fmt.Sprintf("%s/scenarios/%s", apiURL, scenarioID)

	// Create HTTP request
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	// Parse response
	var response StopScenarioResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("Error parsing response: %v\n", err)
		fmt.Printf("Raw response: %s\n", string(body))
		os.Exit(1)
	}

	// Print results
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response:\n")
	fmt.Printf("  Error: %s\n", response.Error)
	fmt.Printf("  Code: %s\n", response.Code)
	fmt.Printf("  Message: %s\n", response.Message)

	// Check if successful
	if resp.StatusCode == http.StatusOK && response.Code == "SUCCESS" {
		fmt.Println("\n✅ Scenario stopped successfully!")
	} else {
		fmt.Printf("\n❌ Failed to stop scenario (Status: %d)\n", resp.StatusCode)
		os.Exit(1)
	}
}
