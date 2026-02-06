# Implementation Plan - MySQL Export Service

## Goal
Build a production-ready, high-performance MySQL export service in Go that streams data directly from MySQL to S3 via CSV/Gzip without disk buffering.

## Advanced Security Hardening: HMAC Authentication

### Goals
*   **Authenticity**: Ensure only authorized servers can trigger exports.
*   **Integrity**: Ensure the query/email haven't been tampered with in transit.
*   **Anti-Replay**: Prevent captured requests from being reuse (via Timestamp).

### Architecture
- **Worker Pool**: Managed concurrency with backpressure and DB semaphore protection.
- **Interfaces**: Pluggable `RowEncoder`, `StorageProvider`, and `EmailSender`.
- **HMAC Middleware**: Verifies `X-Signature` and `X-Timestamp` using `API_SECRET`.
