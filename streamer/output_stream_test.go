package streamer

import (
	"os"
	"reflect"
	"testing"
)

type fields struct {
	input        *Input
	outputStream interface{}
}

func TestOutputStream_GetMediaSegFile(t *testing.T) {
	vi := getTestInput(0, VIDEO)
	ai := getTestInput(1, AUDIO)
	ti := getTestInput(2, TEXT)

	streams := getTestOutputStreams(vi, ai, ti)

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "GetInitSegFile (video)",
			fields: fields{
				input:        vi,
				outputStream: streams.video,
			},
			want: "video_1080p_4M_h264_init.mp4",
		},
		{
			name: "GetInitSegFile - (audio)",
			fields: fields{
				input:        ai,
				outputStream: streams.audio,
			},
			want: "audio_eng_6c_256k_aac_init.mp4",
		},
		{
			name: "GetInitSegFile - (text)",
			fields: fields{
				input:        ti,
				outputStream: streams.text,
			},
			want: "text_und_init.mp4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch s := tt.fields.outputStream.(type) {
			case *VideoOutputStream:
				if got := s.GetInitSegFile(); !reflect.DeepEqual(got.WriteEnd(), tt.want) {
					t.Errorf("OutputStream.GetInitSegFile(video) = %v, want %v", got.WriteEnd(), tt.want)
				}

				os.Remove(s.ipcPipe.writePipeName)
			case *AudioOutputStream:
				if got := s.GetInitSegFile(); !reflect.DeepEqual(got.WriteEnd(), tt.want) {
					t.Errorf("OutputStream.GetInitSegFile(audio) = %v, want %v", got.WriteEnd(), tt.want)
				}

				os.Remove(s.ipcPipe.writePipeName)

			case *TextOutputStream:
				if got := s.GetInitSegFile(); !reflect.DeepEqual(got.WriteEnd(), tt.want) {
					t.Errorf("OutputStream.GetInitSegFile(text) = %v, want %v", got.WriteEnd(), tt.want)
				}
			default:
				panic("Unknown type")
			}
		})
	}
}
