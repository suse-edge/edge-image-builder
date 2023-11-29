package log

import (
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	lineLength = 40

	messageSuccess = "SUCCESS"
	messageSkipped = "SKIPPED"
	messageFailed  = "FAILED " // leave the trailing space for consistent lengths
)

// Audit displays a message to the user. This shouldn't be used for debug logging purposes; all
// messages passed in here should be user-readable.
func Audit(message string) {
	fmt.Println(message)
}

func AuditComponentSuccessful(component string) {
	message := formatComponentStatus(component, messageSuccess)
	Audit(message)
}

func AuditComponentSkipped(component string) {
	message := formatComponentStatus(component, messageSkipped)
	Audit(message)
}

func AuditComponentFailed(component string) {
	message := formatComponentStatus(component, messageFailed)
	Audit(message)
}

func formatComponentStatus(component, status string) string {
	// Example output:
	// Component ... [STATUS]

	name := cases.Title(language.English).String(component)
	numDots := lineLength - (len(name) + 2 + 9) // 2=spaces before/after dots, 9=status msg + []
	dots := strings.Repeat(".", numDots)

	message := fmt.Sprintf("%s %s [%s]", name, dots, status)
	return message
}
