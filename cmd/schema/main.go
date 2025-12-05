package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ConnorsApps/unifi-backup/pkg/config"
	"github.com/swaggest/jsonschema-go"
)

func main() {
	outputFile := flag.String("output", "", "Output file for the JSON schema (default: stdout)")
	ugly := flag.Bool("ugly", false, "Ugly-print the JSON output")
	flag.Parse()

	reflector := jsonschema.Reflector{}

	// Generate the JSON schema from the Config struct
	schema, err := reflector.Reflect(&config.Config{})
	if err != nil {
		log.Fatalf("Failed to generate schema: %v", err)
	}

	// Marshal to JSON
	var jsonData []byte
	if *ugly {
		jsonData, err = json.Marshal(schema)
	} else {
		jsonData, err = json.MarshalIndent(schema, "", "  ")
	}
	if err != nil {
		log.Fatalf("Failed to marshal schema to JSON: %v", err)
	}

	// Output to file or stdout
	if *outputFile != "" {
		err = os.WriteFile(*outputFile, jsonData, 0644)
		if err != nil {
			log.Fatalf("Failed to write schema to file: %v", err)
		}
		fmt.Fprintf(os.Stderr, "✓ Schema written to %s\n", *outputFile)
	} else {
		fmt.Println(string(jsonData))
	}
}
