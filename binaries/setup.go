package binaries

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Koodeyo-Media/shaka-streamer-go/binaries/streamer_binaries"
)

// Version constants.package streamer_binaries

// Change to download different versions.
const (
	FFMPEG_VERSION   = "n4.4-2"
	PACKAGER_VERSION = "v2.6.1"
)

// A map of suffixes that will be combined with the binary download links
// to achieve a full download link.  Different suffix for each platform.
// Extend this dictionary to add more platforms.
var PLATFORM_SUFFIXES = map[string]string{
	// 64-bit Windows
	"win_amd64": "-win-x64.exe",
	// 64-bit Linux
	"manylinux1_x86_64": "-linux-x64",
	// Linux on ARM
	"manylinux2014_aarch64": "-linux-arm64",
	// 64-bit with 10.9 SDK
	"macosx_10_9_x86_64": "-osx-x64",
}

var (
	FFMPEG_DL_PREFIX   = fmt.Sprintf("https://github.com/shaka-project/static-ffmpeg-binaries/releases/download/%s", FFMPEG_VERSION)
	PACKAGER_DL_PREFIX = fmt.Sprintf("https://github.com/shaka-project/shaka-packager/releases/download/%s", PACKAGER_VERSION)

	BINARIES_DL = []string{
		FFMPEG_DL_PREFIX + "/ffmpeg",
		FFMPEG_DL_PREFIX + "/ffprobe",
		PACKAGER_DL_PREFIX + "/packager",
	}
)

func downloadBinary(downloadURL string, downloadDir string) string {
	binaryName := downloadURL[strings.LastIndex(downloadURL, "/")+1:]
	binaryPath := filepath.Join(downloadDir, binaryName)

	fmt.Printf("downloading %s ", binaryName)
	resp, err := http.Get(downloadURL)
	if err != nil {
		fmt.Println("failed")
		panic(err)
	}

	defer resp.Body.Close()

	outFile, err := os.Create(binaryPath)

	if err != nil {
		fmt.Println("failed")
		panic(err)
	}

	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)

	if err != nil {
		fmt.Println("failed")
		panic(err)
	}

	os.Chmod(binaryPath, 0755)
	fmt.Println("(finished)")
	return binaryName
}

// A package containing FFmpeg, FFprobe, and Shaka Packager static builds.
func Setup() {
	for _, binaryDL := range BINARIES_DL {
		// only download binaries for current os.
		downloadLink := fmt.Sprintf("%s-%s-%s", binaryDL, streamer_binaries.OsMap[runtime.GOOS], streamer_binaries.CpuMap[runtime.GOARCH])
		downloadBinary(downloadLink, streamer_binaries.DirPath)
	}
}
