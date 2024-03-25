package llm

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/gptscript-ai/gptscript/pkg/types"
)

type Client interface {
	Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error)
	ListModels(ctx context.Context) (result []string, _ error)
	Supports(ctx context.Context, modelName string) (bool, error)
}

type Registry struct {
	clients []Client
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) AddClient(client Client) error {
	r.clients = append(r.clients, client)
	return nil
}

func (r *Registry) ListModels(ctx context.Context) (result []string, _ error) {
	for _, v := range r.clients {
		models, err := v.ListModels(ctx)
		if err != nil {
			return nil, err
		}
		result = append(result, models...)
	}
	sort.Strings(result)
	return result, nil
}

func (r *Registry) Call(ctx context.Context, messageRequest types.CompletionRequest, status chan<- types.CompletionStatus) (*types.CompletionMessage, error) {
	if messageRequest.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	var errs []error
	for _, client := range r.clients {
		ok, err := client.Supports(ctx, messageRequest.Model)
		if err != nil {
			errs = append(errs, err)
		} else if ok {
			return client.Call(ctx, messageRequest, status)
		}
	}
	if len(errs) == 0 {
		return nil, fmt.Errorf("failed to find a model provider for model [%s]", messageRequest.Model)
	}
	return nil, errors.Join(errs...)
}
