package streamer

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/creasty/defaults"
	"gopkg.in/dealancer/validate.v2"
)

// A wrapper that can be used in Field() to require a bitrate string.
type BitrateString string

func (b BitrateString) Name() string {
	return "bitrate string"
}

func (bs BitrateString) Validate() error {
	if !regexp.MustCompile(`^\d.*[kKmM]$`).MatchString(string(bs)) {
		return errors.New("not a bitrate string (e.g. 500k or 7.5M)")
	}

	return nil
}

type AudioCodecName string

type VideoResolutionName string

type AudioChannelLayoutName string

const (
	AAC  AudioCodecName = "aac"
	OPUS AudioCodecName = "opus"
	AC3  AudioCodecName = "ac3"
	EAC3 AudioCodecName = "eac3"
)

type AudioCodec struct {
	Name AudioCodecName
}

func NewAudioCodec(name AudioCodecName) *AudioCodec {
	return &AudioCodec{
		Name: name,
	}
}

// Returns True if this codec is hardware accelerated.
func (a *AudioCodec) IsHardwareAccelerated() bool {
	return false
}

// Returns a codec string accepted by FFmpeg for this codec.
func (a *AudioCodec) GetFFmpegCodecString(hwaccelAPI string) string {
	// Returns a codec string accepted by FFmpeg for this codec.
	// FFmpeg warns:
	//   The encoder 'opus' is experimental but experimental codecs are not
	//   enabled, add '-strict -2' if you want to use it. Alternatively use the
	//   non experimental encoder 'libopus'.
	if a.Name == OPUS {
		return "libopus"
	}

	return string(a.Name)
}

// Returns an FFmpeg output format suitable for this codec.
func (a *AudioCodec) GetOutputFormat() string {
	// Returns an FFmpeg output format suitable for this codec.
	// TODO: consider Opus in mp4 by default
	// TODO(#31): add support for configurable output format per-codec
	if a.Name == OPUS {
		return "webm"
	} else if a.Name == AAC || a.Name == AC3 || a.Name == EAC3 {
		return "mp4"
	} else {
		panic(fmt.Sprintf("No mapping for output format for codec %s", a))
	}
}

type VideoCodecName string

const (
	H264 VideoCodecName = "h264" // H264, also known as AVC
	VP9  VideoCodecName = "vp9"  // VP9
	AV1  VideoCodecName = "av1"  // AV1
	HEVC VideoCodecName = "hevc" // HEVC, also known as h.265
)

type VideoCodec struct {
	Name  VideoCodecName
	HWAcc bool
}

func NewVideoCodec(name VideoCodecName) *VideoCodec {
	return &VideoCodec{
		Name:  name,
		HWAcc: strings.HasPrefix(string(name), "hw:"),
	}
}

// Returns True if this codec is hardware accelerated.
func (v *VideoCodec) IsHardwareAccelerated() bool {
	return v.HWAcc
}

// Returns a codec string accepted by FFmpeg for this codec
func (v *VideoCodec) GetFFmpegCodecString(hwaccelApi string) string {
	// Returns a codec string accepted by FFmpeg for this codec.
	if v.HWAcc {
		// Overwrite the _hw_acc variable for this codec.
		// name := strings.TrimPrefix(string(v.name), "hw:")
		return string(v.Name) + "_" + hwaccelApi
	}

	return string(v.Name)
}

// Returns an FFmpeg output format suitable for this codec.
func (c *VideoCodec) GetOutputFormat() string {
	// TODO: consider VP9 in mp4 by default
	// TODO(#31): add support for configurable output format per-codec
	switch c.Name {
	case VP9:
		return "webm"
	case H264, HEVC, AV1:
		return "mp4"
	default:
		panic(fmt.Sprintf("No mapping for output format for codec %v", c))
	}
}

type AudioChannelLayout struct {
	/*[]
	The maximum number of channels in this layout.

	For example, the maximum number of channels for stereo is 2.
	*/
	MaxChannels int `yaml:"max_channels"`

	/*
		A map of audio codecs to the target bitrate for this channel layout.
		   For example, in stereo, AAC can have a different bitrate from Opus.
		   This value is a string in bits per second, with the suffix 'k' or 'M' for
		   kilobits per second or megabits per second.
		   For example, this could be '500k' or '7.5M'.
	*/
	Bitrates map[AudioCodecName]BitrateString `yaml:"bitrates" validate:"empty=false"`
}

