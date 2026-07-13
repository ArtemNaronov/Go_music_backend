package cover

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Downloader saves remote cover images to disk.
type Downloader struct {
	client  *http.Client
	rootDir string
}

// NewDownloader creates a cover downloader.
func NewDownloader(rootDir string) *Downloader {
	return &Downloader{
		rootDir: rootDir,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RootDir returns the covers directory.
func (d *Downloader) RootDir() string {
	return d.rootDir
}

// EnsureDir creates the covers directory if needed.
func (d *Downloader) EnsureDir() error {
	return os.MkdirAll(d.rootDir, 0o755)
}

// PathFor returns the on-disk path for an album cover.
func (d *Downloader) PathFor(albumID string) string {
	safeID := strings.NewReplacer("/", "_", "\\", "_", ":", "_").Replace(albumID)
	return filepath.Join(d.rootDir, safeID+".jpg")
}

// Download fetches an image URL and stores it for the album.
func (d *Downloader) Download(albumID, imageURL string) (string, error) {
	if strings.TrimSpace(imageURL) == "" {
		return "", fmt.Errorf("empty image url")
	}
	if err := d.EnsureDir(); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, imageURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover download status %d", resp.StatusCode)
	}

	path := d.PathFor(albumID)
	tmpPath := path + ".tmp"

	file, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}

	return path, nil
}

// Read returns cover bytes from disk if present.
func (d *Downloader) Read(albumID string) ([]byte, bool, error) {
	path := d.PathFor(albumID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return data, true, nil
}
