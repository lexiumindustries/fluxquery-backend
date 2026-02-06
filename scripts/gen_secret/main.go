package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	// Generate 32 bytes of secure random data (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Convert to a hex string (64 characters)
	secret := hex.EncodeToString(bytes)

	fmt.Println("=== New Secure Secret Generated ===")
	fmt.Println(secret)
	fmt.Println("=====================================")
	fmt.Println("1. Copy this secret to your .env or Secret Manager (API_SECRET=...)")
	fmt.Println("2. Provide this secret to the client service via a SECURE channel.")
	fmt.Println("3. DO NOT share this over Slack or Email without encryption.")
}
