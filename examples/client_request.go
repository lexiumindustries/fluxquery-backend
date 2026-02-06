package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// This example shows how another server would securely call the Export API.
func main() {
	secret := "devsecret"
	url := "http://localhost:8080/export"
	body := `{"query":"SELECT * FROM users LIMIT 10", "email":"admin@example.com", "format":"json"}`

	// 1. Prepare Request
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	// 2. Generate Authentication Headers
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Payload for signature: Method + Path + Body + Timestamp
	payload := req.Method + req.URL.Path + body + timestamp

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))

	// 3. Set Headers
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", signature)

	// 4. Send Request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Printf("Response: %s\n", string(respBody))
}
