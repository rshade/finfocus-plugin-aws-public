package router

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

const (
	// defaultBaseURL is the GitHub Releases base URL for downloading region binaries.
	defaultBaseURL = "https://github.com/rshade/finfocus-plugin-aws-public/releases/download"

	// checksumFieldCount is the expected number of fields in a checksums.txt line (hash + filename).
	checksumFieldCount = 2

	// maxBinarySize is the maximum allowed size for extracted binaries (500 MB).
	// This prevents decompression bomb attacks.
	maxBinarySize = 500 * 1024 * 1024
)

// Downloader handles binary download, SHA256 verification, and extraction.
type Downloader struct {
	version    string
	baseURL    string
	targetDir  string
	checksums  map[string]string
	httpClient *http.Client
	mu         sync.Mutex
	logger     zerolog.Logger
}

// NewDownloader creates a new Downloader for the given version.
func NewDownloader(version, targetDir string, logger zerolog.Logger) *Downloader {
	return &Downloader{
		version:    version,
		baseURL:    defaultBaseURL,
		targetDir:  targetDir,
		httpClient: &http.Client{},
		logger:     logger.With().Str("component", "downloader").Logger(),
	}
}

// Download downloads, verifies, and extracts a region binary.
// It returns the absolute path to the extracted binary.
func (d *Downloader) Download(ctx context.Context, region string) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Ensure checksums are loaded
	if d.checksums == nil {
		if err := d.fetchChecksums(ctx); err != nil {
			return "", fmt.Errorf("failed to fetch checksums: %w", err)
		}
	}

	// Construct tarball filename
	tarballName := fmt.Sprintf("finfocus-plugin-aws-public_%s_%s_%s_%s.tar.gz",
		d.version, titleCase(goos), goarch, region)

	// Download the tarball
	tarballURL := fmt.Sprintf("%s/v%s/%s", d.baseURL, d.version, tarballName)
	d.logger.Info().
		Str("url", tarballURL).
		Str("region", region).
		Msg("downloading region binary")

	tarballPath := filepath.Join(d.targetDir, tarballName)
	if err := d.downloadFile(ctx, tarballURL, tarballPath); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	// Verify SHA256
	expectedHash, ok := d.checksums[tarballName]
	if !ok {
		_ = os.Remove(tarballPath)
		return "", fmt.Errorf("no checksum found for %s", tarballName)
	}

	if err := d.verify(tarballPath, expectedHash); err != nil {
		_ = os.Remove(tarballPath)
		return "", err
	}

	d.logger.Debug().Str("file", tarballName).Msg("SHA256 verification passed")

	// Extract binary from tarball
	binaryPath, err := d.extractBinary(tarballPath, region)
	if err != nil {
		_ = os.Remove(tarballPath)
		return "", fmt.Errorf("extraction failed: %w", err)
	}

	// Clean up tarball
	_ = os.Remove(tarballPath)

	// Set executable permission
	//nolint:gosec // G302: Binary must be executable
	if chmodErr := os.Chmod(binaryPath, 0755); chmodErr != nil {
		return "", fmt.Errorf("failed to set executable permission: %w", chmodErr)
	}

	d.logger.Info().
		Str("region", region).
		Str("path", binaryPath).
		Msg("region binary installed")

	return binaryPath, nil
}

// fetchChecksums downloads and parses the checksums.txt file from the release.
func (d *Downloader) fetchChecksums(ctx context.Context) error {
	url := fmt.Sprintf("%s/v%s/checksums.txt", d.baseURL, d.version)
	d.logger.Debug().Str("url", url).Msg("fetching checksums")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums download returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksums: %w", err)
	}

	d.checksums = parseChecksums(string(body))
	d.logger.Debug().Int("entries", len(d.checksums)).Msg("checksums loaded")
	return nil
}

// parseChecksums parses a checksums.txt file into a map of filename to SHA256 hash.
// Format: <hash>  <filename> (two spaces between hash and filename).
func parseChecksums(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: hash  filename (with two spaces)
		parts := strings.Fields(line)
		if len(parts) == checksumFieldCount {
			result[parts[1]] = parts[0]
		}
	}
	return result
}

// verify computes SHA256 of the file and compares against the expected hash.
func (d *Downloader) verify(filePath, expectedHash string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for verification: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, copyErr := io.Copy(h, f); copyErr != nil {
		return fmt.Errorf("failed to compute SHA256: %w", copyErr)
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

// downloadFile downloads a URL to a local file path.
func (d *Downloader) downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, createErr := os.Create(destPath)
	if createErr != nil {
		return fmt.Errorf("failed to create file: %w", createErr)
	}
	defer out.Close()

	if _, copyErr := io.Copy(out, resp.Body); copyErr != nil {
		return fmt.Errorf("failed to write file: %w", copyErr)
	}

	return nil
}

// extractBinary extracts the region binary from a tar.gz archive.
func (d *Downloader) extractBinary(tarballPath, region string) (string, error) {
	f, err := os.Open(tarballPath)
	if err != nil {
		return "", fmt.Errorf("failed to open tarball: %w", err)
	}
	defer f.Close()

	gz, gzErr := gzip.NewReader(f)
	if gzErr != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", gzErr)
	}
	defer gz.Close()

	expectedBinaryName := fmt.Sprintf("finfocus-plugin-aws-public-%s", region)

	tr := tar.NewReader(gz)
	for {
		header, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return "", fmt.Errorf("tar read error: %w", nextErr)
		}

		baseName := filepath.Base(header.Name)
		if baseName != expectedBinaryName && baseName != expectedBinaryName+".exe" {
			continue
		}

		return d.extractTarEntry(tr, baseName)
	}

	return "", fmt.Errorf("binary %s not found in archive", expectedBinaryName)
}

// extractTarEntry writes a single tar entry to the target directory with size limits.
func (d *Downloader) extractTarEntry(r io.Reader, baseName string) (string, error) {
	destPath := filepath.Join(d.targetDir, baseName)
	out, createErr := os.Create(destPath)
	if createErr != nil {
		return "", fmt.Errorf("failed to create binary: %w", createErr)
	}

	// Limit extraction size to prevent decompression bombs
	limitedReader := io.LimitReader(r, maxBinarySize)
	if _, copyErr := io.Copy(out, limitedReader); copyErr != nil {
		if closeErr := out.Close(); closeErr != nil {
			d.logger.Warn().Err(closeErr).Msg("failed to close file after copy error")
		}
		return "", fmt.Errorf("failed to extract binary: %w", copyErr)
	}
	if closeErr := out.Close(); closeErr != nil {
		return "", fmt.Errorf("failed to close extracted binary: %w", closeErr)
	}

	return destPath, nil
}

// titleCase returns the string with the first letter capitalized (for OS naming in GitHub releases).
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
