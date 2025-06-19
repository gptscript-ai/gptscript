package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	humav2 "github.com/danielgtaylor/huma/v2"
	"github.com/gptscript-ai/gptscript/pkg/hash"
	"github.com/gptscript-ai/gptscript/pkg/mvl"
	"github.com/gptscript-ai/gptscript/pkg/types"
	nmcp "github.com/nanobot-ai/nanobot/pkg/mcp"
)

var (
	DefaultLoader = &Local{}
	DefaultRunner = DefaultLoader

	logger = mvl.Package()
)

type Local struct {
	lock       sync.Mutex
	sessions   map[string]*Session
	sessionCtx context.Context
	cancel     context.CancelFunc
}

type Session struct {
	ID     string
	Client *nmcp.Client
	Config ServerConfig
}

type Config struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// ServerConfig represents an MCP server configuration for tools calls.
// It is important that this type doesn't have any maps.
type ServerConfig struct {
	DisableInstruction bool     `json:"disableInstruction"`
	Command            string   `json:"command"`
	Args               []string `json:"args"`
	Env                []string `json:"env"`
	Server             string   `json:"server"`
	URL                string   `json:"url"`
	BaseURL            string   `json:"baseURL,omitempty"`
	Headers            []string `json:"headers"`
	Scope              string   `json:"scope"`
	AllowedTools       []string `json:"allowedTools"`
}

func (s *ServerConfig) GetBaseURL() string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	if s.Server != "" {
		return s.Server
	}
	return s.URL
}

func (l *Local) Load(ctx context.Context, tool types.Tool) (result []types.Tool, _ error) {
	if !tool.IsMCP() {
		return nil, nil
	}

	_, configData, _ := strings.Cut(tool.Instructions, "\n")

	var servers Config
	if err := json.Unmarshal([]byte(strings.TrimSpace(configData)), &servers); err != nil {
		return nil, fmt.Errorf("failed to parse MCP configuration: %w\n%s", err, configData)
	}

	if len(servers.MCPServers) == 0 {
		// Try to load just one server
		var server ServerConfig
		if err := json.Unmarshal([]byte(strings.TrimSpace(configData)), &server); err != nil {
			return nil, fmt.Errorf("failed to parse single MCP server configuration: %w\n%s", err, configData)
		}
		if server.Command == "" && server.URL == "" && server.Server == "" {
			return nil, fmt.Errorf("no MCP server configuration found in tool instructions: %s", configData)
		}
		servers.MCPServers = map[string]ServerConfig{
			"default": server,
		}
	}

	if len(servers.MCPServers) > 1 {
		return nil, fmt.Errorf("only a single MCP server definition is supported")
	}

	for server := range maps.Keys(servers.MCPServers) {
		tools, err := l.LoadTools(ctx, servers.MCPServers[server], server, tool.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to load MCP session for server %s: %w", server, err)
		}

		return tools, nil
	}

	// This should never happen, but just in case
	return nil, fmt.Errorf("no MCP server configuration found in tool instructions: %s", configData)
}

func (l *Local) LoadTools(ctx context.Context, server ServerConfig, serverName, toolName string) ([]types.Tool, error) {
	allowedTools := server.AllowedTools
	// Reset so we don't start a new MCP server, no reason to if one is already running and the allowed tools change.
	server.AllowedTools = nil

	session, err := l.loadSession(server, serverName)
	if err != nil {
		return nil, err
	}

	return l.sessionToTools(ctx, session, toolName, allowedTools)
}

func (l *Local) ShutdownServer(server ServerConfig) error {
	if l == nil {
		return nil
	}

	id := hash.Digest(server)

	l.lock.Lock()

	if l.sessionCtx == nil {
		l.lock.Unlock()
		return nil
	}

	session := l.sessions[id]
	delete(l.sessions, id)

	l.lock.Unlock()

	if session != nil && session.Client != nil {
		session.Client.Session.Close()
		session.Client.Session.Wait()
	}

	return nil
}

