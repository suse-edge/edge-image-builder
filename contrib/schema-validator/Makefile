.PHONY: all build run clean

BINARY_NAME=schema-validator
SOURCE_FILE=schema-validator.go
YAML_FILE=eib.yaml
SCHEMA_FILE=../schema-generator/schema.json

all: build

build:
	go build -o $(BINARY_NAME) $(SOURCE_FILE)

run:
	go run $(SOURCE_FILE) -d $(YAML_FILE) -s $(SCHEMA_FILE)

clean:
	rm -f $(BINARY_NAME)
