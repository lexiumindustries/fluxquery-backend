package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid request signature")
	ErrRequestExpired   = errors.New("request timestamp expired or too far in future")
)

// VerifyHMAC verifies the authenticity and integrity of a request using HMAC-SHA256.
// It constructs the expected signature using the shared secret and the payload (Method + Path + Body + Timestamp)
// and compares it with the provided signature in constant time.
//
// Arguments:
//   - secret: The shared secret key (API_SECRET).
//   - method: HTTP method (e.g., "POST").
//   - path: Request URL path (e.g., "/export").
//   - body: Raw request body content.
//   - timestamp: Unix timestamp string from X-Timestamp header.
//   - signature: Hex-encoded HMAC signature from X-Signature header.
//
// Returns error if signature is invalid, timestamp is expired, or format is wrong.
func VerifyHMAC(secret, method, path, body, timestamp, signature string) error {
	if secret == "" {
		return nil // No secret configured, skip auth (useful for local dev if desired)
	}

	// 1. Check timestamp drift (replay protection)
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	now := time.Now().Unix()
	drift := now - ts
	if drift < -300 || drift > 300 { // 5-minute window
		return ErrRequestExpired
	}

	// 2. Compute expected signature
	mac := hmac.New(sha256.New, []byte(secret))
	payload := method + path + body + timestamp
	mac.Write([]byte(payload))
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	// 3. Constant time comparison to prevent timing attacks
	if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
		return ErrInvalidSignature
	}

	return nil
}
