package streamer

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"runtime"

	"github.com/creasty/defaults"
	"gopkg.in/dealancer/validate.v2"
)

type StreamingMode string

const (
	LIVE StreamingMode = "live" // Indicates a live stream, which has no end.
	VOD  StreamingMode = "vod"  // Indicates a video-on-demand (VOD) stream, which is finite.
)

type ManifestFormat string

const (
	DASH ManifestFormat = "dash"
	HLS  ManifestFormat = "hls"
)

type ProtectionScheme string

const (
	CENC ProtectionScheme = "cenc" // AES-128-CTR mode.
	CBCS ProtectionScheme = "cbcs" // AES-128-CBC mode with pattern encryption.
)

type ProtectionSystem string

const (
	WIDEVINE  ProtectionSystem = "Widevine"
	FAIRPLAY  ProtectionSystem = "FairPlay"
	PLAYREADY ProtectionSystem = "PlayReady"
	MARLIN    ProtectionSystem = "Marlin"
	COMMON    ProtectionSystem = "CommonSystem"
)

type EncryptionMode string

const (
	Widevine EncryptionMode = "widevine" // Widevine key server mode
	RAW      EncryptionMode = "raw"      // Raw key mode
)

// An object containing the attributes for a DASH MPD UTCTiming element
type UtcTimingPair struct {
	// SchemeIdUri attribute to be used for the UTCTiming element
	SchemeIdUri string `yaml:"scheme_id_uri"`
	// Value attribute to be used for the UTCTiming element
	Value string `yaml:"value"`
}

// An object representing a list of keys for Raw key encryption
type RawKeyConfig struct {
	/*
		An arbitary string or a predefined DRM label like AUDIO, SD, HD, etc.
		  If not specified, indicates the default key and key_id.
	*/
	Label string `yaml:"label"`
	// A key identifier as a 32-digit hex string
	KeyID string `yaml:"key_id" validate:"empty=false"`
	// The encryption key to use as a 32-digit hex string
	Key string `yaml:"key" validate:"empty=false"`
}

// Validations
func (rc *RawKeyConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain RawKeyConfig

	if err := unmarshal((*plain)(rc)); err != nil {
		return err
	}

	// validations
	if err := validate.Validate(rc); err != nil {
		panic(err)
	}

	return nil
}

// An object representing the encryption config for Shaka Streamer.
type EncryptionConfig struct {
	// If true, encryption is enabled.
	// Otherwise, all other encryption settings are ignored.
	Enable bool `yaml:"enable" default:"false"`

	// Encryption mode to use. By default it is widevine but can be changed to raw.
	EncryptionMode EncryptionMode `yaml:"encryption_mode"`

	// Protection Systems to be generated. Supported protection systems include
	// Widevine, PlayReady, FairPlay, Marin and CommonSystem.
	ProtectionSystems []ProtectionSystem `yaml:"protection_systems"`
	/*
	   One or more concatenated PSSH boxes in hex string format. If this and
	    `protection_systems` is not specified, a v1 common PSSH box will be generated.

	    Applies to 'raw' encryption_mode only.
	*/
	PSSH string `yaml:"pssh"`

	/*
		IV in hex string format. If not specified, a random IV will be generated.
			Applies to 'raw' encryption_mode only.
	*/
	IV string `yaml:"iv"`

	/*
		A list of encryption keys to use.
		  Applies to 'raw' encryption_mode only.
	*/
	Keys []RawKeyConfig `yaml:"keys"`

	/*
		The content ID, in hex.
		  If omitted, a random content ID will be chosen for you.

		  Applies to 'widevine' encryption_mode only.
	*/
	ContentID string `yaml:"content_id"`

	/*
		The URL of your key server.

			This is used to generate an encryption key. By default, it is Widevine's UAT server.

			Applies to 'widevine' encryption_mode only.
	*/
	KeyServerURL string `yaml:"key_server_url" validate:"empty=true | format=url"`

	/*
		The name of the signer when authenticating to the key server.

		  Applies to 'widevine' encryption_mode only.

		  Defaults to the Widevine test account.
	*/
	Signer string `yaml:"signer"`
	/*
	   The signing key, in hex, when authenticating to the key server.

	     Applies to 'widevine' encryption_mode only.

	     Defaults to the Widevine test account's key.
	*/
	SigningKey string `yaml:"signing_key"`

	/*
		The signing IV, in hex, when authenticating to the key server.

		  Applies to 'widevine' encryption_mode only.

		  Defaults to the Widevine test account's IV.
	*/
	SigningIV string `yaml:"signing_iv"`

	// The protection scheme (cenc or cbcs) to use when encrypting.
	ProtectionScheme ProtectionScheme `yaml:"protection_scheme"`

	// The seconds of unencrypted media at the beginning of the stream.
	ClearLead int `yaml:"clear_lead" default:"10"`
}

