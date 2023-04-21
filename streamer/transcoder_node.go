package streamer

import (
	"fmt"
	"strconv"
	"strings"
)

// A module that pushes input to ffmpeg to transcode into various formats.
type TranscoderNode struct {
	NodeBase
	inputs         []Input
	pipelineConfig PipelineConfig
	outputs        []MediaOutputStream
	index          int
	ffmpeg         string
}

func NewTranscoderNode(inputs []Input, pipelineConfig PipelineConfig, outputs []MediaOutputStream, index int, hermeticFFmpeg string) *TranscoderNode {
	n := &TranscoderNode{
		inputs:         inputs,
		pipelineConfig: pipelineConfig,
		outputs:        outputs,
		index:          index,
		ffmpeg:         "ffmpeg",
	}

	if hermeticFFmpeg != "" {
		n.ffmpeg = hermeticFFmpeg
	}

	return n
}

func (t *TranscoderNode) Start() {
	args := []string{
		t.ffmpeg,
		// Do not prompt for output files that already exist. Since we created
		// the named pipe in advance, it definitely already exists. A prompt
		// would block ffmpeg to wait for user input.
		"-y",
	}

	if t.pipelineConfig.Quiet {
		args = append(args, []string{
			// Suppresses all messages except errors.
			// Without this, a status line will be printed by default showing
			// progress and transcoding speed.
			"-loglevel", "error",
		}...)
	}

	for _, output := range t.outputs {
		if output.IsHardwareAccelerated() && t.pipelineConfig.HWAccelAPI == "vaapi" {
			args = append(args, []string{
				// Hardware acceleration args.
				// TODO(#17): Support multiple VAAPI devices.
				"-vaapi_device", "/dev/dri/renderD128",
			}...)
		}
	}

	for _, input := range t.inputs {
		// Get any required input arguments for this input.
		// These are like hard-coded extra_input_args for certain input types.
		// This means users don't have to know much about FFmpeg options to handle
		// these common cases.
		args = append(args, input.GetInputArgs()...)

		// The config file may specify additional args needed for this input.
		// This allows, for example, an external-command-type input to generate
		// almost anything ffmpeg could ingest.  The extra args need to be parsed
		// from a string into an argument array.  Note that shlex.split on an empty
		// string will produce an empty array.
		args = append(args, strings.Fields(input.ExtraInputArgs)...)

		if input.InputType == LOOPED_FILE {
			// These are handled here instead of in get_input_args() because these
			// arguments are specific to ffmpeg and are not understood by ffprobe.
			args = append(args, []string{
				// Loop the input forever.
				"-stream_loop", "-1",
				// Read input in real time; don't go above 1x processing speed.
				"-re",
			}...)
		}

		if t.pipelineConfig.StreamingMode == LIVE {
			args = append(args, []string{
				// A larger queue to buffer input from the pipeline (default is 8).
				// This is in packets, but for raw images, that means frames.  A
				// 720p PPM frame is 2.7MB, and a 1080p PPM is 6.2MB.  The entire
				// queue, when full, must fit into memory.
				"-thread_queue_size", "200",
			}...)
		}

		if input.StartTime != "" {
			args = append(args, []string{
				// Encode from intended starting time of the input.
				"-ss", input.StartTime,
			}...)
		}
		if input.EndTime != "" {
			args = append(args, []string{
				// Encode until intended ending time of the input.
				"-to", input.EndTime,
			}...)
		}

		// The input name always comes after the applicable input arguments.
		args = append(args, []string{
			// The input itself.
			"-i", input.Name,
		}...)
	}

	for i, input := range t.inputs {
		mapArgs := []string{
			// Map corresponding input stream to output file.
			// The format is "<INPUT FILE NUMBER>:<STREAM SPECIFIER>", so "i" here
			// is the input file number, and "input.GetStreamSpecifier()" builds
			// the stream specifier for this input. The output stream for this
			// input is implied by where we are in the ffmpeg argument list.
			"-map", fmt.Sprintf("%d:%s", i, input.GetStreamSpecifier()),
		}

		for _, stream := range t.outputs {
			streamInput := stream.GetInput()

			if streamInput.TrackNum != input.TrackNum && streamInput.Name != input.Name {
				// Skip outputs that don't match this exact input object.
				continue
			}

			if stream.SkippedTranscoding() {
				// This input won't be transcoded. This is common for VTT text input.
				continue
			}

			// Map arguments must be repeated for each output file.
			args = append(args, mapArgs...)

			switch input.MediaType {
			case AUDIO:
				audioStream, _ := stream.(*AudioOutputStream)
				args = append(args, t.encodeAudio(audioStream, input)...)
			case VIDEO:
				videoStream, _ := stream.(*VideoOutputStream)
				args = append(args, t.encodeVideo(videoStream, input)...)
			case TEXT:
				textStream, _ := stream.(*TextOutputStream)
				args = append(args, t.encodeText(textStream, input)...)
			}

			ipcPipe := stream.GetIpcPipe()
			args = append(args, ipcPipe.WriteEnd())
		}
	}

	env := map[string]string{}

	if t.pipelineConfig.DebugLogs {
		// Use this environment variable to turn on ffmpeg's logging.  This is
		// independent of the -loglevel switch above.
		ffmpegLogFile := fmt.Sprintf("TranscoderNode-%d.log", t.index)
		env["FFREPORT"] = fmt.Sprintf("file=%s:level=32", ffmpegLogFile)
	}

	// start process
	t.Process = t.CreateProcess(BaseParams{args: args, env: env})
}

