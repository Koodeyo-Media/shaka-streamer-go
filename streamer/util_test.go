package streamer

import (
	"os"
	"path/filepath"

	"github.com/Koodeyo-Media/shaka-streamer-go/tests"
)

func getTestInput(idx int, mediaType MediaType) *Input {
	return NewInput(FILE, filepath.Join(".", "..", tests.TestDir, tests.TestFiles[idx]), mediaType, []string{})
}

func getTestVideoCodec() *VideoCodec {
	return &VideoCodec{Name: H264, HWAcc: false}
}

func getTestAudioCodec() *AudioCodec {
	return &AudioCodec{Name: AAC}
}

type TestOutputStreams struct {
	video *VideoOutputStream
	audio *AudioOutputStream
	text  *TextOutputStream
}

func getTestOutputStreams(inputs ...*Input) *TestOutputStreams {
	s := &TestOutputStreams{}

	for _, i := range inputs {
		switch i.MediaType {
		case VIDEO:
			vc := getTestVideoCodec()
			vr := NewBitrateConfig().GetResolutionValue(i.Resolution)
			vos := NewVideoOutputStream(i, os.TempDir(), vc, vr)
			s.video = vos
		case AUDIO:
			ac := getTestAudioCodec()
			al := NewBitrateConfig().GetChannelLayoutValue(i.ChannelLayout)
			aos := NewAudioOutputStream(i, os.TempDir(), ac, al)
			s.audio = aos
		case TEXT:
			s.text = NewTextOutputStream(i, os.TempDir(), true)
		}
	}

	return s
}