func NewAudioChannelLayout(maxChannels int, bitrates map[AudioCodecName]BitrateString) *AudioChannelLayout {
	return &AudioChannelLayout{
		MaxChannels: maxChannels,
		Bitrates:    bitrates,
	}
}

var DefaultAudioChannelLayouts = &map[AudioChannelLayoutName]*AudioChannelLayout{
	"mono": NewAudioChannelLayout(1, map[AudioCodecName]BitrateString{
		AAC:  "64k",
		OPUS: "32k",
		AC3:  "96k",
		EAC3: "48k",
	}),
	"stereo": NewAudioChannelLayout(2, map[AudioCodecName]BitrateString{
		AAC:  "128k",
		OPUS: "64k",
		AC3:  "192k",
		EAC3: "96k",
	}),
	"surround": NewAudioChannelLayout(6, map[AudioCodecName]BitrateString{
		AAC:  "256k",
		OPUS: "128k",
		AC3:  "384k",
		EAC3: "192k",
	}),
}

type VideoResolution struct {
	// The maximum width in pixels for this named resolution.
	MaxWidth int `yaml:"max_width"`

	// The maximum height in pixels for this named resolution.
	MaxHeight int `yaml:"max_height"`

	/*
		The maximum frame rate in frames per second for this named resolution.

		  By default, the max frame rate is unlimited.
	*/
	MaxFrameRate float64 `yaml:"max_frame_rate"`

	Bitrates map[VideoCodecName]BitrateString `yaml:"bitrates"`
}

// validations
func (vr *VideoResolution) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(vr); err != nil {
		panic(err)
	}

	type plain VideoResolution

	if err := unmarshal((*plain)(vr)); err != nil {
		return err
	}

	if err := validate.Validate(vr); err != nil {
		panic(err)
	}

	return nil
}

// Default MaxFrameRate
func (vr *VideoResolution) SetDefaults() {
	// By default, the max frame rate is unlimited.
	if defaults.CanUpdate(vr.MaxFrameRate) {
		vr.MaxFrameRate = math.Inf(1)
	}
}

func NewVideoResolution(maxWidth int, maxHeight int, maxFrameRate float64, bitrates map[VideoCodecName]BitrateString) *VideoResolution {

	return &VideoResolution{
		MaxWidth:     maxWidth,
		MaxHeight:    maxHeight,
		MaxFrameRate: maxFrameRate,
		Bitrates:     bitrates,
	}
}

// Default bitrates and resolutions are tracked internally at
// go/shaka-streamer-bitrates.
// These are common resolutions, and the bitrates per codec are derived from
// internal encoding guidelines.
var DefaultVideoResolutions = &map[VideoResolutionName]*VideoResolution{
	"144p": NewVideoResolution(256, 144, 30, map[VideoCodecName]BitrateString{
		H264: "108k",
		VP9:  "96k",
		HEVC: "96k",
		AV1:  "72k",
	}),
	"240p": NewVideoResolution(426, 240, 30, map[VideoCodecName]BitrateString{
		H264: "242k",
		VP9:  "151k",
		HEVC: "151k",
		AV1:  "114k",
	}),
	"360p": NewVideoResolution(640, 360, 30, map[VideoCodecName]BitrateString{
		H264: "400k",
		VP9:  "277k",
		HEVC: "277k",
		AV1:  "210k",
	}),
	"480p": NewVideoResolution(854, 480, 30, map[VideoCodecName]BitrateString{
		H264: "1M",
		VP9:  "512k",
		HEVC: "512k",
		AV1:  "389k",
	}),
	"576p": NewVideoResolution(1024, 576, 30, map[VideoCodecName]BitrateString{ // PAL analog broadcast TV resolution
		H264: "1.5M",
		VP9:  "768k",
		HEVC: "768k",
		AV1:  "450k",
	}),
	"720p": NewVideoResolution(1280, 720, 30, map[VideoCodecName]BitrateString{
		H264: "2M",
		VP9:  "1M",
		HEVC: "1M",
		AV1:  "512k",
	}),
	"720p-hfr": NewVideoResolution(1280, 720, 0, map[VideoCodecName]BitrateString{
		H264: "3M",
		VP9:  "2M",
		HEVC: "2M",
		AV1:  "778k",
	}),
	"1080p": NewVideoResolution(1920, 1080, 30, map[VideoCodecName]BitrateString{
		H264: "4M",
		VP9:  "2M",
		HEVC: "2M",
		AV1:  "850k",
	}),
	"1080p-hfr": NewVideoResolution(1920, 1080, 0, map[VideoCodecName]BitrateString{
		H264: "5M",
		VP9:  "3M",
		HEVC: "3M",
		AV1:  "1M",
	}),
	"1440p": NewVideoResolution(2560, 1440, 30, map[VideoCodecName]BitrateString{
		H264: "9M",
		VP9:  "6M",
		HEVC: "6M",
		AV1:  "3.5M",
	}),
	"1440p-hfr": NewVideoResolution(2560, 1440, 0, map[VideoCodecName]BitrateString{
		H264: "14M",
		VP9:  "9M",
		HEVC: "9M",
		AV1:  "5M",
	}),
	"4k": NewVideoResolution(4096, 2160, 30, map[VideoCodecName]BitrateString{
		H264: "17M",
		VP9:  "12M",
		HEVC: "12M",
		AV1:  "6M",
	}),
	"4k-hfr": NewVideoResolution(4096, 2160, 0, map[VideoCodecName]BitrateString{
		H264: "25M",
		VP9:  "18M",
		HEVC: "18M",
		AV1:  "9M",
	}),
	"8k": NewVideoResolution(8192, 4320, 30, map[VideoCodecName]BitrateString{
		H264: "40M",
		VP9:  "24M",
		HEVC: "24M",
		AV1:  "12M",
	}),
	"8k-hfr": NewVideoResolution(8192, 4320, 0, map[VideoCodecName]BitrateString{
		H264: "60M",
		VP9:  "36M",
		HEVC: "36M",
		AV1:  "18M",
	}),
}

