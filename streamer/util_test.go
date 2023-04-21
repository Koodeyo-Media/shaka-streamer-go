package streamer

import (
	"path/filepath"

	"github.com/Koodeyo-Media/shaka-streamer-go/tests"
)

func getTestInput(idx int, mediaType MediaType) Input {
	rootDir, _ := RootDir()
	return *NewInput(FILE, filepath.Join(rootDir, tests.TestDir, tests.TestFiles[idx]), mediaType, []string{})
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

func getTestOutputStreams(inputs ...Input) TestOutputStreams {
	s := &TestOutputStreams{}
	rootDir, _ := RootDir()
	pipeDir := filepath.Join(rootDir, "tmp")

	for _, i := range inputs {
		switch i.MediaType {
		case VIDEO:
			vc := getTestVideoCodec()
			vr := *NewBitrateConfig().GetResolutionValue(i.Resolution)
			vos := NewVideoOutputStream(i, pipeDir, vc, vr)
			s.video = vos
		case AUDIO:
			ac := getTestAudioCodec()
			al := *NewBitrateConfig().GetChannelLayoutValue(i.ChannelLayout)
			aos := NewAudioOutputStream(i, pipeDir, ac, al)
			s.audio = aos
		case TEXT:
			s.text = NewTextOutputStream(i, pipeDir, true)
		}
	}

	return *s
}
