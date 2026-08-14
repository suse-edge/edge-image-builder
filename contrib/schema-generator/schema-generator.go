package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/invopop/jsonschema"
	"github.com/suse-edge/edge-image-builder/pkg/image"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func main() {
	// Helper variables for schema properties that require pointers.
	// The jsonschema library uses pointers for primitive types (like integers and booleans)
	// to distinguish between a zero value (e.g., 0 or false) and an omitted value (nil).
	// For example, MinItems: &uint64_1 means "minimum items is 1", whereas MinItems: nil
	// means "no minimum items constraint".
	uint64_1 := uint64(1)
	minItems1 := &uint64_1
	minLength1 := &uint64_1
	minPriority := json.Number("0")
	maxPriority := json.Number("99")

	// By default, the library looks for `json` tags. We need to configure it
	// to use `yaml` tags, as that's what the edge-image-builder structs use.
	reflector := &jsonschema.Reflector{
		// Treat all fields as optional unless they have a `validate:"required"` tag.
		// This allows us to manually specify required fields later, which gives us more control
		// over conditional requirements.
		RequiredFromJSONSchemaTags: true,
		FieldNameTag:               "yaml",
	}

	schema := reflector.Reflect(&image.Definition{})

	// Manually mark fields as required that are not tagged with `validate:"required"`
	// but are validated programmatically in the codebase.
	// See: https://github.com/suse-edge/edge-image-builder/blob/main/pkg/image/validation.go
	if definition, ok := schema.Definitions["Definition"]; ok {
		definition.AdditionalProperties = jsonschema.FalseSchema
		definition.Required = append(definition.Required, "image", "operatingSystem", "apiVersion")
		definition.Description = `Edge Image Builder Configuration.
ROOT OBJECT. All other configurations must be nested within this object.

Example:
{
  "apiVersion": "1.0",
  "image": {
    "imageType": "iso",
    "arch": "x86_64",
    "baseImage": "sles15sp5.iso",
    "outputImageName": "my-image"
  },
  "operatingSystem": {
    "users": [{"username": "root", "encryptedPassword": "..."}],
    "isoConfiguration": { "installDevice": "/dev/sda" }
  }
}`

		// Add conditional requirements based on imageType.
		// We use AllOf to apply multiple independent schemas.
		// Each schema in the list checks a specific condition (If) and applies constraints (Then).
		definition.AllOf = []*jsonschema.Schema{
			{
				// Condition: If image.imageType is "iso"
				If: &jsonschema.Schema{
					Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
						p := jsonschema.NewProperties()
						p.Set("image", &jsonschema.Schema{
							Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
								p := jsonschema.NewProperties()
								p.Set("imageType", &jsonschema.Schema{Const: "iso"})
								return p
							}(),
						})
						return p
					}(),
				},
				// Then: Ensure certain fields in operatingSystem.rawConfiguration are NOT present.
				// This prevents users from configuring RAW-specific settings when building an ISO.
				// Then: Ensure operatingSystem.isoConfiguration is NOT present and rawConfiguration IS present.
				Then: &jsonschema.Schema{
					Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
						p := jsonschema.NewProperties()
						p.Set("operatingSystem", &jsonschema.Schema{
							Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
								p := jsonschema.NewProperties()
								p.Set("isoConfiguration", &jsonschema.Schema{
									Description: "Configuration specific to ISO image builds. Optional when imageType is 'iso'.",
								})
								p.Set("rawConfiguration", &jsonschema.Schema{
									Not:         &jsonschema.Schema{},
									Description: "Configuration specific to RAW image builds. Forbidden when imageType is 'iso'.",
								})
								return p
							}(),
						})
						return p
					}(),
				},
			},
			{
				// Condition: If image.imageType is "raw"
				If: &jsonschema.Schema{
					Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
						p := jsonschema.NewProperties()
						p.Set("image", &jsonschema.Schema{
							Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
								p := jsonschema.NewProperties()
								p.Set("imageType", &jsonschema.Schema{Const: "raw"})
								return p
							}(),
						})
						return p
					}(),
				},
				// Then: Ensure operatingSystem.isoConfiguration.installDevice is NOT present.
				// This prevents users from configuring ISO-specific settings when building a RAW image.
				// Then: Ensure operatingSystem.rawConfiguration IS present and isoConfiguration is NOT present.
				Then: &jsonschema.Schema{
					Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
						p := jsonschema.NewProperties()
						p.Set("operatingSystem", &jsonschema.Schema{
							Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
								p := jsonschema.NewProperties()
								p.Set("isoConfiguration", &jsonschema.Schema{
									Not:         &jsonschema.Schema{},
									Description: "Configuration specific to ISO image builds. Forbidden when imageType is 'raw'.",
								})
								p.Set("rawConfiguration", &jsonschema.Schema{
									Description: "Configuration specific to RAW image builds. Optional when imageType is 'raw'.",
								})
								return p
							}(),
						})
						return p
					}(),
				},
			},
		}
	}
	// Manually add enum validation for apiVersion.
	// See: https://github.com/suse-edge/edge-image-builder/blob/main/pkg/version/version.go
	if definition, ok := schema.Definitions["Definition"]; ok {
		if apiVersion, ok := definition.Properties.Get("apiVersion"); ok {
			apiVersion.Enum = []interface{}{"1.0", "1.1", "1.2", "1.3"}
		}
	}

	// Manually add enum validation for fields with `oneof` tags.
	// See: https://github.com/suse-edge/edge-image-builder/blob/main/pkg/image/definition.go
	if imageDefinition, ok := schema.Definitions["Image"]; ok {
		if imageType, ok := imageDefinition.Properties.Get("imageType"); ok {
			imageType.Enum = []interface{}{"iso", "raw"}
			imageType.Description = "Type of image to build. Must be 'iso' or 'raw'."
		}
		if arch, ok := imageDefinition.Properties.Get("arch"); ok {
			arch.Enum = []interface{}{"x86_64", "aarch64"}
			arch.Description = "Target architecture. Must be 'x86_64' or 'aarch64'."
		}
		if baseImage, ok := imageDefinition.Properties.Get("baseImage"); ok {
			baseImage.Description = "Name of the base image to use (e.g., 'sles15sp5-x86_64'). Required."
		}
		if outputImageName, ok := imageDefinition.Properties.Get("outputImageName"); ok {
			outputImageName.Description = "Name of the output image file. Required."
		}
		imageDefinition.Required = append(imageDefinition.Required, "imageType", "arch", "baseImage", "outputImageName")

		// Add conditional requirements for the 'image' definition.
		// (The existing imageDefinition.AllOf block for imageType/baseImage/iso is correct and remains here)
	}

	if k8sDefinition, ok := schema.Definitions["Kubernetes"]; ok {
		k8sDefinition.AdditionalProperties = jsonschema.FalseSchema
		k8sDefinition.Required = append(k8sDefinition.Required, "version")
		k8sDefinition.Description = `Kubernetes configuration.
Example:
{
  "version": "1.29.0",
  "network": { "apiVIP": "1.2.3.4" },
  "nodes": [
    { "hostname": "node1", "type": "server", "initializer": true },
    { "hostname": "node2", "type": "server" }
  ]
}`
		if nodes, ok := k8sDefinition.Properties.Get("nodes"); ok {
			nodes.Description = "List of nodes. Required for multi-node. Example: [{ 'hostname': 'n1', 'type': 'server' }]"
		}
		if network, ok := k8sDefinition.Properties.Get("network"); ok {
			network.Description = "Network config. Example: { 'apiVIP': '1.2.3.4' }"
		}
	}

	if osDefinition, ok := schema.Definitions["OperatingSystem"]; ok {
		osDefinition.Description = `Operating System configuration.
Example (ISO):
{
  "users": [{"username": "root", "encryptedPassword": "..."}],
  "isoConfiguration": { "installDevice": "/dev/sda" },
  "time": { "timezone": "UTC", "ntp": { "servers": ["pool.ntp.org"] } }
}`
		if time, ok := osDefinition.Properties.Get("time"); ok {
			time.Description = "Time configuration. Example: { 'timezone': 'UTC', 'ntp': { 'servers': ['pool.ntp.org'] } }"
		}
		if isoConfig, ok := osDefinition.Properties.Get("isoConfiguration"); ok {
			isoConfig.Description = "ISO configuration object. MUST be nested here. Example: { 'installDevice': '/dev/sda' }"
		}
		if rawConfig, ok := osDefinition.Properties.Get("rawConfiguration"); ok {
			rawConfig.Description = "RAW configuration object. MUST be nested here. Example: { 'diskSize': '10G' }"
		}
		if users, ok := osDefinition.Properties.Get("users"); ok {
			users.Description = "List of users. Example: [{ 'username': 'user', 'password': '...' }]"
		}
	}

	if timeDefinition, ok := schema.Definitions["Time"]; ok {
		if ntp, ok := timeDefinition.Properties.Get("ntp"); ok {
			ntp.Description = "NTP config. 'servers' and 'pools' are lists of STRINGS. Example: { 'servers': ['1.2.3.4'] }"
		}
	}

	// Add conditional requirements for the 'operatingSystem' definition.
	// if osDefinition, ok := schema.Definitions["OperatingSystem"]; ok {
	// 	// We removed the OneOf constraint here because it was too generic.
	// 	// Instead, we enforce the presence/absence of isoConfiguration/rawConfiguration
	// 	// in the top-level Definition.AllOf block based on image.imageType.
	// }

	if userDefinition, ok := schema.Definitions["OperatingSystemUser"]; ok {
		if username, ok := userDefinition.Properties.Get("username"); ok {
			username.Description = "Username for the user. Required."
		}
		if password, ok := userDefinition.Properties.Get("encryptedPassword"); ok {
			password.Description = "Encrypted password for the user. Required if sshKey is not provided."
		}
		if sshKey, ok := userDefinition.Properties.Get("sshKeys"); ok {
			sshKey.Description = "List of SSH keys for the user. Required if encryptedPassword is not provided."
		}
	}

	if ntpDefinition, ok := schema.Definitions["NtpConfiguration"]; ok {
		if servers, ok := ntpDefinition.Properties.Get("servers"); ok {
			servers.Description = "List of NTP server addresses (e.g., ['pool.ntp.org']). Must be a list of strings."
		}
		if pools, ok := ntpDefinition.Properties.Get("pools"); ok {
			pools.Description = "List of NTP pool addresses. Must be a list of strings."
		}
	}

	// Add conditional requirements for the 'Packages' definition.

	// Add conditional requirements for the 'Packages' definition.
	if packagesDefinition, ok := schema.Definitions["Packages"]; ok {
		// If packageList is present (and has at least 1 item)...
		packagesDefinition.If = &jsonschema.Schema{
			Properties: newProperties("packageList", &jsonschema.Schema{
				MinItems: minItems1,
			}),
			Required: []string{"packageList"},
		}
		// ...then either sccRegistrationCode OR additionalRepos (or both) must be present.
		// AnyOf allows one or more of the sub-schemas to match.
		packagesDefinition.Then = &jsonschema.Schema{
			AnyOf: []*jsonschema.Schema{
				{Required: []string{"sccRegistrationCode"}},
				{Required: []string{"additionalRepos"}},
			},
		}
	}

	if addRepoDefinition, ok := schema.Definitions["AddRepo"]; ok {
		addRepoDefinition.Required = append(addRepoDefinition.Required, "url")
		if priority, ok := addRepoDefinition.Properties.Get("priority"); ok {
			priority.Minimum = minPriority
			priority.Maximum = maxPriority
		}
	}

	// Add conditional requirements for the 'Time' definition.
	// See: https://github.com/suse-edge/edge-image-builder/blob/main/pkg/image/validation/os.go
	// Timezone is not strictly required by validation logic, so we make it optional to avoid Gemini errors.

	if ntpDefinition, ok := schema.Definitions["NtpConfiguration"]; ok {
		ntpDefinition.OneOf = []*jsonschema.Schema{
			{
				Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
					p := newProperties("pools", &jsonschema.Schema{MinItems: minItems1})
					return p
				}(),
				Required: []string{"pools"},
			},
			{
				Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
					p := newProperties("servers", &jsonschema.Schema{MinItems: minItems1})
					return p
				}(),
				Required: []string{"servers"},
			},
		}
		ntpDefinition.If = &jsonschema.Schema{
			Properties: newProperties("forceWait", &jsonschema.Schema{Const: true}),
			Required:   []string{"forceWait"},
		}
		ntpDefinition.Then = &jsonschema.Schema{
			AnyOf: []*jsonschema.Schema{
				{Required: []string{"pools"}},
				{Required: []string{"servers"}},
			},
		}
	}

	// Add conditional requirements for the 'Users' definition.
	// See: https://github.com/suse-edge/edge-image-builder/blob/main/pkg/image/validation/os.go
	if userDefinition, ok := schema.Definitions["OperatingSystemUser"]; ok {
		userDefinition.AdditionalProperties = jsonschema.FalseSchema
		userDefinition.Description = `User configuration.
Allowed fields: "username", "uid", "encryptedPassword", "sshKeys", "primaryGroup", "secondaryGroups", "createHomeDir".
DO NOT use "name", "password", "sshKey" (singular).`
		userDefinition.Required = append(userDefinition.Required, "username")
		userDefinition.OneOf = []*jsonschema.Schema{
			{Required: []string{"encryptedPassword"}},
			{Required: []string{"sshKeys"}},
		}
	}

	// Add conditional requirements for the 'Suma' definition.
	if sumaDefinition, ok := schema.Definitions["Suma"]; ok {
		sumaDefinition.Required = append(sumaDefinition.Required, "host", "activationKey")
	}

	// Add conditional requirements for the 'FIPS' definition.
	// See: https://github.com/suse-edge/edge-image-builder/blob/main/pkg/image/validation/os.go
	if fipsDefinition, ok := schema.Definitions["FIPS"]; ok {
		fipsDefinition.Not = &jsonschema.Schema{
			Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
				p := newProperties("enable", &jsonschema.Schema{
					Const: true,
				})
				p.Set("disable", &jsonschema.Schema{
					Const: true,
				})
				return p
			}(),
		}
		fipsDefinition.Not.Properties.Set("disable", &jsonschema.Schema{Const: true})
	}

	// Add conditional requirements for the 'RawConfiguration' definition.
	// See: https://github.com/suse-edge/edge-image-builder/blob/main/pkg/image/validation/os.go
	if rawConfigDefinition, ok := schema.Definitions["RawConfiguration"]; ok {
		if diskSize, ok := rawConfigDefinition.Properties.Get("diskSize"); ok {
			diskSize.Pattern = "^([1-9]\\d+|[1-9])+([MGT])$"
		}
		rawConfigDefinition.If = &jsonschema.Schema{
			Properties: newProperties("expandEncryptedPartition", &jsonschema.Schema{Const: true}),
			Required:   []string{"expandEncryptedPartition"},
		}
		rawConfigDefinition.Then = &jsonschema.Schema{
			Required: []string{"luksKey"},
		}
	}

	// Add conditional requirements for the 'Group' definition.
	// See: https://github.com/suse-edge/edge-image-builder/blob/main/pkg/image/validation/os.go
	if groupDefinition, ok := schema.Definitions["Group"]; ok {
		groupDefinition.Required = append(groupDefinition.Required, "name")
	}

	// Add conditional requirements for the 'Systemd' definition.
	// See: https://github.com/suse-edge/edge-image-builder/blob/main/pkg/image/validation/os.go
	if systemdDefinition, ok := schema.Definitions["Systemd"]; ok {
		if enabled, ok := systemdDefinition.Properties.Get("enable"); ok {
			enabled.Items.MinLength = minLength1
		}
		if disabled, ok := systemdDefinition.Properties.Get("disable"); ok {
			disabled.Items.MinLength = minLength1
		}
	}

	// Add conditional requirements for kernel arguments.
	if osDefinition, ok := schema.Definitions["OperatingSystem"]; ok {
		if kernelArgs, ok := osDefinition.Properties.Get("kernelArgs"); ok {
			kernelArgs.Items.MinLength = minLength1
		}
	}

	// Manually add format validation for IP address fields.
	// See: https://github.com/suse-edge/edge-image-builder/blob/main/pkg/kubernetes/cluster.go
	if networkDefinition, ok := schema.Definitions["Network"]; ok {
		if apiVIP, ok := networkDefinition.Properties.Get("apiVIP"); ok {
			apiVIP.Format = "ipv4"
		}
		if apiVIP6, ok := networkDefinition.Properties.Get("apiVIP6"); ok {
			apiVIP6.Format = "ipv6"
		}
	}

	// Add conditional requirements for the 'Node' definition.
	if nodeDefinition, ok := schema.Definitions["Node"]; ok {
		nodeDefinition.AdditionalProperties = jsonschema.FalseSchema
		nodeDefinition.Description = "Node configuration. DO NOT include IP addresses here (they are not supported)."
		nodeDefinition.Required = append(nodeDefinition.Required, "hostname", "type")
		if nodeType, ok := nodeDefinition.Properties.Get("type"); ok {
			nodeType.Enum = []interface{}{"server", "agent"}
		}
		// Initializer implies server
		nodeDefinition.AllOf = append(nodeDefinition.AllOf, &jsonschema.Schema{
			If: &jsonschema.Schema{
				Properties: newProperties("initializer", &jsonschema.Schema{Const: true}),
				Required:   []string{"initializer"},
			},
			Then: &jsonschema.Schema{
				Properties: newProperties("type", &jsonschema.Schema{Const: "server"}),
			},
		})
	}

	if timeDefinition, ok := schema.Definitions["Time"]; ok {
		timeDefinition.AdditionalProperties = jsonschema.FalseSchema
		if timezone, ok := timeDefinition.Properties.Get("timezone"); ok {
			timezone.Description = "Timezone (e.g., 'UTC'). Note: field name is lowercase 'timezone'."
		}
	}

	if ntpDefinition, ok := schema.Definitions["NtpConfiguration"]; ok {
		ntpDefinition.AdditionalProperties = jsonschema.FalseSchema
	}

	if manifestsDefinition, ok := schema.Definitions["Manifests"]; ok {
		if urls, ok := manifestsDefinition.Properties.Get("urls"); ok {
			urls.Items.Pattern = "^http(s)?://"
		}
	}

	if helmChartDefinition, ok := schema.Definitions["HelmChart"]; ok {
		helmChartDefinition.AdditionalProperties = jsonschema.FalseSchema
		helmChartDefinition.Description = `Helm chart configuration.
Example:
{
  "name": "rancher",
  "repositoryName": "rancher-prime",
  "version": "2.10.0",
  "targetNamespace": "cattle-system",
  "createNamespace": true,
  "installationNamespace": "kube-system",
  "valuesFile": "rancher-values.yaml"
}`
		helmChartDefinition.Required = append(helmChartDefinition.Required, "name", "repositoryName", "version")
		if repoName, ok := helmChartDefinition.Properties.Get("repositoryName"); ok {
			repoName.Description = "Name of the repository to use. Must match a repository defined in 'repositories'. Required. DO NOT use 'repoUrl'."
		}
		helmChartDefinition.If = &jsonschema.Schema{
			Properties: newProperties("createNamespace", &jsonschema.Schema{Const: true}),
			Required:   []string{"createNamespace"},
		}
		helmChartDefinition.Then = &jsonschema.Schema{
			Required: []string{"targetNamespace"},
		}
	}

	if helmRepoDefinition, ok := schema.Definitions["HelmRepository"]; ok {
		helmRepoDefinition.AdditionalProperties = jsonschema.FalseSchema
		helmRepoDefinition.Description = `Helm repository configuration.
Example:
{
  "name": "rancher-prime",
  "url": "https://charts.rancher.com/server-charts/prime"
}`
		helmRepoDefinition.Required = append(helmRepoDefinition.Required, "name", "url")
		if url, ok := helmRepoDefinition.Properties.Get("url"); ok {
			url.Pattern = "^(oci|http|https)://"
		}
		// plainHTTP and skipTLSVerify cannot both be true
		helmRepoDefinition.AllOf = append(helmRepoDefinition.AllOf, &jsonschema.Schema{
			Not: &jsonschema.Schema{
				Properties: func() *orderedmap.OrderedMap[string, *jsonschema.Schema] {
					p := jsonschema.NewProperties()
					p.Set("plainHTTP", &jsonschema.Schema{Const: true})
					p.Set("skipTLSVerify", &jsonschema.Schema{Const: true})
					return p
				}(),
				Required: []string{"plainHTTP", "skipTLSVerify"},
			},
		})
	}

	if helmDefinition, ok := schema.Definitions["Helm"]; ok {
		// If charts are defined, repositories must also be defined.
		helmDefinition.If = &jsonschema.Schema{
			Properties: newProperties("charts", &jsonschema.Schema{
				MinItems: minItems1,
			}),
			Required: []string{"charts"},
		}
		helmDefinition.Then = &jsonschema.Schema{
			Properties: newProperties("repositories", &jsonschema.Schema{
				MinItems: minItems1,
			}),
			Required: []string{"repositories"},
		}
	}

	// Add top-level schema details.
	schema.Version = "http://json-schema.org/draft-07/schema#"
	schema.Title = "Edge Image Builder Configuration"
	schema.Description = "Schema for the configuration file used by the SUSE Edge Image Builder."

	// Fix root type to be object (required for some MCP clients)
	if schema.Ref != "" {
		schema.AllOf = []*jsonschema.Schema{{Ref: schema.Ref}}
		schema.Ref = ""
	}
	schema.Type = "object"

	// Marshal the schema to nicely formatted JSON.
	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling schema to JSON: %v", err)
	}

	// Print the final schema to standard output.
	fmt.Println(string(schemaBytes))
}

// newProperties is a helper function to create a new ordered map for properties.
func newProperties(key string, value *jsonschema.Schema) *orderedmap.OrderedMap[string, *jsonschema.Schema] {
	p := jsonschema.NewProperties()
	p.Set(key, value)
	return p
}
