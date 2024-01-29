package cache

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/acorn-io/gptscript/pkg/version"
	"github.com/adrg/xdg"
)

type Client struct {
	dir string
}

func New() (*Client, error) {
	dir := filepath.Join(xdg.CacheHome, version.ProgramName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Client{
		dir: dir,
	}, nil
}

func (c *Client) Store(ctx context.Context, key string, content []byte) error {
	if c == nil {
		return nil
	}
	return os.WriteFile(filepath.Join(c.dir, key), content, 0644)
}

func (c *Client) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if c == nil {
		return nil, false, nil
	}
	data, err := os.ReadFile(filepath.Join(c.dir, key))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return data, true, nil
}
