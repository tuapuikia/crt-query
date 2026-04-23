# SSLMate Certificate Transparency Query

A simple Go tool to query the SSLMate Certificate Transparency API.

## Setup

1.  **Set your API Key:**
    Obtain an API key from [SSLMATE](https://sslmate.com/) and set it as an environment variable:
    ```bash
    export SSLMATE_API="your_api_key_here"
    ```

2.  **Run the tool:**
    ```bash
    ./crt-query-sslmate -domain example.com -output results.md
    ```

## Options

- `-domain`: The domain to search for (required).
- `-subdomains`: Include subdomains in the search (default: `true`).
- `-wildcards`: Include wildcard names matching the domain (default: `false`).
- `-limit`: Maximum number of results to fetch. The tool handles pagination automatically to reach this limit. Set to `0` to fetch all available records (default: `0`).
- `-after`: Return only issuances discovered after this specific ID (useful for resuming a previous pagination).
- `-output`: Filename to save results as a Markdown table (e.g., `results.md`). If empty, results are only printed to the console.
- `-delay`: Proactive delay between requests to avoid hitting rate limits (default: `5s`).

## Rate Limiting & Pagination
The SSLMate API has rate limits. This tool handles them in two ways:
1.  **Proactive Delay:** It waits for a specified duration (default `5s`, configurable via `-delay`) between requests.
2.  **Automatic Retry:** If the API returns a `429 Too Many Requests` error, the tool will automatically wait for the duration specified by the server's `Retry-After` header (or 10 seconds if not provided) and then retry the request.

## API Documentation
Refer to the [official SSLMate CT Search API v1 documentation](https://sslmate.com/help/reference/ct_search_api_v1) for more details.
