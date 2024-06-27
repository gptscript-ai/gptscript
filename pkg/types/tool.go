package types

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/shlex"
	"github.com/gptscript-ai/gptscript/pkg/system"
	"golang.org/x/exp/maps"
)

const (
	DaemonPrefix  = "#!sys.daemon"
	OpenAPIPrefix = "#!sys.openapi"
	EchoPrefix    = "#!sys.echo"
	CommandPrefix = "#!"
)

var (
	DefaultFiles = []string{"agent.gpt", "tool.gpt"}
)

type ErrToolNotFound struct {
	ToolName string
}

func ToToolName(toolName, subTool string) string {
	if subTool == "" {
		return toolName
	}
	return fmt.Sprintf("%s from %s", subTool, toolName)
}

func NewErrToolNotFound(toolName string) *ErrToolNotFound {
	return &ErrToolNotFound{
		ToolName: toolName,
	}
}

func (e *ErrToolNotFound) Error() string {
	return fmt.Sprintf("tool not found: %s", e.ToolName)
}

type ToolSet map[string]Tool

type Program struct {
	Name         string         `json:"name,omitempty"`
	EntryToolID  string         `json:"entryToolId,omitempty"`
	ToolSet      ToolSet        `json:"toolSet,omitempty"`
	OpenAPICache map[string]any `json:"-"`
}

func (p Program) IsChat() bool {
	return p.ToolSet[p.EntryToolID].Chat
}

func (p Program) ChatName() string {
	if p.IsChat() {
		name := p.ToolSet[p.EntryToolID].Name
		if name != "" {
			return name
		}
	}
	return p.Name
}

type ToolReference struct {
	Named     string `json:"named,omitempty"`
	Reference string `json:"reference,omitempty"`
	Arg       string `json:"arg,omitempty"`
	ToolID    string `json:"toolID,omitempty"`
}

func (p Program) GetContextToolRefs(toolID string) ([]ToolReference, error) {
	return p.ToolSet[toolID].GetContextTools(p)
}

func (p Program) GetCompletionTools() (result []CompletionTool, err error) {
	return Tool{
		ToolDef: ToolDef{
			Parameters: Parameters{
				Tools: []string{"main"},
			},
		},
		ToolMapping: map[string][]ToolReference{
			"main": {
				{
					Reference: "main",
					ToolID:    p.EntryToolID,
				},
			},
		},
	}.GetCompletionTools(p)
}

func (p Program) TopLevelTools() (result []Tool) {
	for _, tool := range p.ToolSet[p.EntryToolID].LocalTools {
		if target, ok := p.ToolSet[tool]; ok {
			result = append(result, target)
		}
	}
	return
}

func (p Program) SetBlocking() Program {
	tool := p.ToolSet[p.EntryToolID]
	tool.Blocking = true
	tools := maps.Clone(p.ToolSet)
	tools[p.EntryToolID] = tool
	p.ToolSet = tools
	return p
}

type BuiltinFunc func(ctx context.Context, env []string, input string, progress chan<- string) (string, error)

type Parameters struct {
	Name                string           `json:"name,omitempty"`
	Description         string           `json:"description,omitempty"`
	MaxTokens           int              `json:"maxTokens,omitempty"`
	ModelName           string           `json:"modelName,omitempty"`
	ModelProvider       bool             `json:"modelProvider,omitempty"`
	JSONResponse        bool             `json:"jsonResponse,omitempty"`
	Chat                bool             `json:"chat,omitempty"`
	Temperature         *float32         `json:"temperature,omitempty"`
	Cache               *bool            `json:"cache,omitempty"`
	InternalPrompt      *bool            `json:"internalPrompt"`
	Arguments           *openapi3.Schema `json:"arguments,omitempty"`
	Tools               []string         `json:"tools,omitempty"`
	GlobalTools         []string         `json:"globalTools,omitempty"`
	GlobalModelName     string           `json:"globalModelName,omitempty"`
	Context             []string         `json:"context,omitempty"`
	ExportContext       []string         `json:"exportContext,omitempty"`
	Export              []string         `json:"export,omitempty"`
	Agents              []string         `json:"agents,omitempty"`
	Credentials         []string         `json:"credentials,omitempty"`
	InputFilters        []string         `json:"inputFilters,omitempty"`
	ExportInputFilters  []string         `json:"exportInputFilters,omitempty"`
	OutputFilters       []string         `json:"outputFilters,omitempty"`
	ExportOutputFilters []string         `json:"exportOutputFilters,omitempty"`
	Blocking            bool             `json:"-"`
}

