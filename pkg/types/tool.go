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
	BreakPrefix   = "#!sys.break"
	CommandPrefix = "#!"
)

var (
	DefaultFiles = []string{"agent.gpt", "tool.gpt"}
)

type ToolType string

const (
	ToolTypeContext    = ToolType("context")
	ToolTypeAgent      = ToolType("agent")
	ToolTypeOutput     = ToolType("output")
	ToolTypeInput      = ToolType("input")
	ToolTypeTool       = ToolType("tool")
	ToolTypeCredential = ToolType("credential")
	ToolTypeDefault    = ToolType("")

	// The following types logically exist but have no real code reference. These are kept
	// here just so that we have a comprehensive list

	ToolTypeAssistant = ToolType("assistant")
	ToolTypeProvider  = ToolType("provider")
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
	ExportCredentials   []string         `json:"exportCredentials,omitempty"`
	InputFilters        []string         `json:"inputFilters,omitempty"`
	ExportInputFilters  []string         `json:"exportInputFilters,omitempty"`
	OutputFilters       []string         `json:"outputFilters,omitempty"`
	ExportOutputFilters []string         `json:"exportOutputFilters,omitempty"`
	Blocking            bool             `json:"-"`
	Type                ToolType         `json:"type,omitempty"`
}

func (p Parameters) allExports() []string {
	return slices.Concat(
		p.ExportContext,
		p.Export,
		p.ExportCredentials,
		p.ExportInputFilters,
		p.ExportOutputFilters,
	)
}

func (p Parameters) allReferences() []string {
	return slices.Concat(
		p.GlobalTools,
		p.Tools,
		p.Context,
		p.Agents,
		p.Credentials,
		p.InputFilters,
		p.OutputFilters,
	)
}

func (p Parameters) ToolRefNames() []string {
	return slices.Concat(
		p.Tools,
		p.Agents,
		p.Export,
		p.ExportContext,
		p.Context,
		p.Credentials,
		p.ExportCredentials,
		p.InputFilters,
		p.ExportInputFilters,
		p.OutputFilters,
		p.ExportOutputFilters)
}

