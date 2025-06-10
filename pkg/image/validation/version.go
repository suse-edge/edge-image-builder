package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/suse-edge/edge-image-builder/pkg/image"
)

const (
	versionComponent = "Version"
)

type imageDefinitionField struct {
	// Key of the value from the definition YAML file.
	Key string
	// Chain represents the nested field structure under the image.Definition type.
	Chain []string
}

// Key is APIVersion where the fields are introduced.
var definitionFields = map[string][]imageDefinitionField{
	"1.1": {
		{Key: "kubernetes.helm.charts.apiVersions", Chain: []string{"Kubernetes", "Helm", "Charts", "APIVersions"}},
		{Key: "operatingSystem.enableFIPS", Chain: []string{"OperatingSystem", "EnableFIPS"}},
	},
	"1.2": {
		{Key: "kubernetes.network.apiVIP6", Chain: []string{"Kubernetes", "Network", "APIVIP6"}},
		{Key: "kubernetes.helm.charts.releaseName", Chain: []string{"Kubernetes", "Helm", "Charts", "ReleaseName"}},
		{Key: "operatingSystem.rawConfiguration.luksKey", Chain: []string{"OperatingSystem", "RawConfiguration", "LUKSKey"}},
		{Key: "operatingSystem.rawConfiguration.expandEncryptedPartition", Chain: []string{"OperatingSystem", "RawConfiguration", "ExpandEncryptedPartition"}},
		{Key: "operatingSystem.packages.enableExtras", Chain: []string{"OperatingSystem", "Packages", "EnableExtras"}},
		{Key: "embeddedArtifactRegistry.registries", Chain: []string{"EmbeddedArtifactRegistry", "Registries"}},
	},
}

func validateVersion(ctx *image.Context) []FailedValidation {
	var failures []FailedValidation
	var rootValue = reflect.ValueOf(ctx.ImageDefinition)

	for apiVersion, fields := range definitionFields {
		if strings.Compare(ctx.ImageDefinition.APIVersion, apiVersion) >= 0 {
			continue
		}

		for _, field := range fields {
			if isValueNonZero(rootValue, field.Chain) {
				failures = append(failures, FailedValidation{
					UserMessage: fmt.Sprintf("Field `%s` is only available in API version >= %s", field.Key, apiVersion),
				})
			}
		}
	}

	return failures
}

// Check whether a value in a chain of fields is non-zero.
func isValueNonZero(value reflect.Value, fieldChain []string) bool {
	for i, name := range fieldChain {
		if value.Kind() == reflect.Ptr && !value.IsNil() {
			value = value.Elem() // Dereference pointer if not nil
		}

		if value.Kind() != reflect.Struct {
			return false // Path broken, or not a struct
		}

		value = value.FieldByName(name) // Move on to the next value
		if !value.IsValid() {
			return false // Field is not found at this level, meaning the path doesn't exist
		}

		if i == len(fieldChain)-1 {
			return !value.IsZero() // Field chain exhausted, target field found
		}

		if value.Kind() == reflect.Slice {
			// Recursively check the slice against the remaining field chain
			for j := 0; j < value.Len(); j++ {
				if isValueNonZero(value.Index(j), fieldChain[i+1:]) {
					return true
				}
			}

			return false
		}
	}

	return false // Field chain exhausted, target field not found
}
