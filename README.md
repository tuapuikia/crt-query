# CRT Query Tool

A simple Go utility to query the `crt.sh` PostgreSQL database for SSL/TLS certificate records.

## Features
- Queries `crt.sh` directly via PostgreSQL.
- Supports domain-based searching using Full Text Search (FTS).
- Optional filtering by common name.
- Outputs results to the console and optionally to a Markdown file.
- Statically linked binary for portability across Linux distributions.

## Compilation

To compile the tool as a statically linked binary:

```bash
CGO_ENABLED=0 go build -o crt-query main.go
```

## Usage

Run the binary with the desired flags:

```bash
./crt-query -domain example.com -top 5 -output results.md
```

### Options

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-domain` | The domain name to query for certificates. | `example.com` |
| `-top` | Number of latest records to return. | `10` |
| `-filter` | A string to filter the common name (e.g., 'api'). | (empty) |
| `-output` | Filename to save results as a Markdown table. | (empty) |

## Example

```bash
./crt-query -domain example.com -filter www -top 3
```

This will search for the top 3 latest certificates for `example.com` that contain `www` in their common name and print them to the console.
