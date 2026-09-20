.PHONY: all build run clean

BINARY_NAME=schema-generator
SOURCE_FILE=schema-generator.go
OUTPUT_FILE=schema.json

all: build

build:
	go build -o $(BINARY_NAME) $(SOURCE_FILE)

run:
	go run $(SOURCE_FILE) > $(OUTPUT_FILE)

clean:
	rm -f $(BINARY_NAME) $(OUTPUT_FILE)
