# Schema Generator

This tool generates the JSON schema for the Edge Image Builder configuration file. It inspects the Go structs in the `pkg/image` package and applies additional validations to match the logic enforced by the application.

## Files

- `schema-generator.go`: The source code for the tool.
- `schema.json`: The generated JSON schema.
- `go.mod` & `go.sum`: These files define the Go module for this tool. They are necessary because this tool is a standalone Go program with its own dependencies (like `github.com/invopop/jsonschema`) that might differ from or be independent of the main project's dependencies. This keeps the tool isolated and reproducible.
- `Makefile`: Helper script to build and run the tool.

## Usage

You can use the provided `Makefile` to interact with the tool.

### Generate the Schema

To run the tool and generate the `schema.json` file:

```bash
make run
```

This will execute `go run schema-generator.go` and redirect the output to `schema.json`.

### Build the Binary

To compile the tool into a binary named `schema-generator`:

```bash
make build
```

### Clean

To remove the generated binary and the `schema.json` file:

```bash
make clean
```
