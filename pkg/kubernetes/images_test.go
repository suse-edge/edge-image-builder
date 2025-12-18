package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestExtractCNI(t *testing.T) {
	tests := map[string]struct {
		input                 map[string]any
		expectedCNI           string
		expectedMultusEnabled bool
		expectedErr           string
	}{
		"CNI not configured": {
			input:       map[string]any{},
			expectedErr: "invalid cni: <nil>",
		},
		"Empty CNI string": {
			input: map[string]any{
				"cni": "",
			},
			expectedErr: "cni not configured",
		},
		"Empty CNI list": {
			input: map[string]any{
				"cni": []string{},
			},
			expectedErr: "invalid cni value: []",
		},
		"Multiple CNI list": {
			input: map[string]any{
				"cni": []string{"canal", "calico", "cilium"},
			},
			expectedErr: "invalid cni value: [canal calico cilium]",
		},
		"Valid CNI string": {
			input: map[string]any{
				"cni": "calico",
			},
			expectedCNI: "calico",
		},
		"Valid CNI list": {
			input: map[string]any{
				"cni": []string{"calico"},
			},
			expectedCNI: "calico",
		},
		"Valid CNI string with multus": {
			input: map[string]any{
				"cni": "multus, calico",
			},
			expectedCNI:           "calico",
			expectedMultusEnabled: true,
		},
		"Valid CNI list with multus": {
			input: map[string]any{
				"cni": []string{"multus", "calico"},
			},
			expectedCNI:           "calico",
			expectedMultusEnabled: true,
		},
		"Invalid standalone multus": {
			input: map[string]any{
				"cni": "multus",
			},
			expectedErr: "multus must be used alongside another primary cni selection",
		},
		"Invalid standalone multus list": {
			input: map[string]any{
				"cni": []string{"multus"},
			},
			expectedErr: "multus must be used alongside another primary cni selection",
		},
		"Valid CNI with invalid multus placement": {
			input: map[string]any{
				"cni": "cilium, multus",
			},
			expectedErr: "multiple cni values are only allowed if multus is the first one",
		},
		"Valid CNI list with invalid multus placement": {
			input: map[string]any{
				"cni": []string{"cilium", "multus"},
			},
			expectedErr: "multiple cni values are only allowed if multus is the first one",
		},
		"Invalid CNI list": {
			input: map[string]any{
				"cni": []any{"cilium", 6},
			},
			expectedErr: "invalid cni value: 6",
		},
		"Invalid CNI format": {
			input: map[string]any{
				"cni": 6,
			},
			expectedErr: "invalid cni: 6",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := yaml.Marshal(test.input)
			require.NoError(t, err)

			var config map[string]any
			require.NoError(t, yaml.Unmarshal(b, &config))

			cluster := Cluster{
				ServerConfig: config,
			}

			cni, multusEnabled, err := cluster.ExtractCNI()

			if test.expectedErr != "" {
				require.Error(t, err)
				assert.EqualError(t, err, test.expectedErr)
				assert.False(t, multusEnabled)
				assert.Empty(t, cni)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedCNI, cni)
				assert.Equal(t, test.expectedMultusEnabled, multusEnabled)
			}
		})
	}
}
