package router

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseChecksums verifies parsing of checksums.txt format into a map.
func TestParseChecksums(t *testing.T) {
	content := `abc123def456  file1.tar.gz
789abc000def  file2.tar.gz
deadbeef1234  file3.tar.gz`

	result := parseChecksums(content)

	assert.Len(t, result, 3)
	assert.Equal(t, "abc123def456", result["file1.tar.gz"])
	assert.Equal(t, "789abc000def", result["file2.tar.gz"])
	assert.Equal(t, "deadbeef1234", result["file3.tar.gz"])
}

// TestParseChecksums_EmptyLines verifies that empty lines are skipped.
func TestParseChecksums_EmptyLines(t *testing.T) {
	content := `abc123  file1.tar.gz

def456  file2.tar.gz
`

	result := parseChecksums(content)
	assert.Len(t, result, 2)
}

// TestParseChecksums_EmptyContent verifies that empty content returns empty map.
func TestParseChecksums_EmptyContent(t *testing.T) {
	result := parseChecksums("")
	assert.Empty(t, result)
}

// TestParseChecksums_DotSlashPrefix verifies that "./" prefixes on filenames
// are stripped, matching the output of "sha256sum ./*.tar.gz".
func TestParseChecksums_DotSlashPrefix(t *testing.T) {
	content := "abc123  ./file1.tar.gz\ndef456  ./file2.tar.gz\n"
	result := parseChecksums(content)
	assert.Len(t, result, 2)
	assert.Equal(t, "abc123", result["file1.tar.gz"])
	assert.Equal(t, "def456", result["file2.tar.gz"])
}

// TestParseChecksums_MixedPrefixes verifies that files with and without "./"
// prefixes are both handled correctly.
func TestParseChecksums_MixedPrefixes(t *testing.T) {
	content := "abc123  ./file1.tar.gz\ndef456  file2.tar.gz\n"
	result := parseChecksums(content)
	assert.Len(t, result, 2)
	assert.Equal(t, "abc123", result["file1.tar.gz"])
	assert.Equal(t, "def456", result["file2.tar.gz"])
}

// TestTitleCase verifies OS name title-casing for GitHub release URLs.
func TestTitleCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"linux", "Linux"},
		{"darwin", "Darwin"},
		{"windows", "Windows"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, titleCase(tt.input))
		})
	}
}

// TestDownloader_Verify_ValidHash verifies that SHA256 verification passes for a valid hash.
func TestDownloader_Verify_ValidHash(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	d := NewDownloader("1.0.0", t.TempDir(), logger)

	// Create a file with known content
	content := []byte("hello world")
	filePath := filepath.Join(t.TempDir(), "testfile")
	require.NoError(t, os.WriteFile(filePath, content, 0644))

	// Compute expected hash
	h := sha256.Sum256(content)
	expectedHash := hex.EncodeToString(h[:])

	err := d.verify(filePath, expectedHash)
	assert.NoError(t, err)
}

// TestDownloader_Verify_InvalidHash verifies that SHA256 verification fails for a mismatched hash.
func TestDownloader_Verify_InvalidHash(t *testing.T) {
	logger := zerolog.New(zerolog.NewTestWriter(t))
	d := NewDownloader("1.0.0", t.TempDir(), logger)

	content := []byte("hello world")
	filePath := filepath.Join(t.TempDir(), "testfile")
	require.NoError(t, os.WriteFile(filePath, content, 0644))

	err := d.verify(filePath, "0000000000000000000000000000000000000000000000000000000000000000")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SHA256 mismatch")
}

// TestDownloader_FetchChecksums verifies downloading and parsing checksums from a mock server.
func TestDownloader_FetchChecksums(t *testing.T) {
	checksumContent := "abc123  file1.tar.gz\ndef456  file2.tar.gz\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1.0.0/checksums.txt", r.URL.Path)
		fmt.Fprint(w, checksumContent)
	}))
	defer server.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	d := NewDownloader("1.0.0", t.TempDir(), logger)
	d.baseURL = server.URL // Override base URL to mock server

	err := d.fetchChecksums(context.Background())
	require.NoError(t, err)
	assert.Len(t, d.checksums, 2)
	assert.Equal(t, "abc123", d.checksums["file1.tar.gz"])
}

// TestDownloader_FetchChecksums_ServerError verifies error handling when checksums download fails.
func TestDownloader_FetchChecksums_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	logger := zerolog.New(zerolog.NewTestWriter(t))
	d := NewDownloader("1.0.0", t.TempDir(), logger)
	d.baseURL = server.URL

	err := d.fetchChecksums(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 404")
}

// TestDownloader_Download_FullFlow verifies the complete download, verify, and extract flow
// using a mock HTTP server serving a valid tarball with correct checksums.
func TestDownloader_Download_FullFlow(t *testing.T) {
	targetDir := t.TempDir()
	region := "us-east-1"
	binaryName := fmt.Sprintf("finfocus-plugin-aws-public-%s", region)
	binaryContent := []byte("#!/bin/sh\necho 'test binary'")

	logger := zerolog.New(zerolog.NewTestWriter(t))
	d := NewDownloader("1.0.0", targetDir, logger)
	archiveName := d.archiveName(region)

	// Create an archive containing the binary.
	archiveBytes := createTestArchive(t, archiveName, binaryName, binaryContent)

	// Compute checksum of the archive.
	h := sha256.Sum256(archiveBytes)
	archiveHash := hex.EncodeToString(h[:])

	checksumContent := fmt.Sprintf("%s  %s\n", archiveHash, archiveName)

	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1.0.0/checksums.txt":
			fmt.Fprint(w, checksumContent)
		case "/v1.0.0/" + archiveName:
			w.Write(archiveBytes)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	d.baseURL = server.URL

	path, err := d.Download(context.Background(), region)
	require.NoError(t, err)
	assert.NotEmpty(t, path)

	// Verify the extracted binary exists and has correct content
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, binaryContent, content)

	// Verify executable permission
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, info.Mode()&0111 != 0, "binary should be executable")
}

