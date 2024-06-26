package credentials

import "context"

type NoopStore struct{}

func (s NoopStore) Get(context.Context, string) (*Credential, bool, error) {
	return nil, false, nil
}

func (s NoopStore) Add(context.Context, Credential) error {
	return nil
}

func (s NoopStore) Remove(context.Context, string) error {
	return nil
}

func (s NoopStore) List(context.Context) ([]Credential, error) {
	return nil, nil
}