func (p Parameters) ToolRefNames() []string {
	return slices.Concat(
		p.Tools,
		p.Agents,
		p.Export,
		p.ExportContext,
		p.Context,
		p.Credentials,
		p.InputFilters,
		p.ExportInputFilters,
		p.OutputFilters,
		p.ExportOutputFilters)
}

type ToolDef struct {
	Parameters   `json:",inline"`
	Instructions string      `json:"instructions,omitempty"`
	BuiltinFunc  BuiltinFunc `json:"-"`
}

type Tool struct {
	ToolDef `json:",inline"`

	ID          string                     `json:"id,omitempty"`
	ToolMapping map[string][]ToolReference `json:"toolMapping,omitempty"`
	LocalTools  map[string]string          `json:"localTools,omitempty"`
	Source      ToolSource                 `json:"source,omitempty"`
	WorkingDir  string                     `json:"workingDir,omitempty"`
}

func IsMatch(subTool string) bool {
	return strings.ContainsAny(subTool, "*?[")
}

func (t *Tool) AddToolMapping(name string, tool Tool) {
	if t.ToolMapping == nil {
		t.ToolMapping = map[string][]ToolReference{}
	}

	ref := name
	_, subTool := SplitToolRef(name)
	if IsMatch(subTool) && tool.Name != "" {
		ref = strings.Replace(ref, subTool, tool.Name, 1)
	}

	if existing, ok := t.ToolMapping[name]; ok {
		var found bool
		for _, toolRef := range existing {
			if toolRef.ToolID == tool.ID && toolRef.Reference == ref {
				found = true
				break
			}
		}
		if found {
			return
		}
	}

	t.ToolMapping[name] = append(t.ToolMapping[name], ToolReference{
		Reference: ref,
		ToolID:    tool.ID,
	})
}

// SplitArg splits a tool string into the tool name and arguments, and discards the alias if there is one.
// Examples:
// toolName => toolName, ""
// toolName as myAlias => toolName, ""
// toolName with value1 as arg1 and value2 as arg2 => toolName, "value1 as arg1 and value2 as arg2"
// toolName as myAlias with value1 as arg1 and value2 as arg2 => toolName, "value1 as arg1 and value2 as arg2"
func SplitArg(hasArg string) (prefix, arg string) {
	var (
		fields  = strings.Fields(hasArg)
		withIdx = slices.Index(fields, "with")
		asIdx   = slices.Index(fields, "as")
	)

	if withIdx == -1 {
		if asIdx != -1 {
			return strings.Join(fields[:asIdx], " "),
				strings.Join(fields[asIdx:], " ")
		}
		return strings.TrimSpace(hasArg), ""
	}

	if asIdx != -1 && asIdx < withIdx {
		return strings.Join(fields[:asIdx], " "),
			strings.Join(fields[withIdx+1:], " ")
	}

	return strings.Join(fields[:withIdx], " "),
		strings.Join(fields[withIdx+1:], " ")
}

