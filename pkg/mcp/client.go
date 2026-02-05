package mcp

import (
	nmcp "github.com/gptscript-ai/gptscript/pkg/nanobot/mcp"
)

func (l *Local) Client(server ServerConfig, clientOpts ...nmcp.ClientOption) (*Client, error) {
	session, err := l.loadSession(server, "default", clientOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: session.Client,
		ID:     session.ID,
	}, nil
}

type Client struct {
	*nmcp.Client
	ID string
}

func (c *Client) Capabilities() nmcp.ServerCapabilities {
	return c.Session.InitializeResult.Capabilities
}
