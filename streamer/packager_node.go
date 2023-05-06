package streamer

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func buildPath(outputLocation string, subPath string) string {
	// Sometimes the segment dir is empty.  This handles that special case.
	if subPath == "" {
		return outputLocation
	}

	// Don't use os.path.join, since URLs must use forward slashes and Streamer
	// could be used on Windows.
	if isUrl(outputLocation) {
		return strings.Join([]string{outputLocation, "/", subPath}, "")
	}

	return filepath.Join(outputLocation, subPath)
}

func isUrl(s string) bool {
	u, err := url.Parse(s)
	return err == nil && u.Scheme != "" && u.Host != ""
}

// A module that feeds information from two named pipes into shaka-packager.
type PackagerNode struct {
	NodeBase
	pipelineConfig PipelineConfig
	outputLocation string
	segmentDir     string
	OutputStreams  []MediaOutputStream
	index          int
	packager       string
}

func NewPackagerNode(pipelineConfig PipelineConfig, outputLocation string, streams []MediaOutputStream, index int, hermeticPackager string) *PackagerNode {
	pn := &PackagerNode{
		pipelineConfig: pipelineConfig,
		outputLocation: outputLocation,
		OutputStreams:  streams,
		index:          index,
		packager:       "packager",
		segmentDir:     buildPath(outputLocation, pipelineConfig.SegmentFolder),
	}

	if hermeticPackager != "" {
		pn.packager = hermeticPackager
	}

	return pn
}

func (pn *PackagerNode) Start() {
	args := []string{pn.packager}

	for _, stream := range pn.OutputStreams {
		args = append(args, pn.setupStream(stream))
	}

	if pn.pipelineConfig.Quiet {
		args = append(args, "--quiet") // Only output error logs
	}

	if pn.pipelineConfig.SegmentSize > 0 {
		args = append(args, "--segment_duration", strconv.FormatFloat(pn.pipelineConfig.SegmentSize, 'f', 2, 64))
	}

	if pn.pipelineConfig.StreamingMode == LIVE {
		args = append(args,
			// Number of seconds the user can rewind through backwards.
			"--time_shift_buffer_depth", strconv.Itoa(pn.pipelineConfig.AvailabilityWindow),
			// Number of segments preserved outside the current live window.
			// NOTE: This must not be set below 3, or the first segment in an HLS
			// playlist may become unavailable before the playlist is updated.
			"--preserved_segments_outside_live_window", "3",
			// Number of seconds of content encoded/packaged that is ahead of the
			// live edge.
			"--suggested_presentation_delay", strconv.Itoa(pn.pipelineConfig.PresentationDelay),
			// Number of seconds between manifest updates.
			"--minimum_update_period", strconv.Itoa(pn.pipelineConfig.UpdatePeriod),
		)
	}

	args = append(args, pn.setupManifestFormat()...)

	if pn.pipelineConfig.Encryption.Enable {
		args = append(args, pn.setupEncryption()...)
	}

	stdout := os.Stdout

	if pn.pipelineConfig.DebugLogs {
		logFile := fmt.Sprintf("PackagerNode-%d.log", pn.index)
		logF, err := os.Create(logFile)

		if err != nil {
			panic(fmt.Sprintf("failed to create Packager log file: %v", err))
		}

		defer logF.Close()
		stdout = logF
	}

	// start process
	pn.Process = pn.CreateProcess(BaseParams{args: args, stdout: stdout})
}

