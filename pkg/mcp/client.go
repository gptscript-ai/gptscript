package mcp

import (
	"context"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type Client interface {
	mcpclient.MCPClient
	Capabilities() mcp.ServerCapabilities
}

func (l *Local) Client(server ServerConfig) (Client, error) {
	session, err := l.loadSession(server, true)
	if err != nil {
		return nil, err
	}

	return &client{session}, nil
}

type client struct {
	*Session
}

func (c *client) Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	return c.Client.Initialize(ctx, request)
}

func (c *client) Ping(ctx context.Context) error {
	return c.Client.Ping(ctx)
}

func (c *client) ListResourcesByPage(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	return c.Client.ListResourcesByPage(ctx, request)
}

func (c *client) ListResources(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	return c.Client.ListResources(ctx, request)
}

func (c *client) ListResourceTemplatesByPage(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	return c.Client.ListResourceTemplatesByPage(ctx, request)
}

func (c *client) ListResourceTemplates(ctx context.Context, request mcp.ListResourceTemplatesRequest) (*mcp.ListResourceTemplatesResult, error) {
	return c.Client.ListResourceTemplates(ctx, request)
}

func (c *client) ReadResource(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return c.Client.ReadResource(ctx, request)
}

func (c *client) Subscribe(ctx context.Context, request mcp.SubscribeRequest) error {
	return c.Client.Subscribe(ctx, request)
}

func (c *client) Unsubscribe(ctx context.Context, request mcp.UnsubscribeRequest) error {
	return c.Client.Unsubscribe(ctx, request)
}

func (c *client) ListPromptsByPage(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	return c.Client.ListPromptsByPage(ctx, request)
}

func (c *client) ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	return c.Client.ListPrompts(ctx, request)
}

func (c *client) GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return c.Client.GetPrompt(ctx, request)
}

func (c *client) ListToolsByPage(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return c.Client.ListToolsByPage(ctx, request)
}

func (c *client) ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	return c.Client.ListTools(ctx, request)
}

func (c *client) CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return c.Client.CallTool(ctx, request)
}

func (c *client) SetLevel(ctx context.Context, request mcp.SetLevelRequest) error {
	return c.Client.SetLevel(ctx, request)
}

func (c *client) Complete(ctx context.Context, request mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	return c.Client.Complete(ctx, request)
}

func (c *client) Close() error {
	return c.Client.Close()
}

func (c *client) OnNotification(handler func(notification mcp.JSONRPCNotification)) {
	c.Client.OnNotification(handler)
}

func (c *client) Capabilities() mcp.ServerCapabilities {
	return c.InitResult.Capabilities
}
