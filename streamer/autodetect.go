// A module to contain auto-detection logic; based on ffprobe.
package streamer

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// TYPES_WE_CANT_PROBE are input types that cannot be probed by ffprobe.
var TYPES_WE_CANT_PROBE = []InputType{
	EXTERNAL_COMMAND,
}

// HermeticFFProbe is a module level variable that might be set by the controller node
// if the user chooses to use the shaka streamer bundled binaries.
var HermeticFFProbe string = "ffprobe"

/*
Autodetect some feature of the input, if possible, using ffprobe.
Args:

	input (Input): An input object from input_configuration.

	field (str): A field to pass to ffprobe's -show_entries option.

Returns:

	The requested field from ffprobe as a string, or None if this fails.
*/
func probe(i *Input, field string) (string, error) {
	if ContainsInputType(TYPES_WE_CANT_PROBE, i.InputType) {
		// Not supported for this type.
		return "", fmt.Errorf("%s not supported", i.InputType)
	}

	args := []string{
		// Probe this input file
		HermeticFFProbe,
		i.Name,
	}

	// Add any required input arguments for this input type
	args = append(args, i.getInputArgs()...)

	args = append(args,
		// Specifically, this stream
		"-select_streams", i.getStreamSpecifier(),
		// Show the needed metadata only
		"-show_entries", field,
		// Print the metadata in a compact form, which is easier to parse
		"-of", "compact=p=0:nk=1",
	)

	cmd := exec.Command(args[0], args[1:]...)
	outputBytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running command: %v", err)
	}

	// The output is either some probe information or just a blank line.
	s := strings.TrimSpace(string(outputBytes))
	if s == "" {
		return "", nil
	}

	// With certain container formats, ffprobe returns a duplicate
	// output and some empty lines in between. Issue #119
	lines := strings.Split(s, "\n")
	s = lines[0]

	//  Webcams on Linux seem to behave badly if the device is rapidly opened and
	//  closed.  Therefore, sleep for 1 second after a webcam probe.
	if i.InputType == WEBCAM {
		time.Sleep(time.Second)
	}

	return s, nil
}

// IsPresent returns true if the stream for this input is indeed found.
// If we can't probe this input type, assume it is present.
func IsPresent(i *Input) bool {
	_, err := probe(i, "stream=index")
	return err == nil
}

// GetLanguage returns the autodetected the language of the input.
func GetLanguage(i *Input) *string {
	l, _ := probe(i, "stream_tags=language")
	return &l
}

// GetInterlaced returns true if we detect that the input is interlaced.
func GetInterlaced(i *Input) bool {
	s, err := probe(i, "stream=field_order")
	if err != nil {
		return false
	}

	// These constants represent the order of the fields (2 fields per frame) of
	// different types of interlaced video.  They can be found in
	// https://www.ffmpeg.org/ffmpeg-codecs.html under the description of the
	// field_order option.  Anything else (including None) should be considered
	// progressive (non-interlaced) video.

	return ContainsString([]string{"tt", "bb", "tb", "bt"}, s)
}

func GetFrameRate(i *Input) *float64 {
	s, err := probe(i, "stream=avg_frame_rate")
	if err != nil || len(s) < 1 {
		return nil
	}
	// This string is the framerate in the form of a fraction, such as '24/1' or
	// '30000/1001'. Occasionally, there is a pipe after the framerate, such as
	// '32700/1091|'. We must split it into pieces and do the division to get a
	// float.
	pieces := strings.Split(strings.TrimSuffix(s, "|"), "/")
	if len(pieces) == 1 {
		frameRate, _ := strconv.ParseFloat(pieces[0], 64)
		return &frameRate
	} else {
		numerator, _ := strconv.ParseFloat(pieces[0], 64)
		denominator, _ := strconv.ParseFloat(pieces[1], 64)
		frameRate := numerator / denominator

		// The detected frame rate for interlaced content is twice what it should be.
		// It's actually the field rate, where it takes two interlaced fields to make
		// a frame. Because we have to know if it's interlaced already, we must
		// assert that is_interlaced has been set before now.
		if i.IsInterlaced {
			frameRate /= 2.0
		}

		return &frameRate
	}
}

// Returns the autodetected resolution of the input.
func GetResolution(i *Input) *VideoResolutionName {
	// resolutionString
	rs, err := probe(i, "stream=width,height")

	if err != nil {
		return nil
	}

	// This is the resolution of the video in the form of 'WIDTH|HEIGHT'.  For
	// example, '1920|1080'.  Occasionally, there is a pipe after the resolution,
	// such as '1920|1080|'.  We have to split up width and height and match that
	// to a named resolution.
	// resolutionArray
	ra := strings.Split(strings.TrimRight(rs, "|"), "|")
	width, _ := strconv.Atoi(ra[0])
	height, _ := strconv.Atoi(ra[1])

	for key, bucket := range DefaultVideoResolutions {
		// The first bucket this fits into is the one.
		if width <= bucket.MaxWidth && height <= bucket.MaxHeight && i.FrameRate <= bucket.MaxFrameRate {
			return &key
		}
	}

	return nil
}

// Returns the autodetected channel count of the input.
func GetChannelLayout(i *Input) *AudioChannelLayoutName {
	// channelCountString
	cs, err := probe(i, "stream=channels")

	if err != nil {
		return nil
	}

	// channelCount
	cc, _ := strconv.Atoi(cs)
	for key, bucket := range DefaultAudioChannelLayouts {
		if cc <= bucket.MaxChannels {
			return &key
		}
	}

	return nil
}
