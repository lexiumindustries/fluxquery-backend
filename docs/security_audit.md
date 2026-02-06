# Security Audit Report - MySQL Export Service

This report identifies potential security vulnerabilities and provides recommendations for hardening the service.

## 1. Resource Exhaustion (Denial of Service)

### [CRITICAL] Memory OOM in Attachments
*   **Vulnerability**: The code previously read entire files into memory for attachments.
*   **Remediation**: Implemented a 25MB limit and fallback to download links.

### [HIGH] Access to Restricted Tables
*   **Vulnerability**: Potential for users to query `information_schema`.
*   **Remediation**: Implemented a query validator blocklist for system tables.

### [MEDIUM] SMTP Header Injection
*   **Vulnerability**: Potential for newline injection in email fields.
*   **Remediation**: Implemented strict email format validation.

## 2. Authentication
*   **Vulnerability**: Initially relied on simple secret headers.
*   **Hardening**: Implemented HMAC-SHA256 request signing with replay protection.
