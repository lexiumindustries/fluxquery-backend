package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	TotalRequests int
	Concurrency   int
	Format        string
	Limit         int
	Description   string
}

type Result struct {
	JobID          string
	Status         int
	AcceptDuration time.Duration
	TotalDuration  time.Duration // Time until job "COMPLETED"
	Error          error
}

func main() {
	// Define Scenarios
	scenarios := []Config{
		{TotalRequests: 50, Concurrency: 10, Format: "pdf", Limit: 100, Description: "Baseline (Low Load)"},
		{TotalRequests: 100, Concurrency: 50, Format: "csv", Limit: 2000, Description: "Stress Test (High Concurrency)"},
		{
			TotalRequests: 5,
			Concurrency:   2,
			Format:        "pdf",
			Limit:         100000,
			Description:   "Complex JOIN (Users + Transactions) - 100k rows",
		},
	}

	for _, scenario := range scenarios {
		runScenario(scenario)
	}
}

func runScenario(cfg Config) {
	fmt.Printf("\n=======================================================\n")
	fmt.Printf("Scenario: %s\n", cfg.Description)
	fmt.Printf("Requests: %d | Concurrency: %d | Format: %s | Limit: %d\n", cfg.TotalRequests, cfg.Concurrency, cfg.Format, cfg.Limit)
	fmt.Printf("=======================================================\n")

	results := make(chan Result, cfg.TotalRequests)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cfg.Concurrency) // Limit concurrency

	startTime := time.Now()

	for i := 0; i < cfg.TotalRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			res := executeRequest(cfg)
			results <- res

			// Simple progress bar
			if id%10 == 0 {
				fmt.Print(".")
			}
		}(i)
	}

	wg.Wait()
	close(results)
	totalTime := time.Since(startTime)
	fmt.Println()

	// Analyze Results
	var acceptLatencies []time.Duration
	var processLatencies []time.Duration
	var failures int

	for res := range results {
		if res.Error != nil || res.Status != 202 {
			failures++
		} else {
			acceptLatencies = append(acceptLatencies, res.AcceptDuration)
			if res.TotalDuration > 0 {
				processLatencies = append(processLatencies, res.TotalDuration)
			}
		}
	}

	sort.Slice(acceptLatencies, func(i, j int) bool { return acceptLatencies[i] < acceptLatencies[j] })
	sort.Slice(processLatencies, func(i, j int) bool { return processLatencies[i] < processLatencies[j] })

	// Report
	fmt.Printf("\nRESULTS:\n")
	fmt.Printf("Total Duration: %v\n", totalTime)
	fmt.Printf("Throughput: %.2f req/sec\n", float64(cfg.TotalRequests)/totalTime.Seconds())
	fmt.Printf("Success Rate: %.1f%%\n", float64(cfg.TotalRequests-failures)/float64(cfg.TotalRequests)*100)

	if len(acceptLatencies) > 0 {
		fmt.Printf("API Response Time (P95): %v\n", acceptLatencies[int(float64(len(acceptLatencies))*0.95)])
	}
	if len(processLatencies) > 0 {
		fmt.Printf("Job Completion Time (P95): %v\n", processLatencies[int(float64(len(processLatencies))*0.95)])
	}
}

func executeRequest(cfg Config) Result {
	secret := "devsecret"
	baseURL := "http://localhost:8080"

	start := time.Now()

	// 1. Submit Job
	query := fmt.Sprintf("SELECT id, name, created_at FROM users LIMIT %d", cfg.Limit)
	if strings.Contains(cfg.Description, "JOIN") {
		query = fmt.Sprintf(`
			SELECT u.name, u.email, t.amount, t.currency, t.created_at 
			FROM users u 
			JOIN transactions t ON u.id = t.user_id 
			LIMIT %d`, cfg.Limit)
	}

	payload := map[string]string{
		"query":  query,
		"email":  "benchmark@example.com",
		"format": cfg.Format,
	}
	bodyBytes, _ := json.Marshal(payload)
	body := string(bodyBytes)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// Sign
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("POST" + "/export" + body + timestamp))
	signature := hex.EncodeToString(mac.Sum(nil))

	req, _ := http.NewRequest("POST", baseURL+"/export", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", signature)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Result{Error: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		return Result{Status: resp.StatusCode, AcceptDuration: time.Since(start)}
	}

	var respJson map[string]string
	json.NewDecoder(resp.Body).Decode(&respJson)
	jobID := respJson["job_id"]
	acceptTime := time.Since(start)

	// 2. Poll for Completion
	// Poll every 500ms
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(300 * time.Second) // Increased timeout for heavy joins

	for {
		select {
		case <-timeout:
			return Result{JobID: jobID, Status: 202, AcceptDuration: acceptTime, Error: fmt.Errorf("timeout waiting for job")}
		case <-ticker.C:
			_, finished, err := checkStatus(baseURL, jobID, secret)
			if err != nil {
				continue // Retry on temp error
			}
			if finished {
				return Result{
					JobID:          jobID,
					Status:         202,
					AcceptDuration: acceptTime,
					TotalDuration:  time.Since(start),
				}
			}
			// If status is PENDING or PROCESSING, continue
		}
	}
}

func checkStatus(baseURL, jobID, secret string) (string, bool, error) {
	req, _ := http.NewRequest("GET", baseURL+"/jobs?id="+jobID, nil)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", false, fmt.Errorf("status check failed: %d", resp.StatusCode)
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", false, err
	}

	status := data["status"].(string)
	if status == "COMPLETED" || status == "FAILED" {
		return status, true, nil
	}
	return status, false, nil
}