type BitrateConfig struct {
	/*
		A map of named channel layouts.

		  For example, the key would be a name like "stereo", and the value would be an
		  object with all the parameters of how stereo audio would be encoded (2
		  channels max, bitrates, etc.).
	*/
	AudioChannelLayouts map[AudioChannelLayoutName]*AudioChannelLayout `yaml:"audio_channel_layouts"`

	/*
		A map of named resolutions.

		  For example, the key would be a name like "1080p", and the value would be an
		  object with all the parameters of how 1080p video would be encoded (max size,
		  bitrates, etc.)
	*/
	VideoResolutions map[VideoResolutionName]*VideoResolution `yaml:"video_resolutions"`
}

func NewBitrateConfig() *BitrateConfig {
	return &BitrateConfig{
		AudioChannelLayouts: *DefaultAudioChannelLayouts,
		VideoResolutions:    *DefaultVideoResolutions,
	}
}

func (bc *BitrateConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// set defaults.
	if err := defaults.Set(bc); err != nil {
		panic(err)
	}

	type plain BitrateConfig

	if err := unmarshal((*plain)(bc)); err != nil {
		return err
	}

	// validations
	if err := validate.Validate(bc); err != nil {
		panic(err)
	}

	return nil
}

// Defaults
func (bc *BitrateConfig) SetDefaults() {
	if defaults.CanUpdate(bc.VideoResolutions) {
		bc.VideoResolutions = *DefaultVideoResolutions
	}

	if defaults.CanUpdate(bc.AudioChannelLayouts) {
		bc.AudioChannelLayouts = *DefaultAudioChannelLayouts
	}
}

func (bc *BitrateConfig) GetResolutionValue(resolution VideoResolutionName) *VideoResolution {
	return bc.VideoResolutions[resolution]
}

func (bc *BitrateConfig) GetChannelLayoutValue(channelLayout AudioChannelLayoutName) *AudioChannelLayout {
	return bc.AudioChannelLayouts[channelLayout]
}

func (bc *BitrateConfig) VideoResolutionKeys() []VideoResolutionName {
	keys := make([]VideoResolutionName, 0, len(bc.VideoResolutions))

	for key := range bc.VideoResolutions {
		keys = append(keys, key)
	}

	return keys
}

func (bc *BitrateConfig) ChannelLayoutKeys() []AudioChannelLayoutName {
	keys := make([]AudioChannelLayoutName, 0, len(bc.AudioChannelLayouts))

	for key := range bc.AudioChannelLayouts {
		keys = append(keys, key)
	}

	return keys
}