type ToolDef struct {
	Parameters   `json:",inline"`
	Instructions string            `json:"instructions,omitempty"`
	BuiltinFunc  BuiltinFunc       `json:"-"`
	MetaData     map[string]string `json:"metaData,omitempty"`
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
	if t.Parameters.Type != ToolTypeDefault {
		_, _ = fmt.Fprintf(buf, "Type: %s\n", strings.ToUpper(string(t.Type[0]))+string(t.Type[1:]))
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
	if len(t.Parameters.ExportCredentials) > 0 {
		for _, exportCred := range t.Parameters.ExportCredentials {
			_, _ = fmt.Fprintf(buf, "Share Credential: %s\n", exportCred)
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

	if t.Name != "" {
		keys := maps.Keys(t.MetaData)
		sort.Strings(keys)
		for _, key := range keys {
			buf.WriteString("---\n")
			buf.WriteString("!metadata:")
			buf.WriteString(t.Name)
			buf.WriteString(":")
			buf.WriteString(key)
			buf.WriteString("\n")
			buf.WriteString(t.MetaData[key])
			buf.WriteString("\n")
		}
	}

	return buf.String()
}

func (t Tool) GetNextAgentGroup(prg *Program, agentGroup []ToolReference, toolID string) (result []ToolReference, _ error) {
	newAgentGroup := toolRefSet{}
	newAgentGroup.AddAll(t.GetToolsByType(prg, ToolTypeAgent))

	if newAgentGroup.HasTool(toolID) {
		// Join new agent group
		return newAgentGroup.List()
	}

	return agentGroup, nil
}

func (t Tool) getAgents(prg *Program) (result []ToolReference, _ error) {
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
		if trimmed := strings.TrimSuffix(strings.TrimSuffix(normed, "Agent"), "Assistant"); trimmed != "" {
			normed = trimmed
		}
		toolRefs[i].Named = normed
	}

	return toolRefs, nil
}

func (t Tool) GetToolsByType(prg *Program, toolType ToolType) ([]ToolReference, error) {
	if toolType == ToolTypeAgent {
		// Agents are special, they can only be sourced from direct references and not the generic 'tool:' or shared by references
		return t.getAgents(prg)
	}

	toolSet := &toolRefSet{}

	var (
		directRefs          []string
		toolsListFilterType = []ToolType{toolType}
	)

	switch toolType {
	case ToolTypeContext:
		directRefs = t.Context
	case ToolTypeOutput:
		directRefs = t.OutputFilters
	case ToolTypeInput:
		directRefs = t.InputFilters
	case ToolTypeTool:
		toolsListFilterType = append(toolsListFilterType, ToolTypeDefault, ToolTypeAgent)
	case ToolTypeCredential:
		directRefs = t.Credentials
	default:
		return nil, fmt.Errorf("unknown tool type %v", toolType)
	}

	toolSet.AddAll(t.GetToolRefsFromNames(directRefs))

	toolRefs, err := t.GetToolRefsFromNames(t.Tools)
	if err != nil {
		return nil, err
	}

	for _, toolRef := range toolRefs {
		tool, ok := prg.ToolSet[toolRef.ToolID]
		if !ok {
			continue
		}
		if slices.Contains(toolsListFilterType, tool.Type) {
			toolSet.Add(toolRef)
		}
	}

	exportSources, err := t.getExportSources(prg)
	if err != nil {
		return nil, err
	}

	for _, exportSource := range exportSources {
		var (
			tool       = prg.ToolSet[exportSource.ToolID]
			exportRefs []string
		)

		switch toolType {
		case ToolTypeContext:
			exportRefs = tool.ExportContext
		case ToolTypeOutput:
			exportRefs = tool.ExportOutputFilters
		case ToolTypeInput:
			exportRefs = tool.ExportInputFilters
		case ToolTypeTool:
			exportRefs = tool.Export
		case ToolTypeCredential:
			exportRefs = tool.ExportCredentials
		default:
			return nil, fmt.Errorf("unknown tool type %v", toolType)
		}
		toolSet.AddAll(tool.GetToolRefsFromNames(exportRefs))
	}

	return toolSet.List()
}

func (t Tool) addExportsRecursively(prg *Program, toolSet *toolRefSet) error {
	toolRefs, err := t.GetToolRefsFromNames(t.allExports())
	if err != nil {
		return err
	}

	for _, toolRef := range toolRefs {
		if toolSet.Contains(toolRef) {
			continue
		}

		toolSet.Add(toolRef)
		if err := prg.ToolSet[toolRef.ToolID].addExportsRecursively(prg, toolSet); err != nil {
			return err
		}
	}

	return nil
}

func (t Tool) getExportSources(prg *Program) ([]ToolReference, error) {
	// We start first with all references from this tool. This gives us the
	// initial set of export sources.
	// Then all tools in the export sources in the set we look for exports of those tools recursively.
	// So a share of a share of a share should be added.

	toolSet := toolRefSet{}
	toolRefs, err := t.GetToolRefsFromNames(t.allReferences())
	if err != nil {
		return nil, err
	}

	for _, toolRef := range toolRefs {
		if err := prg.ToolSet[toolRef.ToolID].addExportsRecursively(prg, &toolSet); err != nil {
			return nil, err
		}
		toolSet.Add(toolRef)
	}

	return toolSet.List()
}

func (t Tool) GetChatCompletionTools(prg Program, agentGroup ...ToolReference) (result []ChatCompletionTool, err error) {
	toolSet := &toolRefSet{}
	toolSet.AddAll(t.GetToolsByType(&prg, ToolTypeTool))
	toolSet.AddAll(t.GetToolsByType(&prg, ToolTypeAgent))

	if t.Chat {
		for _, agent := range agentGroup {
			// don't add yourself
			if agent.ToolID != t.ID {
				toolSet.Add(agent)
			}
		}
	}

	refs, err := toolSet.List()
	if err != nil {
		return nil, err
	}

	return toolRefsToCompletionTools(refs, prg), nil
}

func toolRefsToCompletionTools(completionTools []ToolReference, prg Program) (result []ChatCompletionTool) {
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
			result = append(result, ChatCompletionTool{
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

func (t ToolSource) IsGit() bool {
	return t.Repo != nil && t.Repo.VCS == "git"
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

func (t Tool) IsAgentsOnly() bool {
	return t.IsNoop() && len(t.Context) == 0
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
