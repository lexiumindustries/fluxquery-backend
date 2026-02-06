package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Println("Usage: go run sign_request.go <secret> <method> <path> <body>")
		fmt.Println("Example: go run sign_request.go mysecret POST /export '{\"query\":\"...\"}'")
		return
	}

	secret := os.Args[1]
	method := os.Args[2]
	path := os.Args[3]
	body := os.Args[4]
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	mac := hmac.New(sha256.New, []byte(secret))
	payload := method + path + body + timestamp
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	fmt.Printf("X-Timestamp: %s\n", timestamp)
	fmt.Printf("X-Signature: %s\n", signature)
}
