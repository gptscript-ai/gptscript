package remote

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/gptscript-ai/gptscript/pkg/cache"
	"github.com/gptscript-ai/gptscript/pkg/credentials"
	"github.com/gptscript-ai/gptscript/pkg/engine"
	env2 "github.com/gptscript-ai/gptscript/pkg/env"
	"github.com/gptscript-ai/gptscript/pkg/loader"
	"github.com/gptscript-ai/gptscript/pkg/openai"
	"github.com/gptscript-ai/gptscript/pkg/prompt"
	"github.com/gptscript-ai/gptscript/pkg/runner"
	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Client struct {
	clientsLock     sync.Mutex
	cache           *cache.Client
	clients         map[string]clientInfo
	runner          *runner.Runner
	envs            []string
	credStore       credentials.CredentialStore
	defaultProvider string
}

func New(r *runner.Runner, envs []string, cache *cache.Client, credStore credentials.CredentialStore, defaultProvider string) *Client {
	return &Client{
		cache:           cache,
		runner:          r,
		envs:            envs,
		credStore:       credStore,
		defaultProvider: defaultProvider,
		clients:         make(map[string]clientInfo),
	}
}

func (c *Client) Call(ctx context.Context, messageRequest types.CompletionRequest, env []string, status chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	_, provider := c.parseModel(messageRequest.Model)
	if provider == "" {
		return nil, fmt.Errorf("failed to find remote model %s", messageRequest.Model)
	}

	client, err := c.load(ctx, provider, env...)
	if err != nil {
		return nil, err
	}

	toolName, modelName := types.SplitToolRef(messageRequest.Model)
	if modelName == "" {
		// modelName is empty, then the messageRequest.Model is not of the form 'modelName from provider'
		// Therefore, the modelName is the toolName
		modelName = toolName
	}
	messageRequest.Model = modelName
	return client.Call(ctx, messageRequest, env, status)
}

func (c *Client) ListModels(ctx context.Context, providers ...string) (result []string, _ error) {
	for _, provider := range providers {
		client, err := c.load(ctx, provider)
		if err != nil {
			return nil, err
		}
		models, err := client.ListModels(ctx, "")
		if err != nil {
			return nil, err
		}
		for _, model := range models {
			result = append(result, model+" from "+provider)
		}
	}

	sort.Strings(result)
	return
}

func (c *Client) parseModel(modelString string) (modelName, providerName string) {
	toolName, subTool := types.SplitToolRef(modelString)
	if subTool == "" {
		// This is just a plain model string "gpt4o"
		return toolName, c.defaultProvider
	}
	// This is a provider string "modelName from provider"
	return subTool, toolName
}

func (c *Client) Supports(ctx context.Context, modelString string) (bool, error) {
	_, providerName := c.parseModel(modelString)
	if providerName == "" {
		return false, nil
	}

	_, err := c.load(ctx, providerName)
	if err != nil {
		return false, err
	}

	return true, nil
}

func isHTTPURL(toolName string) bool {
	return strings.HasPrefix(toolName, "http://") ||
		strings.HasPrefix(toolName, "https://")
}

func (c *Client) clientFromURL(ctx context.Context, apiURL string, envs []string) (*openai.Client, error) {
	parsed, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}
	env := "GPTSCRIPT_PROVIDER_" + env2.ToEnvLike(parsed.Hostname()) + "_API_KEY"
	key := os.Getenv(env)

	if key == "" && !isLocalhost(apiURL) {
		var err error
		key, err = c.retrieveAPIKey(ctx, env, apiURL, envs)
		if err != nil {
			return nil, err
		}
	}

	return openai.NewClient(ctx, c.credStore, openai.Options{
		BaseURL: apiURL,
		Cache:   c.cache,
		APIKey:  key,
	})
}

func (c *Client) load(ctx context.Context, toolName string, env ...string) (*openai.Client, error) {
	c.clientsLock.Lock()
	defer c.clientsLock.Unlock()

	client, ok := c.clients[toolName]
	if ok && !isHTTPURL(toolName) && engine.IsDaemonRunning(client.url) {
		return client.client, nil
	}

	if isHTTPURL(toolName) {
		remoteClient, err := c.clientFromURL(ctx, toolName, env)
		if err != nil {
			return nil, err
		}
		c.clients[toolName] = clientInfo{
			client: remoteClient,
			url:    toolName,
		}
		return remoteClient, nil
	}

	prg, err := loader.Program(ctx, toolName, "", loader.Options{
		Cache: c.cache,
	})
	if err != nil {
		return nil, err
	}

	url, err := c.runner.Run(engine.WithToolCategory(ctx, engine.ProviderToolCategory), prg.SetBlocking(), c.envs, "")
	if err != nil {
		return nil, err
	}

	oClient, err := openai.NewClient(ctx, c.credStore, openai.Options{
		BaseURL:  strings.TrimSuffix(url, "/") + "/v1",
		Cache:    c.cache,
		CacheKey: prg.EntryToolID,
	})
	if err != nil {
		return nil, err
	}

	c.clients[toolName] = clientInfo{
		client: oClient,
		url:    url,
	}
	return oClient, nil
}

func (c *Client) retrieveAPIKey(ctx context.Context, env, url string, envs []string) (string, error) {
	return prompt.GetModelProviderCredential(ctx, c.credStore, url, env, fmt.Sprintf("Please provide your API key for %s", url), append(envs, c.envs...))
}

func isLocalhost(url string) bool {
	return strings.HasPrefix(url, "http://localhost") || strings.HasPrefix(url, "http://127.0.0.1") ||
		strings.HasPrefix(url, "https://localhost") || strings.HasPrefix(url, "https://127.0.0.1")
}

type clientInfo struct {
	client *openai.Client
	url    string
}
