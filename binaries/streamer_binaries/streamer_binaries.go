package streamer_binaries

import (
	"fmt"
	"os"
	"runtime"
)

// Binaries version
var Version = "0.5.2"

// Get the directory path where this file resides.
var DirPath, _ = os.Getwd()

// Compute the part of the file name that indicates the OS.
var OsMap = map[string]string{
	"linux":   "linux",
	"windows": "win",
	"darwin":  "osx",
}

// Compute the part of the file name that indicates the CPU architecture.
var CpuMap = map[string]string{
	"amd64": "x64", // Linux/Mac report this key
	"386":   "x86", // Windows reports this key
	"arm64": "arm64",
}

// The path to the installed FFmpeg binary.
var Ffmpeg = fmt.Sprintf("ffmpeg-%s-%s", OsMap[runtime.GOOS], CpuMap[runtime.GOARCH])

// The path to the installed FFprobe binary.
var Ffprobe = fmt.Sprintf("ffprobe-%s-%s", OsMap[runtime.GOOS], CpuMap[runtime.GOARCH])

// The path to the installed Shaka Packager binary.
var Packager = fmt.Sprintf("packager-%s-%s", OsMap[runtime.GOOS], CpuMap[runtime.GOARCH])
