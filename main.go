package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/ahrav/go-packfile-analyzer/scanner"
)

func main() {
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	repoURL := "https://github.com/trufflesecurity/trufflehog.git"
	wants := []string{
		"eaceca8c2e77a8b0dae5ce976bf1901b8accd68f",
		// Add more SHAs as needed.
	}
	haves := []string{
		"9edeb164f449abb91c7abd81d82ea4fe80a8ed8a",
		// Add more SHAs as needed.
	}

	ps, err := scanner.NewPackScanner(repoURL, wants, haves)
	if err != nil {
		log.Fatalf("Failed to create PackScanner: %v", err)
	}

	ctx := context.Background()

	// Start scanning the packfile.
	pfr, err := ps.ScanPackfile(ctx)
	if err != nil {
		log.Fatalf("Failed to scan packfile: %v", err)
	}

	// Example of reading and processing the raw Git object bytes.

	// Read and process the raw Git object bytes.
	// For demonstration, we'll write them to stdout.
	// In a real application, you might process or store them differently.
	buf := make([]byte, 4096) // Buffer size can be adjusted as needed.

	for {
		n, err := pfr.Read(buf)
		if n > 0 {
			// Process the raw bytes. Here, we simply print them.
			fmt.Printf("Read %d bytes: %s\n", n, string(buf[:n]))
		}

		if err != nil {
			if err == io.EOF {
				// End of data.
				break
			}
			// Handle other errors.
			log.Fatalf("Error reading packfile data: %v", err)
		}
	}

	return nil
}
