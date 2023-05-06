/*
Top-level module API.

If you'd like to import Shaka Streamer as a Python module and build it into
your own application, this is the top-level API you can use for that.  You may
also want to look at the source code to the command-line front end script
`shaka-streamer`.
*/
package streamer

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Koodeyo-Media/shaka-streamer-go/binaries/streamer_binaries"
)

// Controls all other nodes and manages shared resources.
type ControllerNode struct {
	tempDir          string
	hermeticFfmpeg   string
	hermeticPackager string
	inputConfig      InputConfig
	pipelineConfig   PipelineConfig
	nodes            []interface{}
}

type ControllerParams struct {
	outputLocation     string
	inputConfigDict    InputConfig
	pipelineConfigDict PipelineConfig
	bitrateConfigDict  BitrateConfig
	bucketURL          string
	checkDeps          bool
	useHermetic        bool
}

func NewControllerNode() *ControllerNode {
	globalTempDir := os.TempDir()

	// Create a temporary directory with a name that indicates who made it.
	tempDir, err := os.MkdirTemp(globalTempDir, "shaka-live-")
	if err != nil {
		panic(err)
	}

	return &ControllerNode{
		tempDir: tempDir,
	}
}

func (c ControllerNode) Start(params ControllerParams) *ControllerNode {
	rootDir, _ := RootDir()

	if params.useHermetic {
		ffmpeg := FileExists(filepath.Join(rootDir, streamer_binaries.Ffmpeg))
		ffprobe := FileExists(filepath.Join(rootDir, streamer_binaries.Ffprobe))
		packager := FileExists(filepath.Join(rootDir, streamer_binaries.Packager))

		if !ffmpeg || !ffprobe || !packager {
			panic("shaka-streamer-binaries was not found.\n  Install it with `pip install shaka-streamer-binaries`.\n  Alternatively, use the `--use-system-binaries` option if you want to use the system-wide binaries of ffmpeg/ffprobe/packager.")
		}
	}

	if len(c.nodes) > 0 {
		panic("Controller already started!")
	}

	if params.checkDeps {
		if params.useHermetic {
			// If we are using the hermetic binaries, check the module version.
			// We must match on the first two digits, but the last one can vary between
			// the two modules.
			shortenVersion := func(version string) string {
				components := strings.Split(version, ".")
				return strings.Join(components[0:2], ".")
			}

			nextShortVersion := func(version string) string {
				components := strings.Split(version, ".")
				i, err := strconv.Atoi(components[1])
				if err != nil {
					return ""
				}
				components[1] = strconv.Itoa(i + 1)
				return strings.Join(components[0:2], ".")
			}

			streamerShortVersion := shortenVersion(Version)
			streamerBinariesShortVersion := shortenVersion(streamer_binaries.Version)

			if streamerBinariesShortVersion != streamerShortVersion {
				// This is the recommended install command. It installs the most
				// recent version of the binary package that matches the current
				// version of streamer itself. This is much easier to do in nodejs
				// dependencies, because you can use a specifier like "1.2.x", but in
				// Python, you have to use a specifier like ">=1.2,<1.3".
				pipCommand := fmt.Sprintf("pip3 install 'shaka-streamer-binaries>=%s,<%s'", streamerShortVersion, nextShortVersion(Version))

				err := VersionError{
					Name:            "shaka-streamer-binaries",
					Problem:         "version does not match",
					RequiredVersion: streamerShortVersion,
					ExactMatch:      true,
					Addendum:        fmt.Sprintf("Install with: %s", pipCommand),
				}

				panic(err)
			}
		} else {
			// Check that ffmpeg version is 4.1 or above.
			if err := CheckCommandVersion("FFmpeg", []string{"ffmpeg", "-version"}, []int{4, 1}); err != nil {
				panic(err)
			}

			// Check that ffprobe version (used for autodetect features) is 4.1 or above.
			if err := CheckCommandVersion("ffprobe", []string{"ffprobe", "-version"}, []int{4, 1}); err != nil {
				panic(err)
			}

			// Check that Shaka Packager version is 2.6.0 or above.
			if err := CheckCommandVersion("Shaka Packager", []string{"packager", "-version"}, []int{2, 6, 1}); err != nil {
				panic(err)
			}
		}

		if params.bucketURL != "" {
			// Check that the Google Cloud SDK is at least v212, which introduced
			// gsutil 4.33 with an important rsync bug fix.
			// https://cloud.google.com/sdk/docs/release-notes
			// https://github.com/GoogleCloudPlatform/gsutil/blob/master/CHANGES.md
			// This is only required if the user asked for upload to cloud storage.
			if err := CheckCommandVersion("Google Cloud SDK", []string{"gcloud", "--version"}, []int{212, 0, 0}); err != nil {
				panic(err)
			}
		}
	}

	if params.bucketURL != "" {
		// If using cloud storage, make sure the user is logged in and can access
		// the destination, independent of the version check above.
		CloudNode{}.CheckAccess(params.bucketURL)
	}

	cn := NewControllerNode()

	if params.useHermetic {
		cn.hermeticFfmpeg = filepath.Join(rootDir, streamer_binaries.Ffmpeg)
		cn.hermeticPackager = filepath.Join(rootDir, streamer_binaries.Packager)
		HermeticFFProbe = filepath.Join(rootDir, streamer_binaries.Ffprobe)
	}

	cn.inputConfig = params.inputConfigDict
	cn.pipelineConfig = params.pipelineConfigDict

	if !IsURL(params.outputLocation) {
		// Check if the directory for outputted Packager files exists, and if it
		// does, delete it and remake a new one.
		if err := RemoveIfExists(params.outputLocation); err != nil {
			panic(err)
		}

		if err := os.MkdirAll(params.outputLocation, os.ModePerm); err != nil {
			panic(err)
		}
	} else {
		// Check some restrictions and other details on HTTP output.
		if !params.pipelineConfigDict.SegmentPerFile {
			panic("For HTTP PUT uploads, the pipeline segment_per_file setting must be set to True!")
		}

		if params.bucketURL != "" {
			panic("Cloud bucket upload is incompatible with HTTP PUT support.")
		}

		if len(params.inputConfigDict.MultiPeriodInputsList) > 0 {
			// TODO: Edit Multiperiod input list implementation to support HTTP outputs
			panic("Multiperiod input list support is incompatible with HTTP outputs.")
		}
	}

	if params.pipelineConfigDict.LowLatencyDashMode {
		// Check some restrictions on LL-DASH packaging.
		if !ContainsString(ManifestFormatListToStringList(params.pipelineConfigDict.ManifestFormat), string(DASH)) {
			panic("low_latency_dash_mode is only compatible with DASH outputs. manifest_format must include DASH")
		}

		if len(params.pipelineConfigDict.UTCTimings) == 0 {
			panic("For low_latency_dash_mode, the utc_timings must be set.")
		}
	}

	// Note that we remove the trailing slash from the output location, because
	// otherwise GCS would create a subdirectory whose name is "".
	outputLocation := strings.TrimSuffix(params.outputLocation, "/")

	// InputConfig contains inputs only.
	if len(cn.inputConfig.Inputs) > 0 {
		cn.appendNodesForInputsList(appendNodeParams{
			inputs:         cn.inputConfig.Inputs,
			outputLocation: outputLocation,
		})
	} else {
		// InputConfig contains multiperiod_inputs_list only.
		// Create one Transcoder node and one Packager node for each period.
		for i, singlePeriod := range cn.inputConfig.MultiPeriodInputsList {
			cn.appendNodesForInputsList(appendNodeParams{
				inputs:         singlePeriod.Inputs,
				outputLocation: outputLocation,
				periodDir:      fmt.Sprintf("period_%v", i+1),
				index:          i + 1,
			})
		}

		if cn.pipelineConfig.StreamingMode == VOD {
			// packageNodes = c.packagerNodes()
		}
	}

	return cn
}

type appendNodeParams struct {
	inputs         []Input
	outputLocation string
	periodDir      string
	index          int
}

func (c ControllerNode) appendNodesForInputsList(params appendNodeParams) {
}

func (cn ControllerNode) packagerNodes() []PackagerNode {
	var nodes []PackagerNode

	for _, node := range cn.nodes {
		if pn, _ := node.(*PackagerNode); pn != nil {
			nodes = append(nodes, *pn)
		}
	}

	return nodes
}

func (c ControllerNode) Stop() {
	// Implementation of the stop method
}

func (c ControllerNode) Close() {
	os.RemoveAll(c.tempDir)
}
