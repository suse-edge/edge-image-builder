package combustion

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembleScript_DynamicNetwork(t *testing.T) {
	script, err := assembleScript([]string{"foo.sh", "bar.sh", "baz.sh"}, "")
	require.NoError(t, err)

	assert.Contains(t, script, "# combustion: network")
	assert.NotContains(t, script, "# combustion: prepare network")

	assert.NotContains(t, script, `if [ "${1-}" = "--prepare" ]; then`)
	assert.NotContains(t, script, "./configure-network.sh")

	// alphabetic ordering
	assert.Contains(t, script, `
echo "Running bar.sh"
./bar.sh

echo "Running baz.sh"
./baz.sh

echo "Running foo.sh"
./foo.sh
`)
}

func TestAssembleScript_StaticNetwork(t *testing.T) {
	script, err := assembleScript([]string{"foo.sh", "bar.sh", "baz.sh"}, "configure-network.sh")
	require.NoError(t, err)

	assert.Contains(t, script, "# combustion: prepare network")
	assert.NotContains(t, script, "# combustion: network")

	assert.Contains(t, script, `if [ "${1-}" = "--prepare" ]; then`)
	assert.Contains(t, script, "./configure-network.sh")

	// alphabetic ordering
	assert.Contains(t, script, `
echo "Running bar.sh"
./bar.sh

echo "Running baz.sh"
./baz.sh

echo "Running foo.sh"
./foo.sh
`)
}
