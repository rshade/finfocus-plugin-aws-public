package router

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiscover_FindsMatchingBinaries verifies that Discover() correctly identifies
// region-specific binaries matching the naming convention finfocus-plugin-aws-public-{region}.
func TestDiscover_FindsMatchingBinaries(t *testing.T) {
	dir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	// Create valid region binaries
	regions := []string{"us-east-1", "us-west-2", "eu-west-1"}
	for _, region := range regions {
		path := filepath.Join(dir, "finfocus-plugin-aws-public-"+region)
		require.NoError(t, os.WriteFile(path, []byte("binary"), 0755))
	}

	result := Discover(dir, logger)

	assert.Len(t, result, 3)
	for _, region := range regions {
		assert.Contains(t, result, region)
		assert.Equal(t, filepath.Join(dir, "finfocus-plugin-aws-public-"+region), result[region])
	}
}

// TestDiscover_IgnoresInvalidFiles verifies that Discover() skips files that don't
// match the expected binary naming pattern.
func TestDiscover_IgnoresInvalidFiles(t *testing.T) {
	dir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	// Create valid binary
	require.NoError(t, os.WriteFile(filepath.Join(dir, "finfocus-plugin-aws-public-us-east-1"), []byte("binary"), 0755))

	// Create invalid files
	invalidNames := []string{
		"finfocus-plugin-aws-public",        // no region
		"finfocus-plugin-aws-public-",       // empty region
		"some-other-binary",                 // unrelated binary
		"finfocus-plugin-aws-public.tar.gz", // archive, not binary
		"README.md",                         // documentation
	}
	for _, name := range invalidNames {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("data"), 0755))
	}

	result := Discover(dir, logger)

	assert.Len(t, result, 1)
	assert.Contains(t, result, "us-east-1")
}

// TestDiscover_EmptyDirectory verifies that Discover() returns an empty map when
// the directory contains no matching binaries.
func TestDiscover_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	result := Discover(dir, logger)

	assert.Empty(t, result)
}

// TestDiscover_NonExistentDirectory verifies that Discover() gracefully handles
// a directory that doesn't exist.
func TestDiscover_NonExistentDirectory(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	nonExistentDir := filepath.Join(t.TempDir(), "nonexistent")

	result := Discover(nonExistentDir, logger)

	assert.Empty(t, result)
}

// TestDiscover_SkipsDirectories verifies that Discover() skips subdirectories
// even if they match the naming pattern.
func TestDiscover_SkipsDirectories(t *testing.T) {
	dir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	// Create a directory with a region-like name
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "finfocus-plugin-aws-public-us-east-1"), 0755))

	// Create a valid binary
	require.NoError(t, os.WriteFile(filepath.Join(dir, "finfocus-plugin-aws-public-us-west-2"), []byte("binary"), 0755))

	result := Discover(dir, logger)

	assert.Len(t, result, 1)
	assert.Contains(t, result, "us-west-2")
}

// TestDiscover_GovCloudRegions verifies that Discover() correctly identifies
// GovCloud region binaries with the gov- prefix in the region name.
func TestDiscover_GovCloudRegions(t *testing.T) {
	dir := t.TempDir()
	logger := zerolog.New(zerolog.NewTestWriter(t))

	govRegions := []string{"us-gov-west-1", "us-gov-east-1"}
	for _, region := range govRegions {
		path := filepath.Join(dir, "finfocus-plugin-aws-public-"+region)
		require.NoError(t, os.WriteFile(path, []byte("binary"), 0755))
	}

	result := Discover(dir, logger)

	assert.Len(t, result, 2)
	for _, region := range govRegions {
		assert.Contains(t, result, region)
	}
}

// TestBinaryPattern_Matches verifies the binary name regex against valid patterns.
func TestBinaryPattern_Matches(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // expected region, empty if no match
	}{
		{"standard region", "finfocus-plugin-aws-public-us-east-1", "us-east-1"},
		{"west region", "finfocus-plugin-aws-public-us-west-2", "us-west-2"},
		{"eu region", "finfocus-plugin-aws-public-eu-west-1", "eu-west-1"},
		{"ap region", "finfocus-plugin-aws-public-ap-southeast-1", "ap-southeast-1"},
		{"sa region", "finfocus-plugin-aws-public-sa-east-1", "sa-east-1"},
		{"ca region", "finfocus-plugin-aws-public-ca-central-1", "ca-central-1"},
		{"govcloud west", "finfocus-plugin-aws-public-us-gov-west-1", "us-gov-west-1"},
		{"govcloud east", "finfocus-plugin-aws-public-us-gov-east-1", "us-gov-east-1"},
		{"windows binary", "finfocus-plugin-aws-public-us-east-1.exe", "us-east-1"},
		{"no region", "finfocus-plugin-aws-public", ""},
		{"empty region", "finfocus-plugin-aws-public-", ""},
		{"unrelated", "some-other-binary", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := binaryPattern.FindStringSubmatch(tt.input)
			if tt.expected == "" {
				assert.Len(t, matches, 0)
			} else {
				require.Len(t, matches, 2)
				assert.Equal(t, tt.expected, matches[1])
			}
		})
	}
}
