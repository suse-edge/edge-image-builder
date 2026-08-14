package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/yaml.v3"
)

func main() {
	// 1. Parse command-line flags.
	var yamlFilePath string
	var schemaFilePath string

	flag.StringVar(&yamlFilePath, "d", "", "Path to the EIB definition YAML file (shorthand)")
	flag.StringVar(&yamlFilePath, "definition", "", "Path to the EIB definition YAML file")
	flag.StringVar(&schemaFilePath, "s", "", "Path to the JSON schema file (shorthand)")
	flag.StringVar(&schemaFilePath, "schema", "", "Path to the JSON schema file")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: schema-validator -d|--definition <path-to-yaml-file> -s|--schema <path-to-json-schema>\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nFlags:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -d (or --definition) <path/to/eib.yaml>\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Path to the EIB definition YAML file\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  -s (or --schema) <path/to/schema.json>\n")
		fmt.Fprintf(flag.CommandLine.Output(), "        Path to the JSON schema file\n")
	}

	flag.Parse()

	if yamlFilePath == "" || schemaFilePath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// 2. Read the YAML file from the provided path.
	yamlFile, err := os.ReadFile(yamlFilePath)
	if err != nil {
		fmt.Printf("Error reading YAML file '%s': %s\n", yamlFilePath, err)
		os.Exit(1)
	}

	// 3. Unmarshal the YAML into a generic interface{} which can represent any YAML structure.
	var yamlData any
	if err := yaml.Unmarshal(yamlFile, &yamlData); err != nil {
		fmt.Printf("Error unmarshalling YAML: %s\n", err)
		os.Exit(1)
	}

	// Recursively convert numeric apiVersion to string to accommodate YAML's type inference.
	yamlData = convertNumericApiVersionToString(yamlData)

	// 4. The gojsonschema library requires loaders for both the schema and the document to be validated.
	//    We create a loader for the schema from its file path.
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaFilePath)
	//    We create a loader for the YAML data that has been unmarshalled into a Go type.
	documentLoader := gojsonschema.NewGoLoader(yamlData)

	// 5. Perform the validation.
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		fmt.Printf("Error during validation: %s\n", err)
		os.Exit(1)
	}

	// 6. Check the validation result and provide feedback to the user.
	if result.Valid() {
		fmt.Printf("The document '%s' is valid.\n", yamlFilePath)
	} else {
		fmt.Printf("The document '%s' is not valid. see errors:\n", yamlFilePath)
		for _, err := range result.Errors() {
			// Filter out generic conditional errors that are not actionable for the user.
			// The 'allOf' and 'condition_then' errors are implementation details of the schema.
			// The specific error, like a missing required field, is what's truly useful.
			if err.Type() == "allOf" || err.Type() == "condition_then" {
				continue
			}
			fmt.Printf("- %s\n", err)
		}
		os.Exit(1)
	}
}

// convertNumericApiVersionToString traverses the unmarshalled YAML data and converts any 'apiVersion'
// field that has been parsed as a number into a string representation. This handles cases
// where the YAML parser interprets values like `1.1` as a float.
func convertNumericApiVersionToString(data any) any {
	switch v := data.(type) {
	case map[string]any:
		// Check for the apiVersion key at the current level.
		if apiVersion, ok := v["apiVersion"]; ok {
			// If it's a number, convert it to a string.
			if num, isNum := apiVersion.(float64); isNum {
				v["apiVersion"] = fmt.Sprintf("%.1f", num)
			}
		}
		// Recurse into the rest of the map.
		for key, val := range v {
			v[key] = convertNumericApiVersionToString(val)
		}
	case []any:
		for i, item := range v {
			v[i] = convertNumericApiVersionToString(item)
		}
	}
	return data
}
