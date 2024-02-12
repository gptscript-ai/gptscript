package cache

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"github.com/gptscript-ai/gptscript/pkg/version"
)

type Client struct {
	dir  string
	noop bool
}

type Options struct {
	Cache    *bool  `usage:"Disable caching" default:"true"`
	CacheDir string `usage:"Directory to store cache (default: $XDG_CACHE_HOME/gptscript)"`
}

func complete(opts ...Options) (result Options) {
	for _, opt := range opts {
		result.CacheDir = types.FirstSet(opt.CacheDir, result.CacheDir)
		result.Cache = types.FirstSet(opt.Cache, result.Cache)
	}
	if result.Cache == nil {
		result.Cache = &[]bool{true}[0]
	}
	if result.CacheDir == "" {
		result.CacheDir = filepath.Join(xdg.CacheHome, version.ProgramName)
	}
	return
}

func New(opts ...Options) (*Client, error) {
	opt := complete(opts...)
	if err := os.MkdirAll(opt.CacheDir, 0755); err != nil {
		return nil, err
	}
	return &Client{
		dir:  opt.CacheDir,
		noop: !*opt.Cache,
	}, nil
}

func (c *Client) Store(key string, content []byte) error {
	if c == nil || c.noop {
		return nil
	}
	return os.WriteFile(filepath.Join(c.dir, key), content, 0644)
}

func (c *Client) Get(key string) ([]byte, bool, error) {
	if c == nil || c.noop {
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