func (e *EncryptionConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// set defaults.
	if err := defaults.Set(e); err != nil {
		panic(err)
	}

	type plain EncryptionConfig

	if err := unmarshal((*plain)(e)); err != nil {
		return err
	}

	// validations
	if err := validate.Validate(e); err != nil {
		panic(err)
	}

	return nil
}

// Set Dynamic Defaults
func (e *EncryptionConfig) SetDefaults() {
	if defaults.CanUpdate(e.EncryptionMode) {
		e.EncryptionMode = Widevine
	}

	if defaults.CanUpdate(e.ContentID) {
		// A randomly-chosen content ID in hex.
		var randomContentID = func() string {
			bytes := make([]byte, 16)
			_, err := rand.Read(bytes)
			if err != nil {
				panic(err)
			}

			return base64.StdEncoding.EncodeToString(bytes)
		}()

		e.ContentID = randomContentID
	}

	if defaults.CanUpdate(e.EncryptionMode) {
		e.EncryptionMode = Widevine
	}

	if defaults.CanUpdate(e.ProtectionScheme) {
		e.ProtectionScheme = CENC
	}

	// Credentials for the Widevine test account.

	if defaults.CanUpdate(e.KeyServerURL) {
		// The Widevine UAT server URL.
		e.KeyServerURL = "https://license.uat.widevine.com/cenc/getcontentkey/widevine_test"
	}

	if defaults.CanUpdate(e.Signer) {
		e.Signer = "widevine_test"
	}

	if defaults.CanUpdate(e.SigningKey) {
		e.SigningKey = "1ae8ccd0e7985cc0b6203a55855a1034afc252980e970ca90e5202689f947ab9"
	}

	if defaults.CanUpdate(e.SigningIV) {
		e.SigningIV = "d58ce954203b7c9a9a9d467f59839249"
	}

	// Don't do any further checks if encryption is disabled
	if !e.Enable {
		return
	}

	if e.EncryptionMode == Widevine {
		fieldNames := []string{"Keys", "PSSH", "IV"}

		for _, fieldName := range fieldNames {
			// skip if keys length is zero
			if fieldName == "Keys" && len(e.Keys) == 0 {
				continue
			}

			if StructFieldHasValue(*e, fieldName) {
				reason := fmt.Sprintf("cannot be set when encryption_mode is \"%s\"", e.Keys)
				panic(NewMalformedField(*e, fieldName, reason))
			}
		}
	} else if e.EncryptionMode == RAW {
		// Check at least one key has been specified
		if len(e.Keys) == 0 {
			reason := "at least one key must be specified"
			panic(NewMalformedField(*e, "Keys", reason))
		}
	}
}

