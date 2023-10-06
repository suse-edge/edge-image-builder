package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddCombustionScript(t *testing.T) {
	// Setup
	bc := BuildConfig{}

	// Test
	bc.AddCombustionScript("foo")
	bc.AddCombustionScript("bar")
	bc.AddCombustionScript("baz")

	// Verify
	require.Equal(t, 3, len(bc.CombustionScripts))
	assert.Equal(t, "foo", bc.CombustionScripts[0])
	assert.Equal(t, "bar", bc.CombustionScripts[1])
	assert.Equal(t, "baz", bc.CombustionScripts[2])
}