// TestDownloader_Download_ChecksumMismatch verifies that download fails when
// the SHA256 checksum doesn't match.
func TestDownloader_Download_ChecksumMismatch(t *testing.T) {
	targetDir := t.TempDir()
	region := "us-east-1"
	binaryName := fmt.Sprintf("finfocus-plugin-aws-public-%s", region)

	logger := zerolog.New(zerolog.NewTestWriter(t))
	d := NewDownloader("1.0.0", targetDir, logger)
	archiveName := d.archiveName(region)
	archiveBytes := createTestArchive(t, archiveName, binaryName, []byte("binary"))

	// Provide wrong checksum
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"
	checksumContent := fmt.Sprintf("%s  %s\n", wrongHash, archiveName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1.0.0/checksums.txt":
			fmt.Fprint(w, checksumContent)
		case "/v1.0.0/" + archiveName:
			w.Write(archiveBytes)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	d.baseURL = server.URL

	_, err := d.Download(context.Background(), region)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SHA256 mismatch")
}

// TestDownloader_Download_MissingChecksum verifies that download succeeds with a
// warning when no checksum entry exists for the requested tarball. This matches
// finfocus-core's lenient verification policy: only a confirmed hash mismatch is fatal.
func TestDownloader_Download_MissingChecksum(t *testing.T) {
	targetDir := t.TempDir()
	region := "us-east-1"
	binaryName := fmt.Sprintf("finfocus-plugin-aws-public-%s", region)
	binaryContent := []byte("#!/bin/sh\necho 'test binary'")
	logger := zerolog.New(zerolog.NewTestWriter(t))
	d := NewDownloader("1.0.0", targetDir, logger)
	archiveName := d.archiveName(region)

	archiveBytes := createTestArchive(t, archiveName, binaryName, binaryContent)

	// Checksums file has no entry for our archive.
	checksumContent := "abc123  some-other-file.tar.gz\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1.0.0/checksums.txt":
			fmt.Fprint(w, checksumContent)
		case "/v1.0.0/" + archiveName:
			w.Write(archiveBytes)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	d.baseURL = server.URL

	path, err := d.Download(context.Background(), region)
	require.NoError(t, err, "missing checksum entry should warn, not fail")
	assert.NotEmpty(t, path)

	// Verify the extracted binary exists and has correct content
	content, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, binaryContent, content)
}

// TestDownloader_Download_ChecksumsFetchError verifies that download succeeds
// with a warning when checksums.txt cannot be fetched (e.g., 404).
func TestDownloader_Download_ChecksumsFetchError(t *testing.T) {
	targetDir := t.TempDir()
	region := "us-east-1"
	binaryName := fmt.Sprintf("finfocus-plugin-aws-public-%s", region)
	binaryContent := []byte("#!/bin/sh\necho 'test binary'")
	logger := zerolog.New(zerolog.NewTestWriter(t))
	d := NewDownloader("1.0.0", targetDir, logger)
	archiveName := d.archiveName(region)

	archiveBytes := createTestArchive(t, archiveName, binaryName, binaryContent)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1.0.0/checksums.txt":
			w.WriteHeader(http.StatusNotFound)
		case "/v1.0.0/" + archiveName:
			w.Write(archiveBytes)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	d.baseURL = server.URL

	path, err := d.Download(context.Background(), region)
	require.NoError(t, err, "checksums.txt fetch failure should warn, not fail")
	assert.NotEmpty(t, path)
}

func createTestArchive(t *testing.T, archiveName, filename string, content []byte) []byte {
	t.Helper()
	if strings.HasSuffix(strings.ToLower(archiveName), ".zip") {
		return createTestZip(t, filename, content)
	}
	return createTestTarball(t, filename, content)
}

// createTestTarball creates a tar.gz archive containing a single file with the given name and content.
func createTestTarball(t *testing.T, filename string, content []byte) []byte {
	t.Helper()

	tmpFile := filepath.Join(t.TempDir(), "test.tar.gz")
	f, err := os.Create(tmpFile)
	require.NoError(t, err)
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	hdr := &tar.Header{
		Name: filename,
		Mode: 0755,
		Size: int64(len(content)),
	}
	require.NoError(t, tw.WriteHeader(hdr))
	_, err = tw.Write(content)
	require.NoError(t, err)

	// Must close writers before reading back
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())

	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	return data
}

// createTestZip creates a zip archive containing a single file with the given name and content.
func createTestZip(t *testing.T, filename string, content []byte) []byte {
	t.Helper()

	tmpFile := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(tmpFile)
	require.NoError(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	header := &zip.FileHeader{
		Name:   filename,
		Method: zip.Deflate,
	}
	header.SetMode(0755)
	w, err := zw.CreateHeader(header)
	require.NoError(t, err)

	_, err = w.Write(content)
	require.NoError(t, err)

	require.NoError(t, zw.Close())
	require.NoError(t, f.Close())

	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	return data
}
