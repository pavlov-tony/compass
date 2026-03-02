package main

import (
	"flag"
	"log"
	"net"
	"os"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

func main() {
	outputFile := flag.String("o", "GeoIP2-Country-Test.mmdb", "output file path")
	flag.Parse()

	// Load the writer
	writer, err := mmdbwriter.New(
		mmdbwriter.Options{
			DatabaseType: "Country",
			RecordSize:   24,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// Define the data mapping based on README.md
	data := map[string]string{
		"2a02:7b00::/32": "NL", // Netherlands
		"141.1.1.1/32":   "DE", // Germany
		"2001:41d0::/32": "FR", // France
		"151.5.0.0/16":   "IT", // Italy
		"217.65.48.0/24": "ES", // Spain
	}

	for cidr, isoCode := range data {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Fatalf("Invalid CIDR: %s", cidr)
		}

		// Create the record structure expected by MaxMind DB
		record := mmdbtype.Map{
			"country": mmdbtype.Map{
				"iso_code": mmdbtype.String(isoCode),
			},
		}

		if err := writer.Insert(network, record); err != nil {
			log.Fatalf("Failed to insert %s: %v", cidr, err)
		}
	}

	// Write to file
	fh, err := os.Create(*outputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer fh.Close()

	if _, err := writer.WriteTo(fh); err != nil {
		log.Fatal(err)
	}
}
