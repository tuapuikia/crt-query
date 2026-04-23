package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type CertificateRecord struct {
	CommonName string
	CertID     int64
	IssuerName string
	IssuedDate time.Time
	ExpiryDate time.Time
}

func sanitize(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func main() {
	// Command-line flags
	top := flag.Int("top", 10, "Number of latest records to return")
	domain := flag.String("domain", "google.com", "Domain name to query")
	filter := flag.String("filter", "", "Filter string to match in the common name (e.g., 'rest')")
	output := flag.String("output", "", "Output filename for Markdown table (e.g., 'results.md')")
	flag.Parse()

	if *top <= 0 {
		*top = 10 // Default to 10 if invalid input
	}

	// Connection details for crt.sh
	// Host: crt.sh, Port: 5432, User: guest, Database: certwatch
	connStr := "host=crt.sh port=5432 user=guest dbname=certwatch sslmode=disable"

	fmt.Println("Connecting to crt.sh PostgreSQL...")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	fmt.Println("Successfully connected to crt.sh!")

	// Query for certificates using the new Full Text Search (FTS) approach
	fmt.Printf("\nQuerying for certificates strictly matching: %s (using FTS + strict domain filter, top %d)...\n", *domain, *top)
	if *filter != "" {
		fmt.Printf("Applying filter: %s\n", *filter)
	}

	// Using identities() and to_tsquery() as recommended for the new schema
	// Added ORDER BY x509_notBefore(c.CERTIFICATE) DESC to get the latest records
	// Added optional filter on x509_commonName
	// Note: crt.sh does not support unnamed prepared statements, so we use sanitized string interpolation.
	safeDomain := sanitize(*domain)
	filterClause := ""
	if *filter != "" {
		filterClause = fmt.Sprintf("AND x509_commonName(c.CERTIFICATE) ILIKE '%%%s%%'", sanitize(*filter))
	}

	query := fmt.Sprintf(`
		SELECT 
			COALESCE(x509_commonName(c.CERTIFICATE), '') AS common_name,
			c.ID,
			COALESCE(ca.name, '') AS issuer_name,
			COALESCE(x509_notBefore(c.CERTIFICATE), '1970-01-01'::timestamp) AS issued_date,
			COALESCE(x509_notAfter(c.CERTIFICATE), '1970-01-01'::timestamp) AS expiry_date
		FROM 
			certificate c
		JOIN
			ca ON c.issuer_ca_id = ca.id
		WHERE 
			to_tsquery('certwatch', '%s') @@ identities(c.CERTIFICATE)
			AND (
				x509_commonName(c.CERTIFICATE) ILIKE '%%.' || '%s'
				OR x509_commonName(c.CERTIFICATE) = '%s'
			)
			%s
		ORDER BY x509_notBefore(c.CERTIFICATE) DESC
		LIMIT %d;
	`, safeDomain, safeDomain, safeDomain, filterClause, *top)

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}
	defer rows.Close()

	var records []CertificateRecord
	for rows.Next() {
		var rec CertificateRecord
		if err := rows.Scan(&rec.CommonName, &rec.CertID, &rec.IssuerName, &rec.IssuedDate, &rec.ExpiryDate); err != nil {
			log.Fatalf("Error scanning row: %v", err)
		}
		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}

	// Print to console
	fmt.Printf("\nResults (Top %d Recent Certificates for %s):\n", *top, *domain)
	fmt.Printf("%-40s | %-10s | %-30s | %-12s | %-12s\n", "Common Name", "Cert ID", "Issuer", "Issued Date", "Expiry Date")
	fmt.Println(strings.Repeat("-", 120))

	for _, rec := range records {
		fmt.Printf("%-40s | %-10d | %-30.30s | %-12s | %-12s\n",
			rec.CommonName,
			rec.CertID,
			rec.IssuerName,
			rec.IssuedDate.Format("2006-01-02"),
			rec.ExpiryDate.Format("2006-01-02"))
	}

	// Output to Markdown file if requested
	if *output != "" {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Certificate Search Results for %s\n\n", *domain))
		sb.WriteString(fmt.Sprintf("- **Query Date:** %s\n", time.Now().Format("2006-01-02 15:04:05")))
		if *filter != "" {
			sb.WriteString(fmt.Sprintf("- **Filter:** `%s`\n", *filter))
		}
		sb.WriteString(fmt.Sprintf("- **Top Records:** %d\n\n", *top))

		sb.WriteString("| Common Name | Cert ID | Issuer | Issued Date | Expiry Date |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")

		for _, rec := range records {
			sb.WriteString(fmt.Sprintf("| %s | %d | %s | %s | %s |\n",
				rec.CommonName,
				rec.CertID,
				rec.IssuerName,
				rec.IssuedDate.Format("2006-01-02"),
				rec.ExpiryDate.Format("2006-01-02")))
		}

		safeOutput := filepath.Base(*output)
		err := os.WriteFile(safeOutput, []byte(sb.String()), 0644)
		if err != nil {
			log.Fatalf("Error writing to output file: %v", err)
		}
		fmt.Printf("\nSuccessfully saved results to: %s\n", safeOutput)
	}
}
