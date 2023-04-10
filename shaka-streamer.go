package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Koodeyo-Media/shaka-streamer-go/binaries"
	"github.com/Koodeyo-Media/shaka-streamer-go/streamer"
	"github.com/Koodeyo-Media/shaka-streamer-go/tests"
	"gopkg.in/yaml.v3"
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
	test_assets := flag.Bool("test-assets", false, "Downloads all the assets for tests.")

	flag.Parse()

	if *setup {
		binaries.Setup()
		return
	}

	if *test_assets {
		if err := tests.FetchCloudAssets(); err != nil {
			fmt.Println(err)
		}
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

	inputConfigData, err := os.ReadFile(*inputConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input config file: %v\n", err)
		os.Exit(1)
	}

	var inputConfigDict streamer.InputConfig
	if err := yaml.Unmarshal(inputConfigData, &inputConfigDict); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing input config file: %v\n", err)
		os.Exit(1)
	}

	pipelineConfigData, err := os.ReadFile(*pipelineConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading pipeline config file: %v\n", err)
		os.Exit(1)
	}

	var pipelineConfigDict streamer.PipelineConfig
	if err := yaml.Unmarshal(pipelineConfigData, &pipelineConfigDict); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing pipeline config file: %v\n", err)
		os.Exit(1)
	}

	var bitrateConfigDict streamer.BitrateConfig
	if *bitrateConfig != "" {
		bitrateConfigData, err := os.ReadFile(*bitrateConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading bitrate config file: %v\n", err)
			os.Exit(1)
		}
		if err := yaml.Unmarshal(bitrateConfigData, &bitrateConfigDict); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing bitrate config file: %v\n", err)
			os.Exit(1)
		}
	}

	if *cloudURL != "" {
		if !strings.HasPrefix(*cloudURL, "gs://") && !strings.HasPrefix(*cloudURL, "s3://") {
			fmt.Fprintln(os.Stderr, "Invalid cloud URL! Only gs:// and s3:// URLs are supported")
			os.Exit(1)
		}
	}
}