// ParseCredentialArgs parses a credential tool name + args into a tool alias (if there is one) and a map of args.
// Example: "toolName as myCredential with value1 as arg1 and value2 as arg2" -> toolName, myCredential, map[string]any{"arg1": "value1", "arg2": "value2"}, nil
//
// Arg references will be resolved based on the input.
// Example:
// - toolName: "toolName with ${var1} as arg1 and ${var2} as arg2"
// - input: `{"var1": "value1", "var2": "value2"}`
// result: toolName, "", map[string]any{"arg1": "value1", "arg2": "value2"}, nil
func ParseCredentialArgs(toolName string, input string) (string, string, map[string]any, error) {
	if toolName == "" {
		return "", "", nil, nil
	}

	inputMap := make(map[string]any)
	if input != "" {
		// Sometimes this function can be called with input that is not a JSON string.
		// This typically happens during chat mode.
		// That's why we ignore the error if this fails to unmarshal.
		_ = json.Unmarshal([]byte(input), &inputMap)
	}

	fields, err := shlex.Split(toolName)
	if err != nil {
		return "", "", nil, err
	}

	// If it's just the tool name, return it
	if len(fields) == 1 {
		return toolName, "", nil, nil
	}

	// Next field is "as" if there is an alias, otherwise it should be "with"
	originalName := fields[0]
	alias := ""
	fields = fields[1:]
	if fields[0] == "as" {
		if len(fields) < 2 {
			return "", "", nil, fmt.Errorf("expected alias after 'as'")
		}
		alias = fields[1]
		fields = fields[2:]
	}

	if len(fields) == 0 { // Nothing left, so just return
		return originalName, alias, nil, nil
	}

	// Next we should have "with" followed by the args
	if fields[0] != "with" {
		return "", "", nil, fmt.Errorf("expected 'with' but got %s", fields[0])
	}
	fields = fields[1:]

	// If there are no args, return an error
	if len(fields) == 0 {
		return "", "", nil, fmt.Errorf("expected args after 'with'")
	}

	args := make(map[string]any)
	prev := "none" // "none", "value", "as", "name", or "and"
	argValue := ""
	for _, field := range fields {
		switch prev {
		case "none", "and":
			argValue = field
			prev = "value"
		case "value":
			if field != "as" {
				return "", "", nil, fmt.Errorf("expected 'as' but got %s", field)
			}
			prev = "as"
		case "as":
			args[field] = argValue
			prev = "name"
		case "name":
			if field != "and" {
				return "", "", nil, fmt.Errorf("expected 'and' but got %s", field)
			}
			prev = "and"
		}
	}

	if prev == "and" {
		return "", "", nil, fmt.Errorf("expected arg name after 'and'")
	}

	// Check and see if any of the arg values are references to an input
	for k, v := range args {
		if strings.HasPrefix(v.(string), "${") && strings.HasSuffix(v.(string), "}") {
			key := strings.TrimSuffix(strings.TrimPrefix(v.(string), "${"), "}")
			if val, ok := inputMap[key]; ok {
				args[k] = val.(string)
			}
		}
	}

	return originalName, alias, args, nil
}

func (t Tool) GetAgents(prg Program) (result []ToolReference, _ error) {
	toolRefs, err := t.GetToolRefsFromNames(t.Agents)
	if err != nil {
		return nil, err
	}

	// Agent Tool refs must be named
	for i, toolRef := range toolRefs {
		if toolRef.Named != "" {
			continue
		}
		tool := prg.ToolSet[toolRef.ToolID]
		name := tool.Name
		if name == "" {
			name = toolRef.Reference
		}
		normed := ToolNormalizer(name)
		normed = strings.TrimSuffix(strings.TrimSuffix(normed, "Agent"), "Assistant")
		toolRefs[i].Named = normed
	}

	return toolRefs, nil
}

func (t Tool) GetToolRefsFromNames(names []string) (result []ToolReference, _ error) {
	for _, toolName := range names {
		toolRefs, ok := t.ToolMapping[toolName]
		if !ok || len(toolRefs) == 0 {
			return nil, NewErrToolNotFound(toolName)
		}
		_, arg := SplitArg(toolName)
		named, ok := strings.CutPrefix(arg, "as ")
		if !ok {
			named = ""
		} else if len(toolRefs) > 1 {
			return nil, fmt.Errorf("can not combine 'as' syntax with wildcard: %s", toolName)
		}
		for _, toolRef := range toolRefs {
			result = append(result, ToolReference{
				Named:     named,
				Arg:       arg,
				Reference: toolRef.Reference,
				ToolID:    toolRef.ToolID,
			})
		}
	}
	return
}

