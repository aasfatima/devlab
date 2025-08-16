package main

import (
	"context"
	pb "devlab/proto"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// REST client for Status API
func getScenarioStatusREST(scenarioID string) error {
	url := fmt.Sprintf("http://localhost:8000/scenarios/%s/status", scenarioID)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	fmt.Printf("REST Status Response (HTTP %d):\n", resp.StatusCode)
	fmt.Println(string(body))
	return nil
}

// gRPC client for Status API
func getScenarioStatusGRPC(scenarioID string) error {
	conn, err := grpc.Dial("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	client := pb.NewScenarioServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	resp, err := client.GetScenarioStatus(ctx, &pb.GetScenarioStatusRequest{
		ScenarioId: scenarioID,
	})
	if err != nil {
		return fmt.Errorf("failed to get scenario status: %w", err)
	}

	// Pretty print the response
	jsonResp, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("gRPC Status Response:\n%s\n", string(jsonResp))
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run status_client.go <scenario_id>")
		fmt.Println("Example: go run status_client.go scn-1234567890")
		return
	}

	scenarioID := os.Args[1]

	fmt.Printf("Getting status for scenario: %s\n\n", scenarioID)

	// Test REST API
	fmt.Println("=== REST API ===")
	if err := getScenarioStatusREST(scenarioID); err != nil {
		fmt.Printf("REST API error: %v\n", err)
	}

	fmt.Println()

	// Test gRPC API
	fmt.Println("=== gRPC API ===")
	if err := getScenarioStatusGRPC(scenarioID); err != nil {
		fmt.Printf("gRPC API error: %v\n", err)
	}
}
