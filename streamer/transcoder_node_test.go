package streamer

import (
	"os"
	"testing"
)

func TestNewTranscoderNode(t *testing.T) {
	vi := getTestInput(0, VIDEO)
	vi.Resolution = VideoResolutionName("144p")
	vis := getTestOutputStreams(vi)
	var videoStream MediaOutputStream = vis.video
	// vi2 := vi
	// vi2.Resolution = VideoResolutionName("240p")
	// fmt.Println(vi2.Resolution)

	pipelineConfig := PipelineConfig{}
	pipelineConfig.StreamingMode = VOD

	var output []MediaOutputStream

	type args struct {
		inputs         []Input
		pipelineConfig PipelineConfig
		outputs        []MediaOutputStream
		index          int
		hermeticFFmpeg string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "144p",
			args: args{
				inputs:         []Input{vi},
				pipelineConfig: pipelineConfig,
				outputs:        append(output, videoStream),
				hermeticFFmpeg: "",
				index:          0,
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := NewTranscoderNode(tt.args.inputs, tt.args.pipelineConfig, tt.args.outputs, tt.args.index, tt.args.hermeticFFmpeg)
			node.Start()
			os.Remove(vis.video.ipcPipe.WriteEnd())
		})
	}
}
