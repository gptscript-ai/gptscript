package mcp

import (
	nmcp "github.com/nanobot-ai/nanobot/pkg/mcp"
)

func (l *Local) Client(server ServerConfig, clientOpts ...nmcp.ClientOption) (*Client, error) {
	session, err := l.loadSession(server, "default", clientOpts...)
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
