package streamer

import (
	"fmt"
	"time"
)

type PeriodConcatNode struct {
	pipelineConfig PipelineConfig
	outputLocation string
	packagerNodes  []PackagerNode
	concatWillFail bool
	status         ProcessStatus
}

func NewPeriodConcatNode(pipelineConfig PipelineConfig, packagerNodes []PackagerNode, outputLocation string) *PeriodConcatNode {
	fpHasVid, fpHasAud := false, false

	for _, outputStream := range packagerNodes[0].OutputStreams {
		switch outputStream.(type) {
		case VideoOutputStream:
			fpHasVid = true
		case AudioOutputStream:
			fpHasAud = true
		}
	}

	concatWillFail := false
	for i, packagerNode := range packagerNodes {
		hasVid, hasAud := false, false
		for _, outputStream := range packagerNode.OutputStreams {
			switch outputStream.(type) {
			case VideoOutputStream:
				hasVid = true
			case AudioOutputStream:
				hasAud = true
			}
		}

		if hasVid != fpHasVid || hasAud != fpHasAud {
			fmt.Printf("\nWARNING: Stopping period concatenation.\n")
			fmt.Printf("Period#%d has %svideo and has %saudio while Period#1 has %svideo and has %saudio.\n",
				i+1,
				func() string {
					if hasVid {
						return ""
					} else {
						return "no "
					}
				}(),
				func() string {
					if hasAud {
						return ""
					} else {
						return "no "
					}
				}(),
				func() string {
					if fpHasVid {
						return ""
					} else {
						return "no "
					}
				}(),
				func() string {
					if fpHasAud {
						return ""
					} else {
						return "no "
					}
				}(),
			)

			fmt.Printf("\nHINT:\n\tBe sure that either all the periods have video or all do not,\n" +
				"\tand all the periods have audio or all do not, i.e. don't mix videoless\n" +
				"\tperiods with other periods that have video.\n" +
				"\tThis is necessary for the concatenation to be performed successfully.\n")
			time.Sleep(5 * time.Second)
			concatWillFail = true
			break
		}
	}

	return &PeriodConcatNode{
		pipelineConfig: pipelineConfig,
		packagerNodes:  packagerNodes,
		outputLocation: outputLocation,
		concatWillFail: concatWillFail,
	}
}

func (n *PeriodConcatNode) ThreadSinglePass() {
	for i, packagerNode := range n.packagerNodes {
		status := packagerNode.CheckStatus()

		if status == Running {
			return
		} else if status == Errored {
			panic(fmt.Sprintf("Concatenation is stopped due to an error in PackagerNode#%d.", i+1))
		}
	}

	if n.concatWillFail {
		panic("Unable to concatenate the inputs.")
	}

	if ContainsString(ManifestFormatListToStringList(n.pipelineConfig.ManifestFormat), string(DASH)) {
		n.dashConcat()
	}

	if ContainsString(ManifestFormatListToStringList(n.pipelineConfig.ManifestFormat), string(HLS)) {
		n.hlsConcat()
	}

	n.status = Finished
}

func (n *PeriodConcatNode) dashConcat() {}
func (n *PeriodConcatNode) hlsConcat()  {}
