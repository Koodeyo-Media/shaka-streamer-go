package main

import (
	"flag"
	"fmt"

	"github.com/Koodeyo-Media/shaka-streamer-go/binaries"
)

func main() {
	inputConfig := flag.String("input-config", "", "The path to the input config file (required).")
	pipelineConfig := flag.String("pipeline-config", "", "The path to the pipeline config file (required).")
	bitrateConfig := flag.String("bitrate-config", "", "The path to a config file which defines custom bitrates and resolutions for transcoding. (optional, see example in config_files/bitrate_config.yaml)")
	cloudURL := flag.String("cloud-url", "", "The Google Cloud Storage or Amazon S3 URL to upload to. (Starts with gs:// or s3://)")
	output := flag.String("output", "output_files", "The output folder to write files to, or an HTTP or HTTPS URL where files will be PUT. Used even if uploading to cloud storage.")
	skipDepsCheck := flag.Bool("skip-deps-check", false, "Skip checks for dependencies and their versions. This can be useful for testing pre-release versions of FFmpeg or Shaka Packager.")
	useSystemBinaries := flag.Bool("use-system-binaries", false, "Use FFmpeg, FFprobe and Shaka Packager binaries found in PATH instead of the ones offered by Shaka Streamer.")
	setup := flag.Bool("setup", false, "Downloads package containing FFmpeg, FFprobe, and Shaka Packager static builds.")

	flag.Parse()

	if *setup {
		binaries.Setup()
		return
	}

	if *inputConfig == "" {
		fmt.Println("The path to the input config file is required.")
		return
	}

	if *pipelineConfig == "" {
		fmt.Println("The path to the pipeline config file is required.")
		return
	}

	fmt.Printf("Input Config: %s\n", *inputConfig)
	fmt.Printf("Pipeline Config: %s\n", *pipelineConfig)
	fmt.Printf("Bitrate Config: %s\n", *bitrateConfig)
	fmt.Printf("Cloud URL: %s\n", *cloudURL)
	fmt.Printf("Output: %s\n", *output)
	fmt.Printf("Skip Deps Check: %t\n", *skipDepsCheck)
	fmt.Printf("Use System Binaries: %t\n", *useSystemBinaries)
}
