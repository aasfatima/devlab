package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	pb "devlab/proto"
)

func main() {
	conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewScenarioServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := client.StartScenario(ctx, &pb.StartScenarioRequest{
		UserId:       "user1",
		ScenarioType: "go",
		Script:       "echo Hello from gRPC!",
	})
	if err != nil {
		log.Fatalf("StartScenario error: %v", err)
	}
	fmt.Printf("Response: %+v\n", resp)
}
