package streamer

import (
	"fmt"
	"runtime"

	"github.com/creasty/defaults"
	"gopkg.in/dealancer/validate.v2"
)

// Define a new type called InputType, which is essentially a string.
type InputType string

const (
	FILE             InputType = "file"             // A track from a file. Usable only with VOD.
	LOOPED_FILE      InputType = "looped_file"      // A track from a file, looped forever by FFmpeg. Usable only with live. Does not support media_type of 'text'.
	WEBCAM           InputType = "webcam"           // A webcam device. Usable only with live. The device path should be given in the name field. For example, on Linux, this might be /dev/video0. Only supports media_type of 'video'.
	MICROPHONE       InputType = "microphone"       // A microphone device. Usable only with live. The device path should given in the name field. For example, on Linux, this might be "default". Only supports media_type of 'audio'.
	EXTERNAL_COMMAND InputType = "external_command" // An external command that generates a stream of audio or video. The command should be given in the name field, using shell quoting rules. The command should send its generated output to the path in the environment variable $SHAKA_STREAMER_EXTERNAL_COMMAND_OUTPUT, which Shaka Streamer set to the path to the output pipe. May require the user of extra_input_args if FFmpeg can't guess the format or framerate. Does not support media_type of 'text'.
)

// Define a new type called MediaType, which is essentially a string.
type MediaType string

// Define constants for each enumerated value of the MediaType.
const (
	AUDIO MediaType = "audio"
	VIDEO MediaType = "video"
	TEXT  MediaType = "text"
)

// An object representing a single input stream to Shaka Streamer.
type Input struct {
	// The type of the input.
	InputType InputType `yaml:"input_type"`

	/*
		Name of the input.

		 With inputType set to 'file', this is a path to a file name.

		 With inputType set to 'looped_file', this is a path to a file name to be
		 looped indefinitely in FFmpeg.

		 With inputType set to 'webcam', this is which webcam.  On Linux, this is a
		 path to the device node for the webcam, such as '/dev/video0'. On macOS, this
		 is a device name, such as 'default'.

		 With inputType set to 'external_command', this is an external command that
		 generates a stream of audio or video. The command will be parsed using shell
		 quoting rules. The command should send its generated output to the path in
		 the environment variable $SHAKA_STREAMER_EXTERNAL_COMMAND_OUTPUT, which Shaka
		 Streamer set to the path to the output pipe.
	*/
	Name string `yaml:"name" validate:"empty=false"`

	/*
		Extra input arguments needed by FFmpeg to understand the input.

		This allows you to take inputs that cannot be understand or detected
		automatically by FFmpeg.

		This string will be parsed using shell quoting rules.
	*/
	ExtraInputArgs string `yaml:"extra_input_args" default:""`

	// The media type of the input stream.
	MediaType MediaType `yaml:"media_type" validate:"empty=false"`

	/*
		The frame rate of the input stream, in frames per second.

			Only valid for media_type of 'video'.

			Can be auto-detected for some input types, but may be required for others.
			For example, required for input_type of 'external_command'.
	*/
	FrameRate float64 `yaml:"frame_rate"`

	/*
		The name of the input resolution (1080p, etc).

			Only valid for media_type of 'video'.

			Can be auto-detected for some input types, but may be required for others.
			For example, required for input_type of 'external_command'.
	*/
	Resolution VideoResolutionName `yaml:"resolution"`

	// The name of the input channel layout (stereo, surround, etc).
	ChannelLayout AudioChannelLayoutName `yaml:"channel_layout"`

	/*
		The track number of the input.

		  The track number is specific to the media_type.  For example, if there is one
		  video track and two audio tracks, media_type of 'audio' and track_num of '0'
		  indicates the first audio track, not the first track overall in that file.

		  If unspecified, track_num will default to 0, meaning the first track matching
		  the media_type field will be used.
	*/
	TrackNum int `yaml:"track_num" default:"0"`

	/*
		True if the input video is interlaced.

		  Only valid for media_type of 'video'.

		  If true, the video will be deinterlaced during transcoding.

		  Can be auto-detected for some input types, but may be default to False for
		  others.  For example, an input_type of 'external_command', it will default to False.
	*/
	IsInterlaced bool `yaml:"is_interlaced"`

	/*
		The language of an audio or text stream.

			With input_type set to 'file' or 'looped_file', this will be auto-detected.
			Otherwise, it will default to 'und' (undetermined).
	*/
	Language string `yaml:"language"`
	/*
		The start time of the slice of the input to use.

			Only valid for VOD and with input_type set to 'file'.

			Not supported with media_type of 'text'.
	*/
	StartTime string `yaml:"start_time"`

	/*
		The end time of the slice of the input to use.

			Only valid for VOD and with input_type set to 'file'.

			Not supported with media_type of 'text'.
	*/
	EndTime string `yaml:"end_time"`

	/*
		Optional value for a custom DRM label, which defines the encryption key
		  applied to the stream. If not provided, the DRM label is derived from stream
		  type (video, audio), resolutions, etc. Note that it is case sensitive.

		  Applies to 'raw' encryption_mode only.
	*/
	DrmLabel string `yaml:"drm_label"`

	// If set, no encryption of the stream will be made
	SkipEncryption int `yaml:"skip_encryption"`

	/*
		A list of FFmpeg filter strings to add to the transcoding of this input.

			Each filter is a single string.  For example, 'pad=1280:720:20:20'.

			Not supported with media_type of 'text'.
	*/
	Filters []string `yaml:"filters"`
}