func (t ToolDef) String() string {
	buf := &strings.Builder{}
	if t.Parameters.GlobalModelName != "" {
		_, _ = fmt.Fprintf(buf, "Global Model Name: %s\n", t.Parameters.GlobalModelName)
	}
	if len(t.Parameters.GlobalTools) != 0 {
		_, _ = fmt.Fprintf(buf, "Global Tools: %s\n", strings.Join(t.Parameters.GlobalTools, ", "))
	}
	if t.Parameters.Name != "" {
		_, _ = fmt.Fprintf(buf, "Name: %s\n", t.Parameters.Name)
	}
	if t.Parameters.Description != "" {
		_, _ = fmt.Fprintf(buf, "Description: %s\n", t.Parameters.Description)
	}
	if len(t.Parameters.Agents) != 0 {
		_, _ = fmt.Fprintf(buf, "Agents: %s\n", strings.Join(t.Parameters.Agents, ", "))
	}
	if len(t.Parameters.Tools) != 0 {
		_, _ = fmt.Fprintf(buf, "Tools: %s\n", strings.Join(t.Parameters.Tools, ", "))
	}
	if len(t.Parameters.Export) != 0 {
		_, _ = fmt.Fprintf(buf, "Share Tools: %s\n", strings.Join(t.Parameters.Export, ", "))
	}
	if len(t.Parameters.Context) != 0 {
		_, _ = fmt.Fprintf(buf, "Context: %s\n", strings.Join(t.Parameters.Context, ", "))
	}
	if len(t.Parameters.ExportContext) != 0 {
		_, _ = fmt.Fprintf(buf, "Share Context: %s\n", strings.Join(t.Parameters.ExportContext, ", "))
	}
	if len(t.Parameters.InputFilters) != 0 {
		_, _ = fmt.Fprintf(buf, "Input Filters: %s\n", strings.Join(t.Parameters.InputFilters, ", "))
	}
	if len(t.Parameters.ExportInputFilters) != 0 {
		_, _ = fmt.Fprintf(buf, "Share Input Filters: %s\n", strings.Join(t.Parameters.ExportInputFilters, ", "))
	}
	if len(t.Parameters.OutputFilters) != 0 {
		_, _ = fmt.Fprintf(buf, "Output Filters: %s\n", strings.Join(t.Parameters.OutputFilters, ", "))
	}
	if len(t.Parameters.ExportOutputFilters) != 0 {
		_, _ = fmt.Fprintf(buf, "Share Output Filters: %s\n", strings.Join(t.Parameters.ExportOutputFilters, ", "))
	}
	if t.Parameters.MaxTokens != 0 {
		_, _ = fmt.Fprintf(buf, "Max Tokens: %d\n", t.Parameters.MaxTokens)
	}
	if t.Parameters.ModelName != "" {
		_, _ = fmt.Fprintf(buf, "Model: %s\n", t.Parameters.ModelName)
	}
	if t.Parameters.ModelProvider {
		_, _ = fmt.Fprintf(buf, "Model Provider: true\n")
	}
	if t.Parameters.JSONResponse {
		_, _ = fmt.Fprintln(buf, "JSON Response: true")
	}
	if t.Parameters.Cache != nil && !*t.Parameters.Cache {
		_, _ = fmt.Fprintln(buf, "Cache: false")
	}
	if t.Parameters.Temperature != nil {
		_, _ = fmt.Fprintf(buf, "Temperature: %f\n", *t.Parameters.Temperature)
	}
	if t.Parameters.Arguments != nil {
		var keys []string
		for k := range t.Parameters.Arguments.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			prop := t.Parameters.Arguments.Properties[key]
			_, _ = fmt.Fprintf(buf, "Parameter: %s: %s\n", key, prop.Value.Description)
		}
	}
	if t.Parameters.InternalPrompt != nil {
		_, _ = fmt.Fprintf(buf, "Internal Prompt: %v\n", *t.Parameters.InternalPrompt)
	}
	if len(t.Parameters.Credentials) > 0 {
		for _, cred := range t.Parameters.Credentials {
			_, _ = fmt.Fprintf(buf, "Credential: %s\n", cred)
		}
	}
	if t.Parameters.Chat {
		_, _ = fmt.Fprintf(buf, "Chat: true\n")
	}

	// Instructions should be printed last
	if t.Instructions != "" && t.BuiltinFunc == nil {
		_, _ = fmt.Fprintln(buf)
		_, _ = fmt.Fprintln(buf, t.Instructions)
	}

	return buf.String()
}

