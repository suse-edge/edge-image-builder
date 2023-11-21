package combustion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembleScript(t *testing.T) {
	script, err := assembleScript([]string{"foo.sh", "bar.sh", "baz.sh"})
	require.NoError(t, err)

	// alphabetic ordering
	assert.Contains(t, script, "./bar.sh\n./baz.sh\n./foo.sh")
}