func NewInput(inputType InputType, name string, mediaType MediaType, filters []string) *Input {
	i := &Input{
		InputType: inputType,
		Name:      name,
		MediaType: mediaType,
		Filters:   filters,
	}

	// Manually set defaults.
	if err := defaults.Set(i); err != nil {
		panic(err)
	}

	return i
}

// Set default values.
// https://stackoverflow.com/questions/56049589/what-is-the-way-to-set-default-values-on-keys-in-lists-when-unmarshalling-yaml-i
func (i *Input) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// set defaults.
	if err := defaults.Set(i); err != nil {
		panic(err)
	}

	type plain Input

	if err := unmarshal((*plain)(i)); err != nil {
		return err
	}

	// validations
	if err := validate.Validate(i); err != nil {
		panic(err)
	}

	return nil
}

// Defaults
func (i *Input) SetDefaults() {
	// Input type
	if defaults.CanUpdate(i.InputType) {
		i.InputType = FILE
	}

	// Check if track is available
	if !IsPresent(*i) {
		panic(NewInputNotFound(*i))
	}

	if i.MediaType == VIDEO {
		// These fields are required for video inputs.
		// We will attempt to auto-detect them if possible.

		if defaults.CanUpdate(i.IsInterlaced) {
			i.IsInterlaced = GetInterlaced(*i)
		}

		if defaults.CanUpdate(i.FrameRate) {
			i.FrameRate = GetFrameRate(*i)
			// FrameRate is required
			i.requireField("FrameRate")
		}

		if defaults.CanUpdate(i.Resolution) {
			i.Resolution = GetResolution(*i)
			// Resolution is required
			i.requireField("Resolution")
		}
	}

	if i.MediaType == AUDIO || i.MediaType == TEXT {
		if defaults.CanUpdate(i.Language) {
			language := GetLanguage(*i)
			if len(language) > 0 {
				i.Language = language
			} else {
				i.Language = "und"
			}
		}
	}

	if i.MediaType == AUDIO {
		if defaults.CanUpdate(i.ChannelLayout) {
			i.ChannelLayout = GetChannelLayout(*i)
			// ChannelLayout is required
			i.requireField("ChannelLayout")
		}
	}

	if i.MediaType == TEXT {
		if i.InputType != FILE {
			reason := fmt.Sprintf("text streams are not supported in input_type %s", i.InputType)
			i.disallowField("InputType", reason)
		}

		//  These fields are not supported with text, because we don't process or transcode it.
		reason := `not supported with media_type "text"`
		i.disallowField("StartTime", reason)
		i.disallowField("EndTime", reason)

		if len(i.Filters) > 0 {
			i.disallowField("Filters", reason)
		}
	}

	if i.InputType != FILE {
		// These fields are only valid for file inputs.
		reason := `only valid when input_type is "file"`
		i.disallowField("StartTime", reason)
		i.disallowField("EndTime", reason)
	}
}

// Set the name to a pipe path into which this input's contents are fed.
func (i *Input) resetName(pipePath string) {
	i.Name = pipePath
}