// An object representing the entire pipeline config for Shaka Streamer.
type PipelineConfig struct {
	// The streaming mode, which can be either 'vod' or 'live'.
	StreamingMode StreamingMode `yaml:"streaming_mode" validate:"empty=false"`

	/*
		If true, reduce the level of output.

		Only errors will be shown in quiet mode.
	*/
	Quiet bool `yaml:"quiet" default:"false"`

	/*
		If true, output simple log files from each node.

		  No control is given over log filenames.  Logs are written to the current
		  working directory.  We do not yet support log rotation.  This is meant only
		  for debugging.
	*/
	DebugLogs bool `yaml:"debug_logs" default:"false"`
	/*
		The FFmpeg hardware acceleration API to use with hardware codecs.

		A per-platform default will be chosen if this field is omitted.

		See documentation here: https://trac.ffmpeg.org/wiki/HWAccelIntro
	*/
	HWAccelAPI string `yaml:"hwaccel_api"`

	/*
		A list of resolution names to encode.

		  Any resolution greater than the input resolution will be ignored, to avoid
		  upscaling the content. This also allows you to reuse a pipeline config for
		  multiple inputs.

		  If not set, it will default to a list of all the (VideoResolutionName)s
		  defined in the bitrate configuration.
	*/
	Resolutions []VideoResolutionName `yaml:"resolutions"`

	/*
		A list of channel layouts to encode.

		  Any channel count greater than the input channel count will be ignored.

		  If not set, it will default to a list of all the (AudioChannelLayoutName)s
		  defined in the bitrate configuration.
	*/
	ChannelLayouts []AudioChannelLayoutName `yaml:"channel_layouts"`

	// The audio codecs to encode with.
	AudioCodecs []AudioCodecName `yaml:"audio_codecs"`

	/*
		The video codecs to encode with.

			Note that the prefix "hw:" indicates that a hardware encoder should be used.
	*/
	VideoCodecs []VideoCodecName `yaml:"video_codecs"`

	/*
		A list of manifest formats (dash or hls) to create.

		  By default, this will create both.
	*/
	ManifestFormat []ManifestFormat `yaml:"manifest_format"`

	// Output filename for the DASH manifest, if created.
	DashOutput string `yaml:"dash_output" default:"dash.mpd"`

	// Output filename for the HLS master playlist, if created.
	HlsOutput string `yaml:"hls_output" default:"hls.m3u8"`

	// Sub-folder for segment output (or blank for none
	SegmentFolder string `yaml:"segment_folder" default:""`

	// The length of each segment in seconds.
	SegmentSize float64 `yaml:"segment_size" default:"4"`

	/*
		If true, force each segment to be in a separate file.

		  Must be true for live content.
	*/
	SegmentPerFile bool `yaml:"segment_per_file" default:"false"`

	// The number of seconds a segment remains available.
	AvailabilityWindow int `yaml:"availability_window" default:"300"`

	// How far back from the live edge the player should be, in seconds.
	PresentationDelay int `yaml:"presentation_delay" default:"30"`

	// How often the player should fetch a new manifest, in seconds.
	UpdatePeriod int `yaml:"update_period" default:"8"`

	// Encryption settings.
	Encryption EncryptionConfig `yaml:"encryption"`

	// TODO: Generalize this to low_latency_mode once LL-HLS is supported by Packager
	LowLatencyDashMode bool `yaml:"low_latency_dash_mode" default:"false"`

	/*
		UTCTiming schemeIdUri and value pairs for the DASH MPD.

		  If multiple UTCTiming pairs are provided for redundancy,
		  list the pairs in the order of preference.

		  Must be set for LL-DASH streaming.
	*/
	UTCTimings []UtcTimingPair `yaml:"utc_timings"`
}

// Validations
func (p *PipelineConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// set defaults.
	if err := defaults.Set(p); err != nil {
		panic(err)
	}

	type plain PipelineConfig

	if err := unmarshal((*plain)(p)); err != nil {
		return err
	}

	// validations
	if err := validate.Validate(p); err != nil {
		panic(err)
	}

	return nil
}

// Set Dynamic Defaults
func (p *PipelineConfig) SetDefaults() {
	// The default hardware acceleration API to use, per platform.
	if defaults.CanUpdate(p.HWAccelAPI) {
		var defaultHwAccelAPI = func() string {
			switch runtime.GOOS {
			case "linux":
				return "vaapi"
			case "darwin":
				return "videotoolbox"
			default:
				return ""
			}
		}()

		p.HWAccelAPI = defaultHwAccelAPI
	}

	if defaults.CanUpdate(p.AudioCodecs) {
		p.AudioCodecs = []AudioCodecName{AAC}
	}

	if defaults.CanUpdate(p.VideoCodecs) {
		p.VideoCodecs = []VideoCodecName{H264}
	}

	if defaults.CanUpdate(p.ManifestFormat) {
		p.ManifestFormat = []ManifestFormat{DASH, HLS}
	}

	if defaults.CanUpdate(p.Encryption) {
		p.Encryption = EncryptionConfig{}
	}

	/*
		Set the default values of the resolutions and channel_layouts
		to the values we have in the bitrate configuration.
		We need the 'type: ignore' here because mypy thinks these variables are lists
		of VideoResolutionName and AudioChannelLayoutName and not Field variables.
	*/
	if defaults.CanUpdate(p.Resolutions) {
		p.Resolutions = NewBitrateConfig().VideoResolutionKeys()
	}

	if defaults.CanUpdate(p.ChannelLayouts) {
		p.ChannelLayouts = NewBitrateConfig().ChannelLayoutKeys()
	}

	if p.StreamingMode == LIVE && !p.SegmentPerFile {
		reason := `must be true when streaming_mode is "live"`
		panic(NewMalformedField(*p, "SegmentPerFile", reason))
	}
}

func (p *PipelineConfig) GetResolutions() []*VideoResolution {
	resolutions := make([]*VideoResolution, 0, len(p.Resolutions))

	for _, name := range p.Resolutions {
		resolutions = append(resolutions, NewBitrateConfig().GetResolutionValue(name))
	}

	return resolutions
}

func (p *PipelineConfig) GetChannelLayouts() []*AudioChannelLayout {
	layouts := make([]*AudioChannelLayout, 0, len(p.ChannelLayouts))

	for _, name := range p.ChannelLayouts {
		layouts = append(layouts, NewBitrateConfig().GetChannelLayoutValue(name))
	}

	return layouts
}
