package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type StartScenarioRequest struct {
	UserID       string `json:"user_id"`
	ScenarioType string `json:"scenario_type"`
	Script       string `json:"script"`
}

type StartScenarioResponse struct {
	ScenarioID string `json:"scenario_id"`
	Status     string `json:"status"`
}

func main() {
	url := "http://localhost:8000/scenarios/start"
	body := StartScenarioRequest{
		UserID:       "user1",
		ScenarioType: "go",
		Script:       "echo Hello from inside the container!",
	}
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	var out StartScenarioResponse
	json.Unmarshal(respBody, &out)
	fmt.Printf("Response: %+v\n", out)
}