func (l *Local) Close() error {
	if l == nil {
		return nil
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	if l.sessionCtx == nil {
		return nil
	}

	defer func() {
		l.cancel()
		l.sessionCtx = nil
	}()

	var errs []error
	for id, session := range l.sessions {
		logger.Infof("closing MCP session %s", id)
		session.Client.Session.Close()
		session.Client.Session.Wait()
	}

	return errors.Join(errs...)
}

func (l *Local) sessionToTools(ctx context.Context, session *Session, toolName string, allowedTools []string) ([]types.Tool, error) {
	allToolsAllowed := allowedTools == nil || slices.Contains(allowedTools, "*")

	tools, err := session.Client.ListTools(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}

	toolDefs := []types.Tool{{ /* this is a placeholder for main tool */ }}
	var toolNames []string

	for _, tool := range tools.Tools {
		if !allToolsAllowed && !slices.Contains(allowedTools, tool.Name) {
			continue
		}
		if tool.Name == "" {
			// I dunno, bad tool?
			continue
		}

		var schema humav2.Schema

		schemaData, err := json.Marshal(tool.InputSchema)
		if err != nil {
			panic(err)
		}

		if err := json.Unmarshal(schemaData, &schema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tool input schema: %w", err)
		}

		annotations, err := json.Marshal(tool.Annotations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal tool annotations: %w", err)
		}

		toolDef := types.Tool{
			ToolDef: types.ToolDef{
				Parameters: types.Parameters{
					Name:        tool.Name,
					Description: tool.Description,
					Arguments:   &schema,
				},
				Instructions: types.MCPInvokePrefix + tool.Name + " " + session.ID,
			},
		}

		if string(annotations) != "{}" && string(annotations) != "null" {
			toolDef.MetaData = map[string]string{
				"mcp-tool-annotations": string(annotations),
			}
		}

		if tool.Annotations != nil && tool.Annotations.Title != "" && !slices.Contains(strings.Fields(tool.Annotations.Title), "as") {
			toolNames = append(toolNames, tool.Name+" as "+tool.Annotations.Title)
		} else {
			toolNames = append(toolNames, tool.Name)
		}

		toolDefs = append(toolDefs, toolDef)
	}

	main := types.Tool{
		ToolDef: types.ToolDef{
			Parameters: types.Parameters{
				Name:        toolName,
				Description: session.Client.Session.InitializeResult.ServerInfo.Name,
				Export:      toolNames,
			},
			MetaData: map[string]string{
				"bundle": "true",
			},
		},
	}

	if session.Client.Session.InitializeResult.Instructions != "" {
		data, _ := json.Marshal(map[string]any{
			"tools":        toolNames,
			"instructions": session.Client.Session.InitializeResult.Instructions,
		})
		toolDefs = append(toolDefs, types.Tool{
			ToolDef: types.ToolDef{
				Parameters: types.Parameters{
					Name: session.ID,
					Type: "context",
				},
				Instructions: types.EchoPrefix + "\n" + `# START MCP SERVER INFO: ` + session.Client.Session.InitializeResult.ServerInfo.Name + "\n" +
					`You have available the following tools from an MCP Server that has provided the following additional instructions` + "\n" +
					string(data) + "\n" +
					`# END MCP SERVER INFO` + "\n",
			},
		})

		main.ExportContext = append(main.ExportContext, session.ID)
	}

	toolDefs[0] = main
	return toolDefs, nil
}

func (l *Local) loadSession(server ServerConfig, serverName string, clientOpts ...nmcp.ClientOption) (*Session, error) {
	id := hash.Digest(server)
	l.lock.Lock()
	existing, ok := l.sessions[id]
	if l.sessionCtx == nil {
		l.sessionCtx, l.cancel = context.WithCancel(context.Background())
	}
	l.lock.Unlock()

	if ok {
		return existing, nil
	}

	c, err := nmcp.NewClient(l.sessionCtx, serverName, nmcp.Server{
		Unsandboxed: true,
		Env:         splitIntoMap(server.Env),
		Command:     server.Command,
		Args:        server.Args,
		BaseURL:     server.GetBaseURL(),
		Headers:     splitIntoMap(server.Headers),
	}, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP stdio client: %w", err)
	}

	result := &Session{
		ID:     id,
		Client: c,
		Config: server,
	}

	l.lock.Lock()
	defer l.lock.Unlock()

	if existing, ok = l.sessions[id]; ok {
		c.Session.Close()
		return existing, nil
	}

	if l.sessions == nil {
		l.sessions = make(map[string]*Session, 1)
	}
	l.sessions[id] = result
	return result, nil
}

func splitIntoMap(list []string) map[string]string {
	result := make(map[string]string, len(list))
	for _, s := range list {
		k, v, ok := strings.Cut(s, "=")
		if ok {
			result[k] = v
		}
	}
	return result
}
