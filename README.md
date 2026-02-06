# FluxQuery MySQL Export Service

A service for exporting data from MySQL to various formats.

- **Streaming Pipeline**: MySQL -> Encoder -> [Gzip] -> Storage. Zero disk buffering ensures speed and low RAM overhead.
- **Multi-Format Support**: Export to **CSV**, **JSON (Lines)**, and **Excel (.xlsx)**.
- **Dual Storage**: Seamlessly switch between **Local Storage** (for development) and **AWS S3** (for production).
- **Advanced Authentication**: Secure server-to-server communication via **HMAC-SHA256 Request Signing**.
- **Hardened Security**:
    - **SQLi Protection**: Strict `SELECT` whitelist and forbidden keyword detection.
    - **Information Leakage**: Blocks access to system tables (`information_schema`, etc.).
    - **Formula Injection**: Sanitizes malicious spreadsheet formulas.
    - **Memory Safety**: 25MB attachment limit to prevent OOM crashes.
- **Email Integration**: Automated notifications with attachments or download links.
- **Production Ready**: Multi-stage Docker support and environment-based configuration.

## Architecture

- **Go 1.23**: Modern, type-safe implementation.
- **Worker Pool**: Managed concurrency with backpressure and DB semaphore protection.
- **Interfaces**: Pluggable `RowEncoder`, `StorageProvider`, and `EmailSender`.

## Configuration

Set up your environment using `.env` (Development) or Environment Variables (Production). See [`.env.example`](./.env.example) for details.

| Variable | Description |
|----------|-------------|
| `APP_ENV` | `development` or `production` |
| `SERVER_PORT` | HTTP Server Port (default: 8080) |
| `MYSQL_DSN` | MySQL Connection String |
| `STORAGE_TYPE` | `local` (writes to `./exports`) or `s3` |
| `API_SECRET` | Used for HMAC signing. Keep this private! |
| `COMPRESSION` | Enable/Disable Gzip compression. |
| `EMAIL_ATTACH_FILE` | Enable/Disable file attachments in emails. |

## Usage

### Local Development
1. Start the DB Emulator: `.\run_db.bat`
2. Start the Service: `.\run.bat`

### Authentication (HMAC Signing)
All requests to `/export` must be signed. Use the provided tools:
- **Go Helper**: `go run scripts/sign_request/main.go`
- **Secret Generator**: `go run scripts/gen_secret/main.go`
- **Examples**: Check the [`/examples`](./examples) folder for Go, TS, and PHP implementations.

### Sample Request (Signed)
```powershell
$body = '{"query":"SELECT * FROM users LIMIT 10", "email":"admin@example.com", "format":"csv"}';
# See walkthrough for full signing command...
```

## Docker

Build the lightweight production image:
```bash
docker build -t fluxquery-export .
```

Run the container:
```bash
docker run -p 8080:8080 --env-file .env fluxquery-export
```

## Documentation
- [User Walkthrough](./docs/walkthrough.md)
- [Security Verification Guide](./docs/verification_guide.md)
- [Implementation Plan](./docs/implementation_plan.md)
- [Security Audit Report](./docs/security_audit.md)

---
Developed for the **FluxQuery** ecosystem.
