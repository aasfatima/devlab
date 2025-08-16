package main

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Create a new token object, specifying signing method and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "demo-user",
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Expires in 24 hours
		"iat":     time.Now().Unix(),
		"iss":     "devlab",
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte("devlab_secret"))
	if err != nil {
		log.Fatal("Error creating token:", err)
	}

	fmt.Println("Valid JWT Token for DevLab API:")
	fmt.Println("==================================")
	fmt.Println(tokenString)
	fmt.Println("==================================")
	fmt.Println("\nUsage:")
	fmt.Println("curl -X POST http://localhost:8000/scenarios/start \\")
	fmt.Println("  -H \"Content-Type: application/json\" \\")
	fmt.Println("  -H \"Authorization: Bearer " + tokenString + "\" \\")
	fmt.Println("  -d '{\"user_id\": \"demo-user\", \"scenario_type\": \"go\"}'")
}
