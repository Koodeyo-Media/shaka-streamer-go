package streamer

import (
	"path/filepath"
	"testing"

	"github.com/Koodeyo-Media/shaka-streamer-go/tests"
)

func Test_probe(t *testing.T) {
	type args struct {
		i     *Input
		field string
	}

	name := filepath.Join(".", "..", tests.TestDir, tests.TestFiles[0])

	input := NewInput(FILE, name, VIDEO, []string{})

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Probe - Track",
			args: args{
				i:     input,
				field: "stream=index",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := probe(tt.args.i, tt.args.field)

			if (len(got) > 0) != tt.want {
				t.Errorf("probe() = %v, want %v", false, tt.want)
			}
		})
	}
}