func (t TranscoderNode) encodeAudio(stream *AudioOutputStream, i Input) []string {
	var filters []string
	args := []string{
		// No video encoding for audio.
		"-vn",
		// TODO: This implied downmixing is not ideal.
		// Set the number of channels to the one specified in the config.
		"-ac", strconv.Itoa(stream.Layout.MaxChannels),
	}

	if stream.Layout.MaxChannels == 6 {
		filters = append(filters,
			// Work around for https://github.com/shaka-project/shaka-packager/issues/598,
			// as seen on https://trac.ffmpeg.org/ticket/6974
			"channelmap=channel_layout=5.1",
		)
	}

	filters = append(filters, i.Filters...)
	hwaccelAPI := t.pipelineConfig.HWAccelAPI

	args = append(args,
		// Set codec and bitrate.
		"-c:a", stream.GetFFmpegCodecString(hwaccelAPI),
		"-b:a", stream.GetBitrate(),
		// Output MP4 in the pipe, for all codecs.
		"-f", "mp4",
		// This explicit fragment duration affects both audio and video, and
		// ensures that there are no single large MP4 boxes that Shaka Packager
		// can't consume from a pipe.
		// FFmpeg fragment duration is in microseconds.
		"-frag_duration", strconv.Itoa(int(t.pipelineConfig.SegmentSize*1e6)),
		// Opus in MP4 is considered "experimental".
		"-strict", "experimental",
	)

	if len(filters) > 0 {
		args = append(args,
			// Set audio filters.
			"-af", strings.Join(filters, ","),
		)
	}

	return args
}