func (t Tool) GetExportedContext(prg Program) ([]ToolReference, error) {
	result := &toolRefSet{}

	exportRefs, err := t.GetToolRefsFromNames(t.ExportContext)
	if err != nil {
		return nil, err
	}

	for _, exportRef := range exportRefs {
		result.Add(exportRef)

		tool := prg.ToolSet[exportRef.ToolID]
		result.AddAll(tool.GetExportedContext(prg))
	}

	return result.List()
}

func (t Tool) GetExportedTools(prg Program) ([]ToolReference, error) {
	result := &toolRefSet{}

	exportRefs, err := t.GetToolRefsFromNames(t.Export)
	if err != nil {
		return nil, err
	}

	for _, exportRef := range exportRefs {
		result.Add(exportRef)
		result.AddAll(prg.ToolSet[exportRef.ToolID].GetExportedTools(prg))
	}

	return result.List()
}

// GetContextTools returns all tools that are in the context of the tool including all the
// contexts that are exported by the context tools. This will recurse all exports.
func (t Tool) GetContextTools(prg Program) ([]ToolReference, error) {
	result := &toolRefSet{}

	contextRefs, err := t.GetToolRefsFromNames(t.Context)
	if err != nil {
		return nil, err
	}

	for _, contextRef := range contextRefs {
		result.AddAll(prg.ToolSet[contextRef.ToolID].GetExportedContext(prg))
		result.Add(contextRef)
	}

	return result.List()
}

func (t Tool) GetOutputFilterTools(program Program) ([]ToolReference, error) {
	result := &toolRefSet{}

	outputFilterRefs, err := t.GetToolRefsFromNames(t.OutputFilters)
	if err != nil {
		return nil, err
	}

	for _, outputFilterRef := range outputFilterRefs {
		result.Add(outputFilterRef)
	}

	contextRefs, err := t.GetContextTools(program)
	if err != nil {
		return nil, err
	}

	for _, contextRef := range contextRefs {
		contextTool := program.ToolSet[contextRef.ToolID]
		result.AddAll(contextTool.GetToolRefsFromNames(contextTool.ExportOutputFilters))
	}

	return result.List()
}

func (t Tool) GetInputFilterTools(program Program) ([]ToolReference, error) {
	result := &toolRefSet{}

	inputFilterRefs, err := t.GetToolRefsFromNames(t.InputFilters)
	if err != nil {
		return nil, err
	}

	for _, inputFilterRef := range inputFilterRefs {
		result.Add(inputFilterRef)
	}

	contextRefs, err := t.GetContextTools(program)
	if err != nil {
		return nil, err
	}

	for _, contextRef := range contextRefs {
		contextTool := program.ToolSet[contextRef.ToolID]
		result.AddAll(contextTool.GetToolRefsFromNames(contextTool.ExportInputFilters))
	}

	return result.List()
}

func (t Tool) GetNextAgentGroup(prg Program, agentGroup []ToolReference, toolID string) (result []ToolReference, _ error) {
	newAgentGroup := toolRefSet{}
	if err := t.addAgents(prg, &newAgentGroup); err != nil {
		return nil, err
	}

	if newAgentGroup.HasTool(toolID) {
		// Join new agent group
		return newAgentGroup.List()
	}

	return agentGroup, nil
}

func (t Tool) GetCompletionTools(prg Program, agentGroup ...ToolReference) (result []CompletionTool, err error) {
	refs, err := t.getCompletionToolRefs(prg, agentGroup)
	if err != nil {
		return nil, err
	}
	return toolRefsToCompletionTools(refs, prg), nil
}

func (t Tool) addAgents(prg Program, result *toolRefSet) error {
	subToolRefs, err := t.GetAgents(prg)
	if err != nil {
		return err
	}

	for _, subToolRef := range subToolRefs {
		// don't add yourself
		if subToolRef.ToolID != t.ID {
			// Add the tool itself and no exports
			result.Add(subToolRef)
		}
	}

	return nil
}

