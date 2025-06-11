package mcp

import (
	nmcp "github.com/nanobot-ai/nanobot/pkg/mcp"
)

func (l *Local) Client(server ServerConfig) (*Client, error) {
	session, err := l.loadSession(server, "default")
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: session.Client,
	}, nil
}

type Client struct {
	*nmcp.Client
}

func (c *Client) Capabilities() nmcp.ServerCapabilities {
	return c.Session.InitializeResult.Capabilities
}
