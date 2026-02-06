# Security Verification Guide

Use these commands to verify that the hardening measures are working as expected.

## 1. SQL Injection Protection
**Malicious Query**: `DROP TABLE users`
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/export" -Method Post -ContentType "application/json" -InFile "scripts/payloads/security/hack_drop.json"
```
*   **Expected Result**: `400 Bad Request` - "only SELECT queries are allowed"

## 2. Information Disclosure Protection
**Malicious Query**: `SELECT * FROM information_schema.tables`
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/export" -Method Post -ContentType "application/json" -InFile "scripts/payloads/security/hack_sys_table.json"
```
*   **Expected Result**: `400 Bad Request` - "access to system table blocked: INFORMATION_SCHEMA"

## 3. SMTP Header Injection Protection
**Malicious Email**: `email: "scottlexium@gmail.com\r\nBcc:hacker@evil.com"`
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/export" -Method Post -ContentType "application/json" -InFile "scripts/payloads/security/hack_email.json"
```
*   **Expected Result**: `400 Bad Request` - "Email validation failed: invalid email address format"

## 4. Memory Safety (OOM Protection)
**Test**: Export 1 Million rows with `EMAIL_ATTACH_FILE=true` and `COMPRESSION=false`.
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/export" -Method Post -ContentType "application/json" -InFile "scripts/payloads/benchmarks/full_csv.json"
```
*   **Expected Result**: Server processes the job successfully but sends a **download link** instead of an attachment. Check logs for: `"Skipping attachment (too large or error)"`.