func (t Tool) addReferencedTools(prg Program, result *toolRefSet) error {
	subToolRefs, err := t.GetToolRefsFromNames(t.Parameters.Tools)
	if err != nil {
		return err
	}

	for _, subToolRef := range subToolRefs {
		// Add the tool
		result.Add(subToolRef)

		// Get all tools exports
		result.AddAll(prg.ToolSet[subToolRef.ToolID].GetExportedTools(prg))
	}

	return nil
}

func (t Tool) addContextExportedTools(prg Program, result *toolRefSet) error {
	contextTools, err := t.GetContextTools(prg)
	if err != nil {
		return err
	}

	for _, contextTool := range contextTools {
		result.AddAll(prg.ToolSet[contextTool.ToolID].GetExportedTools(prg))
	}

	return nil
}

func (t Tool) getCompletionToolRefs(prg Program, agentGroup []ToolReference) ([]ToolReference, error) {
	result := toolRefSet{}

	for _, agent := range agentGroup {
		// don't add yourself
		if agent.ToolID != t.ID {
			result.Add(agent)
		}
	}

	if err := t.addReferencedTools(prg, &result); err != nil {
		return nil, err
	}

	if err := t.addContextExportedTools(prg, &result); err != nil {
		return nil, err
	}

	if err := t.addAgents(prg, &result); err != nil {
		return nil, err
	}

	return result.List()
}

func toolRefsToCompletionTools(completionTools []ToolReference, prg Program) (result []CompletionTool) {
	toolNames := map[string]struct{}{}

	for _, subToolRef := range completionTools {
		subTool := prg.ToolSet[subToolRef.ToolID]

		subToolName := subToolRef.Reference
		if subToolRef.Named != "" {
			subToolName = subToolRef.Named
		}

		args := subTool.Parameters.Arguments
		if args == nil && !subTool.IsCommand() && !subTool.Chat {
			args = &system.DefaultToolSchema
		} else if args == nil && !subTool.IsCommand() {
			args = &system.DefaultChatSchema
		}

		if subTool.Instructions == "" {
			log.Debugf("Skipping zero instruction tool %s (%s)", subToolName, subTool.ID)
		} else {
			result = append(result, CompletionTool{
				Function: CompletionFunctionDefinition{
					ToolID:      subTool.ID,
					Name:        PickToolName(subToolName, toolNames),
					Description: subTool.Parameters.Description,
					Parameters:  args,
				},
			})
		}
	}

	return
}

type Repo struct {
	// VCS The VCS type, such as "git"
	VCS string
	// The URL where the VCS repo can be found
	Root string
	// The path in the repo of this source. This should refer to a directory and not the actual file
	Path string
	// The filename of the source in the repo, relative to Path
	Name string
	// The revision of this source
	Revision string
}

type ToolSource struct {
	Location string `json:"location,omitempty"`
	LineNo   int    `json:"lineNo,omitempty"`
	Repo     *Repo  `json:"repo,omitempty"`
}

func (t ToolSource) String() string {
	return fmt.Sprintf("%s:%d", t.Location, t.LineNo)
}

func (t Tool) GetInterpreter() string {
	if !strings.HasPrefix(t.Instructions, CommandPrefix) {
		return ""
	}
	fields := strings.Fields(strings.TrimPrefix(t.Instructions, CommandPrefix))
	for _, field := range fields {
		name := filepath.Base(field)
		if name != "env" {
			return name
		}
	}
	return fields[0]
}

func (t Tool) IsNoop() bool {
	return t.Instructions == ""
}

func (t Tool) IsCommand() bool {
	return strings.HasPrefix(t.Instructions, CommandPrefix)
}

func (t Tool) IsDaemon() bool {
	return strings.HasPrefix(t.Instructions, DaemonPrefix)
}

func (t Tool) IsOpenAPI() bool {
	return strings.HasPrefix(t.Instructions, OpenAPIPrefix)
}

func (t Tool) IsEcho() bool {
	return strings.HasPrefix(t.Instructions, EchoPrefix)
}

func (t Tool) IsHTTP() bool {
	return strings.HasPrefix(t.Instructions, "#!http://") ||
		strings.HasPrefix(t.Instructions, "#!https://")
}

func FirstSet[T comparable](in ...T) (result T) {
	for _, i := range in {
		if i != result {
			return i
		}
	}
	return
}
