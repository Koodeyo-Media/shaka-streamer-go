package streamer

import (
	"os"
	"testing"
)

func TestControllerNode_Start(t *testing.T) {
	type fields struct {
		tempDir string
		nodes   []interface{}
	}

	tests := []struct {
		name   string
		fields fields
		args   ControllerParams
		want   interface{}
	}{
		{
			name: "start",
			args: ControllerParams{
				inputConfigDict:    InputConfig{},
				pipelineConfigDict: PipelineConfig{},
				bitrateConfigDict:  BitrateConfig{},
				bucketURL:          "",
				outputLocation:     "",
				useHermetic:        false,
				checkDeps:          true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ControllerNode{
				tempDir: tt.fields.tempDir,
				nodes:   tt.fields.nodes,
			}

			c.Start(tt.args)

			os.RemoveAll(c.tempDir)
		})
	}
}
