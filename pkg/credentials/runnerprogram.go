package credentials

import (
	"io"
)

type runnerProgram struct {
	factory *StoreFactory
	action  string
	output  string
	err     error
}

func (r *runnerProgram) Output() ([]byte, error) {
	return []byte(r.output), r.err
}

func (r *runnerProgram) Input(in io.Reader) {
	input, err := io.ReadAll(in)
	if err != nil {
		r.err = err
		return
	}

	prg := r.factory.prg
	prg.EntryToolID = prg.ToolSet[prg.EntryToolID].LocalTools[r.action]

	r.output, r.err = r.factory.runner.Run(r.factory.ctx, prg, string(input))
}
