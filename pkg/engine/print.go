package engine

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

func (e *Engine) runPrint(tool types.Tool) (cmdOut *Return, cmdErr error) {
	id := fmt.Sprint(atomic.AddInt64(&completionID, 1))
	out := strings.TrimPrefix(tool.Instructions, types.PrintPrefix+"\n")

	e.Progress <- types.CompletionStatus{
		CompletionID: id,
		Response: map[string]any{
			"output": out,
			"err":    nil,
		},
	}

	return &Return{
		Result: &out,
	}, nil
}
