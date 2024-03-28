package remote

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
	"golang.org/x/exp/maps"
)

type Client struct {
	clientsLock sync.Mutex
	cache       *cache.Client
	clients     map[string]*openai.Client
	models      map[string]*openai.Client
	runner      *runner.Runner
	envs        []string
}

func New(r *runner.Runner, envs []string, cache *cache.Client) *Client {
	return &Client{
		cache:  cache,
		runner: r,
		envs:   envs,
	}
}

func (c *Client) Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	c.clientsLock.Lock()
	client, ok := c.models[messageRequest.Model]
	c.clientsLock.Unlock()

	if !ok {
		return nil, fmt.Errorf("failed to find remote model %s", messageRequest.Model)
	}

	_, modelName := loader.SplitToolRef(messageRequest.Model)
	messageRequest.Model = modelName
	return client.Call(ctx, messageRequest, status)
}

func (c *Client) ListModels(_ context.Context) (result []string, _ error) {
	c.clientsLock.Lock()
	defer c.clientsLock.Unlock()

	keys := maps.Keys(c.models)
	sort.Strings(keys)
	return keys, nil
}

func (c *Client) Supports(ctx context.Context, modelName string) (bool, error) {
	toolName, modelNameSuffix := loader.SplitToolRef(modelName)
	if modelNameSuffix == "" {
		return false, nil
	}

	client, err := c.load(ctx, toolName)
	if err != nil {
		return false, err
	}

	models, err := client.ListModels(ctx)
	if err != nil {
		return false, err
	}

	if !slices.Contains(models, modelNameSuffix) {
		return false, fmt.Errorf("Failed in find model [%s], supported [%s]", modelNameSuffix, strings.Join(models, ", "))
	}

	c.clientsLock.Lock()
	defer c.clientsLock.Unlock()

	if c.models == nil {
		c.models = map[string]*openai.Client{}
	}

	c.models[modelName] = client
	return true, nil
}

func isHTTPURL(toolName string) bool {
	return strings.HasPrefix(toolName, "http://") ||
		strings.HasPrefix(toolName, "https://")
}

func (c *Client) clientFromURL(apiURL string) (*openai.Client, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}
	env := strings.ToUpper(strings.ReplaceAll(parsed.Hostname(), ".", "_")) + "_API_KEY"
	apiKey := os.Getenv(env)
	if apiKey == "" {
		apiKey = "<unset>"
	}
	return openai.NewClient(openai.Options{
		BaseURL: apiURL,
		Cache:   c.cache,
		APIKey:  apiKey,
	})
}

func (c *Client) load(ctx context.Context, toolName string) (*openai.Client, error) {
	c.clientsLock.Lock()
	defer c.clientsLock.Unlock()

	client, ok := c.clients[toolName]
	if ok {
		return client, nil
	}

	if c.clients == nil {
		c.clients = make(map[string]*openai.Client)
	}

	if isHTTPURL(toolName) {
		remoteClient, err := c.clientFromURL(toolName)
		if err != nil {
			return nil, err
		}
		c.clients[toolName] = remoteClient
		return remoteClient, nil
	}

	prg, err := loader.Program(ctx, toolName, "")
	if err != nil {
		return nil, err
	}

	url, err := c.runner.Run(ctx, prg.SetBlocking(), c.envs, "")
	if err != nil {
		return nil, err
	}

	if strings.HasSuffix(url, "/") {
		url += "v1"
	} else {
		url += "/v1"
	}

	client, err = openai.NewClient(openai.Options{
		BaseURL: url,
		Cache:   c.cache,
	})
	if err != nil {
		return nil, err
	}

	c.clients[toolName] = client
	return client, nil
}
