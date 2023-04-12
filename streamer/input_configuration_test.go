package streamer

import (
	"reflect"
	"testing"
)

func TestNewInput(t *testing.T) {
	type args struct {
		index     int
		mediaType MediaType
	}

	getArgs := func(idx int, mediaType MediaType) args {
		return args{
			index:     idx,
			mediaType: mediaType,
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
			if got := getTestInput(tt.args.index, tt.args.mediaType); !reflect.DeepEqual(got.Resolution, tt.want) {
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
			if got := getTestInput(tt.args.index, tt.args.mediaType); !reflect.DeepEqual(got.FrameRate, tt.want) {
				t.Errorf("FrameRate = %v, want %v", got.FrameRate, tt.want)
			}
		})
	}
}
