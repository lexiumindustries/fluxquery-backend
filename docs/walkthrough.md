# FluxQuery Export Service Walkthrough

## 1. Local Database Emulator
The service uses `go-mysql-server` to emulate a MySQL database locally without needing Docker.

### Capabilities
- **In-Memory**: Fast, no persistence (resets on restart).
- **Seeding**: Automatically seeds 1,000,000 rows on startup.
- **Schema**: `users` table (`id`, `name`, `email`, `created_at`, `score`).

### Running the DB
```powershell
.\run_db.bat
```

## 2. Export Service
The main service connects to the local DB and exports data to Local Storage or S3.

### Features
- **Streaming**: Low memory usage (alloc-free CSV encoding).
- **Multi-Format**: CSV, JSON, Excel (.xlsx).
- **Security**: HMAC-SHA256 request signing required for all exports.
- **Memory Safety**: 25MB attachment limit.

### Running the Service
```powershell
.\run.bat
```

## 3. Usage & Examples

### Triggering an Export
Because the server requires HMAC signing, use the provided helper scripts or the Go/TS examples.

**Using PowerShell Helper:**
```powershell
$body = Get-Content scripts/payloads/benchmarks/export_request.json -Raw; $secret = "devsecret"; $ts = [DateTimeOffset]::Now.ToUnixTimeSeconds().ToString(); $payload = "POST" + "/export" + $body + $ts; $hmac = New-Object System.Security.Cryptography.HMACSHA256; $hmac.Key = [Text.Encoding]::UTF8.GetBytes($secret); $sig = -join ($hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($payload)) | ForEach-Object { "{0:x2}" -f $_ }); Invoke-RestMethod -Uri "http://localhost:8080/export" -Method Post -ContentType "application/json" -Headers @{"X-Timestamp"=$ts; "X-Signature"=$sig} -Body $body
```

**Client Examples:**
- [Go Example](../examples/client_request.go)
- [TypeScript Example](../examples/client_request.ts)
- [PHP Example](../examples/client_request.php)

## 4. Environment Configuration (Dev vs Prod)

| Feature | Development (Local) | Production |
| :--- | :--- | :--- |
| `APP_ENV` | `development` | `production` |
| `STORAGE_TYPE` | `local` (./exports) | `s3` (AWS S3) |
| `API_SECRET` | `devsecret` | High-entropy random hex string |
| `COMPRESSION` | `false` | `true` |

## 5. Production Security Guide

### Secret Generation
```powershell
go run scripts/gen_secret/main.go
```

### Security Checklist
- [x] **SQLi Protected**: Strict `SELECT` whitelist.
- [x] **Information Leakage Protected**: System tables blocked.
- [x] **Formula Injection Protected**: Malicious CSV/Excel cells sanitized.
- [x] **Authenticity Verified**: HMAC-SHA256 required.
- [x] **Replay Protected**: 5-minute request window.
