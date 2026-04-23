package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Issuance represents the structure of the SSLMate API response
type Issuance struct {
	ID         string   `json:"id"`
	TBSHash    string   `json:"tbs_sha256"`
	CertHash   string   `json:"cert_sha256"`
	DNSNames   []string `json:"dns_names"`
	NotBefore  string   `json:"not_before"`
	NotAfter   string   `json:"not_after"`
	Issuer     Issuer   `json:"issuer"`
	Revoked    bool     `json:"revoked"`
}

type Issuer struct {
	Name         string `json:"name"`
	FriendlyName string `json:"friendly_name"`
}

func saveMarkdown(filename string, domain string, issuances []Issuance) {
	if filename == "" || len(issuances) == 0 {
		return
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# SSLMate Certificate Search Results for %s\n\n", domain))
	sb.WriteString(fmt.Sprintf("- **Query Date:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("- **Total Records:** %d\n\n", len(issuances)))

	sb.WriteString("| ID | DNS Names | Issuer | Not Before | Not After | Revoked |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- | :--- |\n")

	for _, iss := range issuances {
		dnsNames := strings.Join(iss.DNSNames, "<br>")
		issuerInfo := fmt.Sprintf("%s<br><small>%s</small>", iss.Issuer.FriendlyName, iss.Issuer.Name)
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %t |\n",
			iss.ID,
			dnsNames,
			issuerInfo,
			iss.NotBefore,
			iss.NotAfter,
			iss.Revoked))
	}

	safeOutput := filepath.Base(filename)
	err := os.WriteFile(safeOutput, []byte(sb.String()), 0644)
	if err != nil {
		log.Printf("Error writing to output file: %v", err)
	}
}

func main() {
	domainPtr := flag.String("domain", "", "The domain to query (e.g., example.com)")
	subdomainsPtr := flag.Bool("subdomains", true, "Include subdomains in the search")
	wildcardsPtr := flag.Bool("wildcards", false, "Include wildcard names matching the domain")
	afterPtr := flag.String("after", "", "Return issuances discovered after this ID (for pagination)")
	limitPtr := flag.Int("limit", 0, "Maximum number of results to fetch (client-side limit, 0 = all)")
	outputPtr := flag.String("output", "", "Output filename for Markdown table (e.g., 'results.md')")
	delayPtr := flag.Duration("delay", 5*time.Second, "Proactive delay between requests (e.g., 1s, 500ms)")

	flag.Parse()

	if *domainPtr == "" {
		fmt.Println("Usage: crt-query-sslmate -domain <domain> [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	apiKey := os.Getenv("SSLMATE_API")
	if apiKey == "" {
		log.Fatal("Error: SSLMATE_API environment variable is not set")
	}

	baseURL := "https://api.certspotter.com/v1/issuances"
	afterID := *afterPtr
	client := &http.Client{}

	fmt.Printf("Querying issuances for %s...\n", *domainPtr)
	fmt.Println(strings.Repeat("-", 80))

	var allIssuances []Issuance

	for {
		params := []string{
			fmt.Sprintf("domain=%s", *domainPtr),
			fmt.Sprintf("include_subdomains=%t", *subdomainsPtr),
			fmt.Sprintf("match_wildcards=%t", *wildcardsPtr),
			"expand=dns_names",
			"expand=issuer",
		}
		if afterID != "" {
			params = append(params, fmt.Sprintf("after=%s", afterID))
		}

		url := fmt.Sprintf("%s?%s", baseURL, strings.Join(params, "&"))

		var resp *http.Response

		// Retry loop for rate limiting
		for {
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				log.Fatalf("Error creating request: %v", err)
			}

			req.Header.Set("Authorization", "Bearer "+apiKey)

			resp, err = client.Do(req)
			if err != nil {
				log.Fatalf("Error performing request: %v", err)
			}

			if resp.StatusCode == http.StatusTooManyRequests {
				retryAfter := resp.Header.Get("Retry-After")
				waitSecs := 10 // Default wait
				if retryAfter != "" {
					if s, err := strconv.Atoi(retryAfter); err == nil {
						waitSecs = s
					}
				}
				fmt.Printf("Rate limited (429). Waiting %d seconds before retrying...\n", waitSecs)
				resp.Body.Close()
				time.Sleep(time.Duration(waitSecs) * time.Second)
				continue
			}
			break
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			log.Fatalf("API request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var pageIssuances []Issuance
		if err := json.NewDecoder(resp.Body).Decode(&pageIssuances); err != nil {
			resp.Body.Close()
			log.Fatalf("Error decoding JSON response: %v", err)
		}
		resp.Body.Close()

		if len(pageIssuances) == 0 {
			break // No more results
		}

		for _, iss := range pageIssuances {
			if *limitPtr > 0 && len(allIssuances) >= *limitPtr {
				break
			}
			
			allIssuances = append(allIssuances, iss)

			fmt.Printf("ID:         %s\n", iss.ID)
			fmt.Printf("Issuer:     %s (%s)\n", iss.Issuer.FriendlyName, iss.Issuer.Name)
			fmt.Printf("Not Before: %s\n", iss.NotBefore)
			fmt.Printf("Not After:  %s\n", iss.NotAfter)
			fmt.Printf("DNS Names:  %s\n", strings.Join(iss.DNSNames, ", "))
			fmt.Printf("Revoked:    %t\n", iss.Revoked)
			fmt.Println(strings.Repeat("-", 80))
		}

		// Save partial results immediately after processing the page
		saveMarkdown(*outputPtr, *domainPtr, allIssuances)

		if *limitPtr > 0 && len(allIssuances) >= *limitPtr {
			break
		}

		// Update 'after' to the ID of the last issuance in the current page
		afterID = pageIssuances[len(pageIssuances)-1].ID

		// Proactive delay to avoid hitting rate limits
		if *delayPtr > 0 {
			time.Sleep(*delayPtr)
		}
	}

	fmt.Printf("Total issuances fetched: %d\n", len(allIssuances))
	if *outputPtr != "" && len(allIssuances) > 0 {
		fmt.Printf("\nSuccessfully saved %d results to: %s\n", len(allIssuances), filepath.Base(*outputPtr))
	}
}
