package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	// AppEnv is the running environment (development/production).
	AppEnv string
	// ServerPort is the HTTP port to listen on.
	ServerPort string
	// MySQLDSN is the connection string for the MySQL database.
	MySQLDSN string
	// AWSRegion is the AWS region for S3 uploads.
	AWSRegion string
	// S3Bucket is the target S3 bucket name.
	S3Bucket string
	// S3Endpoint is an optional custom endpoint (for non-AWS S3 providers like MinIO/Contabo).
	S3Endpoint string
	// S3PathStyle enables path-style addressing (required for some S3 providers).
	S3PathStyle bool
	// StorageType determines where to save exports: "local" or "s3".
	StorageType string
	// LocalStoragePath is the directory for local exports.
	LocalStoragePath string
	// SMTP settings for email notifications.
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	// WorkerCount is the number of concurrent export jobs allowed.
	WorkerCount int
	// MaxDBConcurrency restricts the global number of concurrent DB connections.
	MaxDBConcurrency int64
	// DefaultTimeout is the maximum duration for an export job.
	DefaultTimeout time.Duration
	// ConfigCompression enables Gzip compression for exports.
	ConfigCompression bool
	// AttachFile enables sending the export as an email attachment (if small enough).
	AttachFile bool
	// APISecret is the shared secret for HMAC-SHA256 request signing.
	// APISecret is the shared secret for HMAC-SHA256 request signing.
	APISecret string
	// AllowedOrigins is a list of CORS allowed domains.
	AllowedOrigins []string
}

func Load() *Config {
	return &Config{
		AppEnv:            getEnv("APP_ENV", "development"),
		AllowedOrigins:    getEnvSlice("ALLOWED_ORIGINS", []string{"*"}),
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		MySQLDSN:          getEnv("MYSQL_DSN", "user:password@tcp(localhost:3306)/dbname?parseTime=true"),
		AWSRegion:         getEnv("AWS_REGION", "us-east-1"),
		S3Bucket:          getEnv("S3_BUCKET", "my-export-bucket"),
		S3Endpoint:        getEnv("S3_ENDPOINT", ""),
		S3PathStyle:       getEnvBool("S3_PATH_STYLE", false),
		StorageType:       getEnv("STORAGE_TYPE", "s3"),
		LocalStoragePath:  getEnv("LOCAL_STORAGE_PATH", "./exports"),
		SMTPHost:          getEnv("SMTP_HOST", ""),
		SMTPPort:          getEnvInt("SMTP_PORT", 587),
		SMTPUser:          getEnv("SMTP_USER", ""),
		SMTPPassword:      getEnv("SMTP_PASS", ""),
		SMTPFrom:          getEnv("SMTP_FROM", "noreply@example.com"),
		WorkerCount:       getEnvInt("WORKER_COUNT", 5),
		MaxDBConcurrency:  int64(getEnvInt("MAX_DB_CONCURRENCY", 3)),
		DefaultTimeout:    getEnvDuration("DEFAULT_TIMEOUT", 15*time.Minute),
		ConfigCompression: getEnvBool("COMPRESSION", false),
		AttachFile:        getEnvBool("EMAIL_ATTACH_FILE", false),
		APISecret:         getEnv("API_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	if value, ok := os.LookupEnv(key); ok {
		// detailed parsing could happen here, for now split by comma
		// basic implementation
		var result []string
		start := 0
		for i := 0; i < len(value); i++ {
			if value[i] == ',' {
				result = append(result, value[start:i])
				start = i + 1
			}
		}
		result = append(result, value[start:])
		return result
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return fallback
}
