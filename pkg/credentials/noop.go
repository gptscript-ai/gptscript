package credentials

type NoopStore struct{}

func (s NoopStore) Get(_ string) (*Credential, bool, error) {
	return nil, false, nil
}

func (s NoopStore) Add(_ Credential) error {
	return nil
}

func (s NoopStore) Remove(_ string) error {
	return nil
}

func (s NoopStore) List() ([]Credential, error) {
	return nil, nil
}
