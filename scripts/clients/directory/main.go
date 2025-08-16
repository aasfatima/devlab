package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type DirectoryStructureResponse struct {
	ScenarioID string     `json:"scenario_id"`
	Path       string     `json:"path"`
	Structure  []FileNode `json:"structure"`
	Message    string     `json:"message"`
}

type FileNode struct {
	Path     string   `json:"path"`
	Type     string   `json:"type"`
	IsRoot   bool     `json:"isRoot"`
	Children []string `json:"children,omitempty"`
	Content  string   `json:"content,omitempty"`
	IsOpen   bool     `json:"isOpen"`
	IsSaved  bool     `json:"isSaved"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run directory_client.go <api_url> <scenario_id>")
		fmt.Println("Example: go run directory_client.go http://localhost:8000 scn-123")
		os.Exit(1)
	}

	apiURL := os.Args[1]
	scenarioID := os.Args[2]

	// Test directory structure endpoint
	fmt.Printf("Getting directory structure for scenario: %s\n", scenarioID)

	url := fmt.Sprintf("%s/scenarios/%s/directory", apiURL, scenarioID)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))

	if resp.StatusCode == http.StatusOK {
		var dirResp DirectoryStructureResponse
		if err := json.Unmarshal(body, &dirResp); err != nil {
			fmt.Printf("Error parsing response: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nDirectory Structure for %s:\n", dirResp.ScenarioID)
		fmt.Printf("Path: %s\n", dirResp.Path)
		fmt.Printf("Message: %s\n", dirResp.Message)
		fmt.Printf("Number of items: %d\n", len(dirResp.Structure))

		for _, node := range dirResp.Structure {
			fmt.Printf("  - %s (%s)\n", node.Path, node.Type)
			if node.Type == "folder" && len(node.Children) > 0 {
				fmt.Printf("    Children: %v\n", node.Children)
			}
		}
	}
}
