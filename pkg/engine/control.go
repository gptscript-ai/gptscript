package engine

import (
	"encoding/json"
	"fmt"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

func (e *Engine) runBreak(tool types.Tool, input string) (cmdOut *Return, cmdErr error) {
	info, err := json.Marshal(tool)
	if err != nil {
		return nil, err
	}
	var dict map[string]interface{}
	json.Unmarshal(info, &dict)
	dict["input"] = input
	info, err = json.Marshal(dict)
	return nil, fmt.Errorf("TOOL_BREAK: %s", info)
}
