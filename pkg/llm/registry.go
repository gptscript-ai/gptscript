package llm

import (
	"context"
	"fmt"
	"sort"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Client interface {
	Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error)
	ListModels(ctx context.Context) (result []string, _ error)
}

type Registry struct {
	clientsByModel map[string]Client
}

func NewRegistry() *Registry {
	return &Registry{
		clientsByModel: map[string]Client{},
	}
}

func (r *Registry) AddClient(ctx context.Context, client Client) error {
	models, err := client.ListModels(ctx)
	if err != nil {
		return err
	}
	for _, model := range models {
		r.clientsByModel[model] = client
	}
	return nil
}

func (r *Registry) ListModels(_ context.Context) (result []string, _ error) {
	for k := range r.clientsByModel {
		result = append(result, k)
	}
	sort.Strings(result)
	return result, nil
}

func (r *Registry) Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	if messageRequest.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	client, ok := r.clientsByModel[messageRequest.Model]
	if !ok {
		return nil, fmt.Errorf("model not found: %s", messageRequest.Model)
	}
	return client.Call(ctx, messageRequest, status)
}
