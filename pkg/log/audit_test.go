package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatComponentStatus(t *testing.T) {
	// Tests
	tests := []struct {
		testName  string
		component string
		status    string
		expected  string
	}{
		{
			testName:  "Success message",
			component: "myComponent",
			status:    messageSuccess,
			expected:  "Mycomponent .................. [SUCCESS]",
		},
		{
			testName:  "Skipped message",
			component: "my component",
			status:    messageSkipped,
			expected:  "My Component ................. [SKIPPED]",
		},
		{
			testName:  "Failed message",
			component: "MYCOMPONENT",
			status:    messageFailed,
			expected:  "Mycomponent .................. [FAILED ]",
		},
	}

	// Run
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			found := formatComponentStatus(test.component, test.status)
			assert.Equal(t, test.expected, found)
		})
	}
}
