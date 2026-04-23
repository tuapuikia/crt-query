# Certificate Transparency Query Tools

This repository contains two Go utilities for querying Certificate Transparency (CT) logs and certificate records from different sources: `crt.sh` (via PostgreSQL) and SSLMate (via API).

## Tools Overview

1.  **CRT Query (`crt-query-crtsh`)**: Queries the `crt.sh` PostgreSQL database directly.
2.  **SSLMate CT Query (`crt-query-sslmate`)**: Queries the SSLMate Certificate Transparency Search API.

---

## 1. CRT Query (crt.sh)

A utility to query the `crt.sh` PostgreSQL database for SSL/TLS certificate records using Full Text Search (FTS).

### Compilation

To compile as a statically linked binary:

```bash
cd crt-query-crtsh
CGO_ENABLED=0 go build -o crt-query main.go
```

### Usage

```bash
./crt-query-crtsh/crt-query -domain example.com -top 10 -output results.md
```

### Options

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-domain` | The domain name to query for certificates. | `google.com` |
| `-top` | Number of latest records to return. | `10` |
| `-filter` | A string to filter the common name (e.g., 'api'). | (empty) |
| `-output` | Filename to save results as a Markdown table. | (empty) |

---

## 2. SSLMate CT Query

A tool to query the SSLMate Certificate Transparency API with built-in rate limiting and pagination support.

### Setup

Obtain an API key from [SSLMATE](https://sslmate.com/) and set it as an environment variable:

```bash
export SSLMATE_API="your_api_key_here"
```

### Compilation

```bash
cd crt-query-sslmate
go build -o crt-query-sslmate main.go
```

### Usage

```bash
./crt-query-sslmate/crt-query-sslmate -domain example.com -output results.md
```

### Options

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-domain` | The domain to search for (required). | (empty) |
| `-subdomains` | Include subdomains in the search. | `true` |
| `-wildcards` | Include wildcard names matching the domain. | `false` |
| `-limit` | Max results to fetch (0 = all). | `0` |
| `-after` | Return issuances discovered after this ID. | (empty) |
| `-output` | Filename for Markdown table output. | (empty) |
| `-delay` | Proactive delay between requests. | `5s` |

---

## Security & Best Practices

- **Statically Linked Binaries**: Use `CGO_ENABLED=0` for maximum portability across Linux distributions.
- **Rate Limiting**: The SSLMate tool automatically handles `429 Too Many Requests` errors and respects `Retry-After` headers.
- **Sanitization**: Both tools include logic to prevent common injection issues (SQL sanitization for `crt.sh` and URL encoding/Markdown escaping for SSLMate).

## License
Refer to individual subdirectories for specific details.
