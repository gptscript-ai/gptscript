package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/gptscript-ai/gptscript/pkg/engine"
)

func main() {
	data := struct {
		Call engine.CallContext `json:"call,omitempty"`
	}{}
	if err := json.Unmarshal([]byte(os.Getenv("GPTSCRIPT_CONTEXT")), &data); err != nil {
		panic(err)
	}

	for _, agent := range data.Call.AgentGroup {
		fmt.Println(agent.Reference, agent.ToolID)
	}
}
