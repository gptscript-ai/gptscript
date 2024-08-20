//go:build noyaegi

package engine

import (
	"fmt"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

func (e *Engine) runYaegi(_ Context, tool types.Tool, _ string, _ ToolCategory) (*Return, error) {
	return nil, fmt.Errorf("Interpreter %s at %s disabled", types.YaegiPrefix, tool.ID)
}
