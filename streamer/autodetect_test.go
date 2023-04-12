package streamer

import (
	"testing"
)

func Test_probe(t *testing.T) {
	type args struct {
		i     *Input
		field string
	}

	input := getTestInput(0, VIDEO)

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