/*
Get an FFmpeg stream specifier for this input.

	For example, the first video track would be "v:0", and the 3rd text track
	would be "s:2".  Note that all track numbers are per media type in this
	format, not overall track numbers from the input file, and that they are
	indexed starting at 0.

	See also http://ffmpeg.org/ffmpeg.html#Stream-specifiers
*/
func (i Input) GetStreamSpecifier() string {
	if i.MediaType == VIDEO {
		return fmt.Sprintf("v:%d", i.TrackNum)
	} else if i.MediaType == AUDIO {
		return fmt.Sprintf("a:%d", i.TrackNum)
	} else if i.MediaType == TEXT {
		return fmt.Sprintf("s:%d", i.TrackNum)
	} else {
		panic("Unrecognized media_type! This should not happen.")
	}
}

/*
Get any required input arguments for this input.

These are like hard-coded extra_input_args for certain input types.
This means users don't have to know much about FFmpeg options to handle
these common cases.

Note that for types which support autodetect, these arguments must be
understood by ffprobe as well as ffmpeg.
*/
func (i Input) GetInputArgs() []string {
	argsMatrix := map[InputType]map[string][]string{
		WEBCAM: {
			"Linux": []string{
				"-f", "video4linux2",
			},
			"Darwin": []string{
				"-f", "avfoundation",
				"-framerate", "30",
			},
			"Windows": []string{
				"-f", "dshow",
			},
		},
		MICROPHONE: {
			"Linux": []string{
				"-f", "pulse",
			},
			"Darwin": []string{
				"-f", "avfoundation",
			},
			"Windows": []string{
				"-f", "dshow",
			},
		},
	}

	argsForInputType := argsMatrix[i.InputType]

	// If the input's type wasn't of what interests us.
	if argsForInputType == nil {
		return []string{}
	}

	args := argsForInputType[runtime.GOOS]
	if args == nil {
		panic(fmt.Sprintf("%v is not supported on this platform!", i.InputType))
	}

	return args
}

// An error raised when a required field is missing from the input.
func (i Input) requireField(fieldName string) {
	if !StructFieldHasValue(i, fieldName) {
		panic(NewMissingRequiredField(i, fieldName))
	}
}

// An error raised when a field is malformed.
func (i Input) disallowField(fieldName string, reason string) {
	if StructFieldHasValue(i, fieldName) {
		panic(NewMalformedField(i, fieldName, reason))
	}
}

func (i Input) GetResolution() *VideoResolution {
	return NewBitrateConfig().GetResolutionValue(i.Resolution)
}

func (i Input) GetChannelLayout() *AudioChannelLayout {
	return NewBitrateConfig().GetChannelLayoutValue(i.ChannelLayout)
}

// An object representing a single period in a multiperiod inputs list.
type SinglePeriod struct {
	Inputs []Input `yaml:"inputs"`
}

// An object representing the entire input config to Shaka Streamer.
type InputConfig struct {
	// A list of SinglePeriod objects
	MultiPeriodInputsList []SinglePeriod `yaml:"multiperiod_inputs_list"`

	// A list of Input objects
	Inputs []Input `yaml:"inputs"`
}

func NewInputConfig(inputs []Input, mpi []SinglePeriod) *InputConfig {
	i := &InputConfig{
		Inputs:                inputs,
		MultiPeriodInputsList: mpi,
	}

	// Manually set defaults.
	if err := defaults.Set(i); err != nil {
		panic(err)
	}

	return i
}

func (i *InputConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(i); err != nil {
		panic(err)
	}

	type plain InputConfig

	if err := unmarshal((*plain)(i)); err != nil {
		return err
	}

	// validations
	if err := validate.Validate(i); err != nil {
		panic(err)
	}

	return nil
}

/*
A constructor to check that either inputs or mutliperiod_inputs_list is provided,

	and produce a helpful error message in case both or none are provided.

	We need these checks before passing the input dictionary to the configuration.Base constructor,
	because it does not check for this 'exclusive or-ing' relationship between fields
*/
func (i *InputConfig) SetDefaults() {
	hasInputs := len(i.Inputs) > 0
	hasMultiPeriodInputsList := len(i.MultiPeriodInputsList) > 0

	//  Because these fields are not marked as required at the class level,
	// we need to check ourselves that one of them is provided.
	if hasInputs && hasMultiPeriodInputsList {
		panic(NewConflictingFields(*i, "Inputs", "MultiPeriodInputsList"))
	}

	if !hasInputs && !hasMultiPeriodInputsList {
		panic(NewMissingRequiredExclusiveFields(*i, "Inputs", "MultiPeriodInputsList"))
	}
}