func (t TranscoderNode) encodeVideo(stream *VideoOutputStream, i Input) []string {
	var filters []string
	var args []string

	if i.IsInterlaced {
		filters = append(filters, "pp=fd")
		args = append(args, "-r", strconv.FormatFloat(i.FrameRate, 'f', -1, 64))
	}

	if stream.Resolution.MaxFrameRate < i.FrameRate {
		args = append(args, "-r", strconv.FormatFloat(stream.Resolution.MaxFrameRate, 'f', -1, 64))
	}

	filters = append(filters, i.Filters...)

	hwaccelAPI := t.pipelineConfig.HWAccelAPI

	// -2 in the scale filters means to choose a value to keep the original
	// aspect ratio.
	if stream.IsHardwareAccelerated() && hwaccelAPI == "vaapi" {
		// These filters are specific to Linux's vaapi.
		filters = append(filters, "format=nv12")
		filters = append(filters, "hwupload")
		filters = append(filters, fmt.Sprintf("scale_vaapi=-2:%d", stream.Resolution.MaxHeight))
	} else {
		filters = append(filters, fmt.Sprintf("scale=-2:%d", stream.Resolution.MaxHeight))
	}

	// To avoid weird rounding errors in Sample Aspect Ratio, set it explicitly
	// to 1:1. Without this, you wind up with SAR set to weird values in DASH
	// that are very close to 1, such as 5120:5123. In HLS, the behavior is
	// worse. Some of the width values in the playlist wind up off by one,
	// which causes playback failures in ExoPlayer.
	// https://github.com/shaka-project/shaka-streamer/issues/36
	filters = append(filters, "setsar=1:1")

	// These presets are specifically recognized by the software encoder.
	if (stream.Codec.Name == H264 || stream.Codec.Name == HEVC) && !stream.IsHardwareAccelerated() {
		if t.pipelineConfig.StreamingMode == LIVE {
			// Encodes with highest-speed presets for real-time live streaming.
			args = append(args, "-preset", "ultrafast")
		} else {
			// Take your time for VOD streams.
			args = append(args, "-preset", "slow")
			// Apply the loop filter for higher quality output.
			args = append(args, "-flags", "+loop")
		}
	}

	if stream.Codec.Name == H264 {
		// Use the "high" profile for HD and up, and "main" for everything else.
		// https://en.wikipedia.org/wiki/Advanced_Video_Coding#Profiles
		var profile string
		if stream.Resolution.MaxHeight >= 720 {
			profile = "high"
		} else {
			profile = "main"
		}

		// Set the H264 profile. Without this, the default would be "main".
		// Note that this gets overridden to "baseline" in live streams by the
		// "-preset ultrafast" option, presumably because the baseline encoder
		// is faster.
		args = append(args, "-profile:v", profile)
	}

	if stream.Codec.Name == H264 || stream.Codec.Name == HEVC {
		args = append(args,
			// The only format supported by QT/Apple.
			"-pix_fmt", "yuv420p",
			// Require a closed GOP.  Some decoders don't support open GOPs.
			"-flags", "+cgop",
		)
	} else if stream.Codec.Name == VP9 {
		// TODO: Does -preset apply here?
		args = append(args,
			// According to the wiki (https://trac.ffmpeg.org/wiki/Encode/VP9),
			// this allows threaded encoding in VP9, which makes better use of CPU
			// resources and speeds up encoding.  This is still not the default
			// setting as of libvpx v1.7.
			"-row-mt", "1",
			// speeds up encoding, balancing against quality
			"-speed", "2",
		)
	} else if stream.Codec.Name == AV1 {
		args = append(args,
			// According to graphs at https://bit.ly/2BmIVt6, this AV1 setting
			// results in almost no reduction in quality (0.8%), but a significant
			// boost in speed (20x).
			"-cpu-used", "8",
			// According to the wiki (https://trac.ffmpeg.org/wiki/Encode/AV1),
			// this allows threaded encoding in AV1, which makes better use of CPU
			// resources and speeds up encoding.  This will be ignored by libaom
			// before version 1.0.0-759-g90a15f4f2, and so there may be no benefit
			// unless libaom and ffmpeg are built from source (as of Oct 2019).
			"-row-mt", "1",
			// According to the wiki (https://trac.ffmpeg.org/wiki/Encode/AV1),
			// this allows for threaded _decoding_ in AV1, which will provide a
			// smoother playback experience for the end user.
			"-tiles", "2x2",
			// AV1 is considered "experimental".
			"-strict", "experimental",
		)
	}

	keyframeInterval := int(t.pipelineConfig.SegmentSize * i.FrameRate)

	args = append(args,
		// No audio encoding for video.
		"-an",
		// Set codec and bitrate.
		"-c:v", stream.GetFFmpegCodecString(hwaccelAPI),
		"-b:v", stream.GetBitrate(),
		// Output MP4 in the pipe, for all codecs.
		"-f", "mp4",
		// This flag forces a video fragment at each keyframe.
		"-movflags", "+frag_keyframe",
		// This explicit fragment duration affects both audio and video, and
		// ensures that there are no single large MP4 boxes that Shaka Packager
		// can't consume from a pipe.
		// FFmpeg fragment duration is in microseconds.
		"-frag_duration", strconv.FormatInt(int64(t.pipelineConfig.SegmentSize*1e6), 10),
		// Set minimum and maximum GOP length.
		"-keyint_min", strconv.Itoa(keyframeInterval), "-g", strconv.Itoa(keyframeInterval),
		// Set video filters.
		"-vf", strings.Join(filters, ","),
	)

	return args
}

func (t TranscoderNode) encodeText(stream *TextOutputStream, i Input) []string {
	return []string{
		// Output WebVTT.
		"-f", "webvtt",
	}
}
