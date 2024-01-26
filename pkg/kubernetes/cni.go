package kubernetes

import (
	"fmt"
	"strings"
)

func (c *Cluster) ExtractCNI() (cni string, multusEnabled bool, err error) {
	switch configuredCNI := c.ServerConfig[cniKey].(type) {
	case string:
		if configuredCNI == "" {
			return "", false, fmt.Errorf("cni not configured")
		}

		var cnis []string
		for _, cni = range strings.Split(configuredCNI, ",") {
			cnis = append(cnis, strings.TrimSpace(cni))
		}

		return parseCNIs(cnis)

	case []string:
		return parseCNIs(configuredCNI)

	case []any:
		var cnis []string
		for _, cni := range configuredCNI {
			c, ok := cni.(string)
			if !ok {
				return "", false, fmt.Errorf("invalid cni value: %v", cni)
			}
			cnis = append(cnis, c)
		}

		return parseCNIs(cnis)

	default:
		return "", false, fmt.Errorf("invalid cni: %v", configuredCNI)
	}
}

func parseCNIs(cnis []string) (cni string, multusEnabled bool, err error) {
	const multusPlugin = "multus"

	switch len(cnis) {
	case 1:
		cni = cnis[0]
		if cni == multusPlugin {
			return "", false, fmt.Errorf("multus must be used alongside another primary cni selection")
		}
	case 2:
		if cnis[0] == multusPlugin {
			cni = cnis[1]
			multusEnabled = true
		} else {
			return "", false, fmt.Errorf("multiple cni values are only allowed if multus is the first one")
		}
	default:
		return "", false, fmt.Errorf("invalid cni value: %v", cnis)
	}

	return cni, multusEnabled, nil
}
