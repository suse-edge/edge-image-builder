package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/suse-edge/edge-image-builder/pkg/fileio"
	"github.com/suse-edge/edge-image-builder/pkg/image"
)

func TestCreateRawImageCopyCommand(t *testing.T) {
	// Setup
	builder := Builder{
		context: &image.Context{
			ImageConfigDir: "config-dir",
			ImageDefinition: &image.Definition{
				Image: image.Image{
					BaseImage:       "base-image",
					OutputImageName: "build-image",
				},
			},
		},
	}

	// Test
	cmd := builder.createRawImageCopyCommand()

	// Verify
	require.NotNil(t, cmd)

	assert.Equal(t, copyExec, cmd.Path)
	expectedArgs := []string{
		copyExec,
		builder.generateBaseImageFilename(),
		builder.generateOutputImageFilename(),
	}
	assert.Equal(t, expectedArgs, cmd.Args)
}

func TestWriteModifyScript(t *testing.T) {
	// Setup
	ctx, teardown := setupContext(t)
	defer teardown()

	ctx.ImageDefinition = &image.Definition{
		Image: image.Image{
			OutputImageName: "output-image",
		},
	}

	builder := Builder{context: ctx}
	outputImageFilename := builder.generateOutputImageFilename()

	raw := image.OperatingSystem{
		KernelArgs: []string{"alpha", "beta"},
		RawConfiguration: image.RawConfiguration{
			DiskSize: "64G",
		},
	}

	iso := image.OperatingSystem{
		KernelArgs: []string{"alpha", "beta"},
		IsoConfiguration: image.IsoConfiguration{
			InstallDevice: "/dev/sda",
		},
	}

	luksKey := "1234"
	encryptedRaw := image.OperatingSystem{
		KernelArgs: []string{"alpha", "beta"},
		RawConfiguration: image.RawConfiguration{
			LUKSKey:  luksKey,
			DiskSize: "64G",
		},
	}

	encryptedRawExpand := image.OperatingSystem{
		KernelArgs: []string{"alpha", "beta"},
		RawConfiguration: image.RawConfiguration{
			LUKSKey:                  luksKey,
			ExpandEncryptedPartition: true,
			DiskSize:                 "64G",
		},
	}

	tests := []struct {
		name              string
		includeCombustion bool
		renameFilesystem  bool
		operatingSystem   *image.OperatingSystem
		expectedContains  []string
		expectedMissing   []string
	}{
		{
			name:              "RAW Image Usage",
			includeCombustion: true,
			renameFilesystem:  true,
			operatingSystem:   &raw,
			expectedContains: []string{
				fmt.Sprintf("guestfish --blocksize=$BLOCKSIZE --format=raw --rw -a %s", outputImageFilename),
				fmt.Sprintf("copy-in %s", builder.context.CombustionDir),
				"btrfs filesystem label / INSTALL",
				"truncate -s 64G",
				"virt-resize --expand $ROOT_PART",
			},
			expectedMissing: []string{
				"btrfs filesystem resize max /",
			},
		},
		{
			name:              "ISO Image Usage",
			includeCombustion: false,
			renameFilesystem:  false,
			operatingSystem:   &iso,
			expectedContains: []string{
				fmt.Sprintf("guestfish --blocksize=$BLOCKSIZE --format=raw --rw -a %s", outputImageFilename),
			},
			expectedMissing: []string{
				fmt.Sprintf("copy-in %s", builder.context.CombustionDir),
				"btrfs filesystem label / INSTALL",
				"btrfs filesystem resize max /",
			},
		},
		{
			name:              "Encrypted RAW Image Usage",
			includeCombustion: true,
			renameFilesystem:  true,
			operatingSystem:   &encryptedRaw,
			expectedContains: []string{
				fmt.Sprintf("guestfish --blocksize=$BLOCKSIZE --format=raw --rw -a %s $LUKSFLAG", outputImageFilename),
				fmt.Sprintf("copy-in %s", builder.context.CombustionDir),
				"btrfs filesystem label / INSTALL",
				"truncate -s 64G",
				"virt-resize --expand $ROOT_PART",
				fmt.Sprintf("LUKSFLAG=\"--key all:key:%s\"", luksKey),
			},
			expectedMissing: []string{
				"btrfs filesystem resize max /",
			},
		},
		{
			name:              "Encrypted RAW Image Usage With Expansion",
			includeCombustion: true,
			renameFilesystem:  true,
			operatingSystem:   &encryptedRawExpand,
			expectedContains: []string{
				fmt.Sprintf("guestfish --blocksize=$BLOCKSIZE --format=raw --rw -a %s $LUKSFLAG", outputImageFilename),
				fmt.Sprintf("copy-in %s", builder.context.CombustionDir),
				"btrfs filesystem label / INSTALL",
				"truncate -s 64G",
				"virt-resize --expand $ROOT_PART",
				fmt.Sprintf("LUKSFLAG=\"--key all:key:%s\"", luksKey),
				"btrfs filesystem resize max /",
			},
		},
	}

	// Test
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx.ImageDefinition.OperatingSystem = *test.operatingSystem
			err := builder.writeModifyScript(outputImageFilename, test.includeCombustion, test.renameFilesystem)
			require.NoError(t, err)

			expectedFilename := filepath.Join(ctx.BuildDir, modifyScriptName)
			foundBytes, err := os.ReadFile(expectedFilename)
			require.NoError(t, err)

			stats, err := os.Stat(expectedFilename)
			require.NoError(t, err)
			assert.Equal(t, fileio.ExecutablePerms, stats.Mode())
			foundContents := string(foundBytes)

			for _, findMe := range test.expectedContains {
				assert.Contains(t, foundContents, findMe)
			}
			for _, dontFindMe := range test.expectedMissing {
				assert.NotContains(t, foundContents, dontFindMe)
			}
		})
	}
}

func TestCreateModifyCommand(t *testing.T) {
	// Setup
	builder := Builder{
		context: &image.Context{
			BuildDir: "build-dir",
		},
	}
	// Test
	cmd := builder.createModifyCommand(io.Discard)

	// Verify
	require.NotNil(t, cmd)

	expectedPath := filepath.Join("build-dir", modifyScriptName)
	assert.Equal(t, expectedPath, cmd.Path)
	assert.Equal(t, io.Discard, cmd.Stdout)
	assert.Equal(t, io.Discard, cmd.Stderr)
}
