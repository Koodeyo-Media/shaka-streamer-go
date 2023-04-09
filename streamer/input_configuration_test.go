package streamer

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Koodeyo-Media/shaka-streamer-go/tests"
)

func TestNewInput(t *testing.T) {
	type args struct {
		inputType InputType
		name      string
		mediaType MediaType
		filters   []string
	}

	getArgs := func(idx int, mediaType MediaType) args {
		return args{
			inputType: FILE,
			name:      filepath.Join(".", "..", tests.TestDir, tests.TestFiles[idx]),
			mediaType: mediaType,
			filters:   []string{},
		}
	}

	// Resolutions
	resolutionTests := []struct {
		name string
		args args
		want VideoResolutionName
	}{
		{
			name: "Resolution - 1080p",
			args: getArgs(0, VIDEO),
			want: VideoResolutionName("1080p"),
		},
		{
			name: "Resolution - 720p",
			args: getArgs(1, VIDEO),
			want: VideoResolutionName("720p"),
		},
	}

	for _, tt := range resolutionTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInput(tt.args.inputType, tt.args.name, tt.args.mediaType, tt.args.filters); !reflect.DeepEqual(got.Resolution, tt.want) {
				t.Errorf("Resolution = %v, want %v", got.Resolution, tt.want)
			}
		})
	}

	// FrameRate
	frameRateTests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "FrameRate - 30",
			args: getArgs(0, VIDEO),
			want: 30,
		},
		{
			name: "FrameRate - 24",
			args: getArgs(1, VIDEO),
			want: 24,
		},
	}

	for _, tt := range frameRateTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInput(tt.args.inputType, tt.args.name, tt.args.mediaType, tt.args.filters); !reflect.DeepEqual(got.FrameRate, tt.want) {
				t.Errorf("FrameRate = %v, want %v", got.FrameRate, tt.want)
			}
		})
	}
}
