package tests

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const (
	outputDir       = "output_files/"
	TestDir         = "test_assets/"
	cloudTestAssets = "https://storage.googleapis.com/shaka-streamer-assets/test-assets/"
)

var TestFiles = []string{
	"BigBuckBunny.1080p.mp4",
	"Sintel.2010.720p.Small.mkv",
	"Sintel.2010.Arabic.vtt",
	"Sintel.2010.Chinese.vtt",
	"Sintel.2010.English.vtt",
	"Sintel.2010.Esperanto.vtt",
	"Sintel.2010.French.vtt",
	"Sintel.2010.Spanish.vtt",
	"Sintel.with.subs.mkv",
}

// FetchCloudAssets downloads all the assets needed for tests.
func FetchCloudAssets() error {
	testDirPath := filepath.Join(".", TestDir)

	// Ensure test directory exists
	if err := os.MkdirAll(testDirPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create test directory: %v", err)
	}

	// Download files
	for _, file := range TestFiles {
		filePath := filepath.Join(testDirPath, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			url := cloudTestAssets + file
			if err := downloadFile(url, filePath); err != nil {
				return fmt.Errorf("failed to download file %s: %v", file, err)
			}
		}
	}

	return nil
}

// downloadFile downloads a file from a URL to the specified path.
func downloadFile(url string, filePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return err
	}

	return nil
}