func (pn PackagerNode) setupStream(stream MediaOutputStream) string {
	input := stream.GetInput()
	ipcPipe := stream.GetIpcPipe()

	dict := map[string]string{
		"in":     ipcPipe.ReadEnd(),
		"stream": string(stream.GetType()),
	}

	if input.SkipEncryption > 0 {
		dict["skip_encryption"] = strconv.Itoa(input.SkipEncryption)
	}

	if input.DrmLabel != "" {
		dict["drm_label"] = input.DrmLabel
	}

	// Note: Shaka Packager will not accept 'und' as a language, but Shaka
	// Player will fill that in if the language metadata is missing from the
	// manifest/playlist.
	if input.Language != "" && input.Language != "und" {
		dict["language"] = input.Language
	}

	if pn.pipelineConfig.SegmentPerFile {
		p1 := stream.GetInitSegFile()
		p2 := stream.GetMediaSegFile()
		dict["init_segment"] = buildPath(pn.segmentDir, p1.WriteEnd())
		dict["segment_template"] = buildPath(pn.segmentDir, p2.WriteEnd())
	} else {
		p3 := stream.GetSingleSegFile()
		dict["output"] = buildPath(pn.segmentDir, p3.WriteEnd())
	}

	if stream.IsDashOnly() {
		dict["dash_only"] = "1"
	}

	// The format of this argument to Shaka Packager is a single string of
	// key=value pairs separated by commas.
	args := []string{}

	for key, value := range dict {
		args = append(args, key+"="+value)
	}

	return strings.Join(args, ",")
}

func (pn PackagerNode) setupManifestFormat() []string {
	args := []string{}

	if containsManifestFormat(pn.pipelineConfig.ManifestFormat, DASH) {
		if pn.pipelineConfig.UTCTimings != nil {
			utcTimings := []string{}
			for _, timing := range pn.pipelineConfig.UTCTimings {
				utcTimings = append(utcTimings, timing.SchemeIdUri+"="+timing.Value)
			}

			args = append(args, "--utc_timings", strings.Join(utcTimings, ","))
		}

		if pn.pipelineConfig.LowLatencyDashMode {
			args = append(args, "--low_latency_dash_mode=true")
		}

		if pn.pipelineConfig.StreamingMode == VOD {
			args = append(args, "--generate_static_live_mpd")
		}

		// Generate DASH manifest file.
		args = append(args, "--mpd_output", filepath.Join(pn.outputLocation, pn.pipelineConfig.DashOutput))
	}

	if containsManifestFormat(pn.pipelineConfig.ManifestFormat, HLS) {
		hlsPlaylistType := "VOD"
		if pn.pipelineConfig.StreamingMode == LIVE {
			hlsPlaylistType = "LIVE"
		}

		// Generate HLS playlist file(s).
		args = append(args, "--hls_playlist_type", hlsPlaylistType, "--hls_master_playlist_output", filepath.Join(pn.outputLocation, pn.pipelineConfig.HlsOutput))
	}

	return args
}

// Sets up encryption keys for raw encryption mode
func (pn PackagerNode) setupEncryptionKeys() []string {
	keys := []string{}

	for _, key := range pn.pipelineConfig.Encryption.Keys {
		keyStr := ""

		if key.Label != "" {
			keyStr = "label=" + key.Label + ":"
		}

		keyStr += "key_id=" + key.KeyID + ":key=" + key.Key
		keys = append(keys, keyStr)
	}

	return keys
}

// Sets up encryption of content.
func (pn PackagerNode) setupEncryption() []string {
	encryption := pn.pipelineConfig.Encryption
	args := []string{}

	if encryption.EncryptionMode == Widevine {
		args = []string{
			"--enable_widevine_encryption",
			"--key_server_url", encryption.KeyServerURL,
			"--content_id", encryption.ContentID,
			"--signer", encryption.Signer,
			"--aes_signing_key", encryption.SigningKey,
			"--aes_signing_iv", encryption.SigningIV,
		}
	} else if encryption.EncryptionMode == RAW {
		// raw key encryption mode
		args = []string{
			"--enable_raw_key_encryption",
			"--keys",
			strings.Join(pn.setupEncryptionKeys(), ","),
		}
		if encryption.IV != "" {
			args = append(args, "--iv", encryption.IV)
		}
		if encryption.PSSH != "" {
			args = append(args, "--pssh", encryption.PSSH)
		}
	}

	return args
}

func containsManifestFormat(slice []ManifestFormat, item ManifestFormat) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}

	return false
}
