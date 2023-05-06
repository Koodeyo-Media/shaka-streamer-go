package streamer

import (
	"strconv"
	"strings"
)

type Codec interface {
	GetFFmpegCodecString(hwaccelAPI string) string
	IsHardwareAccelerated() bool
	GetOutputFormat() string
}

type MediaOutputStream interface {
	GetInput() Input
	GetCodec() Codec
	GetType() MediaType
	GetIpcPipe() Pipe
	IsDashOnly() bool
	GetInitSegFile() Pipe
	GetMediaSegFile() Pipe
	GetSingleSegFile() Pipe
	IsHardwareAccelerated() bool
	SkippedTranscoding() bool
}

// Base class for output streams.
type OutputStream struct {
	Type            MediaType
	SkipTranscoding bool
	Input           Input
	Features        map[string]string
	Codec           Codec
	ipcPipe         Pipe
}

func NewOutputStream(t MediaType, in Input, c Codec, pipeDir string, skipTranscoding bool, pipeSuffix string) *OutputStream {
	o := &OutputStream{
		Type:            t,
		Input:           in,
		SkipTranscoding: skipTranscoding,
		Features:        make(map[string]string),
		Codec:           c,
		ipcPipe:         NewPipe(),
	}

	if o.SkipTranscoding {
		// If skip_transcoding is specified, let the Packager read from a plain
		// file instead of an IPC pipe.
		o.ipcPipe.CreateFilePipe(in.Name, "r")
	} else {
		o.ipcPipe.CreateIpcPipe(pipeDir, pipeSuffix)
	}

	return o
}

func (o OutputStream) GetInput() Input {
	return o.Input
}

func (o OutputStream) GetCodec() Codec {
	return o.Codec
}

func (o OutputStream) GetType() MediaType {
	return o.Type
}

func (o OutputStream) GetIpcPipe() Pipe {
	return o.ipcPipe
}

func (o OutputStream) IsHardwareAccelerated() bool {
	return o.Codec != nil && o.Codec.IsHardwareAccelerated()
}

func (o OutputStream) SkippedTranscoding() bool {
	return o.SkipTranscoding
}

func (o OutputStream) GetFFmpegCodecString(hwaccelAPI string) string {
	if o.Codec != nil {
		return o.Codec.GetFFmpegCodecString(hwaccelAPI)
	}

	return ""
}

func (o OutputStream) IsDashOnly() bool {
	return o.Codec != nil && o.Codec.GetOutputFormat() == "webm"
}

func (o OutputStream) GetInitSegFile() Pipe {
	initSegment := map[MediaType]string{
		AUDIO: "audio_{language}_{channels}c_{bitrate}_{codec}_init.{format}",
		VIDEO: "video_{resolution_name}_{bitrate}_{codec}_init.{format}",
		TEXT:  "text_{language}_init.{format}",
	}

	pathTempl := initSegment[o.Type]

	for key, value := range o.Features {
		pathTempl = strings.ReplaceAll(pathTempl, "{"+key+"}", value)
	}

	pipe := NewPipe()
	pipe.CreateFilePipe(pathTempl, "w")

	return pipe
}

func (o OutputStream) GetMediaSegFile() Pipe {
	mediaSegment := map[MediaType]string{
		AUDIO: "audio_{language}_{channels}c_{bitrate}_{codec}_$Number$.{format}",
		VIDEO: "video_{resolution_name}_{bitrate}_{codec}_$Number$.{format}",
		TEXT:  "text_{language}_$Number$.{format}",
	}

	pathTempl := mediaSegment[o.Type]

	for key, value := range o.Features {
		pathTempl = strings.ReplaceAll(pathTempl, "{"+key+"}", value)
	}

	pipe := NewPipe()
	pipe.CreateFilePipe(pathTempl, "w")

	return pipe
}

func (o OutputStream) GetSingleSegFile() Pipe {
	singleSegment := map[MediaType]string{
		AUDIO: "audio_{language}_{channels}c_{bitrate}_{codec}.{format}",
		VIDEO: "video_{resolution_name}_{bitrate}_{codec}.{format}",
		TEXT:  "text_{language}.{format}",
	}

	pathTempl := singleSegment[o.Type]

	for key, value := range o.Features {
		pathTempl = strings.ReplaceAll(pathTempl, "{"+key+"}", value)
	}

	pipe := NewPipe()
	pipe.CreateFilePipe(pathTempl, "w")

	return pipe
}

type AudioOutputStream struct {
	*OutputStream
	Layout AudioChannelLayout
	Codec  *AudioCodec
}

func NewAudioOutputStream(i Input, pipeDir string, c *AudioCodec, l AudioChannelLayout) *AudioOutputStream {
	// The features that will be used to generate the output filename.
	features := make(map[string]string)
	features["language"] = i.Language
	features["channels"] = strconv.Itoa(l.MaxChannels)
	features["bitrate"] = string(l.Bitrates[c.Name])
	features["format"] = c.GetOutputFormat()
	features["codec"] = string(c.Name)

	s := NewOutputStream(AUDIO, i, c, pipeDir, false, "")
	s.Features = features

	return &AudioOutputStream{
		OutputStream: s,
		Layout:       l,
		Codec:        c,
	}
}

// Returns the bitrate for this stream.
func (a AudioOutputStream) GetBitrate() string {
	return string(a.Layout.Bitrates[a.Codec.Name])
}

type VideoOutputStream struct {
	*OutputStream
	Resolution VideoResolution
	Codec      *VideoCodec
}

func NewVideoOutputStream(i Input, pipeDir string, c *VideoCodec, r VideoResolution) *VideoOutputStream {
	// The features that will be used to generate the output filename.
	features := make(map[string]string)
	features["resolution_name"] = string(r.Name)
	features["bitrate"] = string(r.Bitrates[c.Name])
	features["format"] = c.GetOutputFormat()
	features["codec"] = string(c.Name)

	s := NewOutputStream(VIDEO, i, c, pipeDir, false, "")
	s.Features = features

	return &VideoOutputStream{
		OutputStream: s,
		Resolution:   r,
		Codec:        c,
	}
}

// Returns the bitrate for this stream.
func (v VideoOutputStream) GetBitrate() string {
	return string(v.Resolution.Bitrates[v.Codec.Name])
}

type TextOutputStream struct {
	*OutputStream
}

func NewTextOutputStream(i Input, pipeDir string, skipTranscoding bool) *TextOutputStream {
	s := NewOutputStream(TEXT, i, nil, pipeDir, skipTranscoding, ".vtt")

	s.Features = map[string]string{
		"language": i.Language,
		"format":   "mp4",
	}

	tos := &TextOutputStream{
		OutputStream: s,
	}

	return tos
}
