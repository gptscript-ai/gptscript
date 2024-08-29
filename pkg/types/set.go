package types

type toolRefKey struct {
	name   string
	toolID string
	arg    string
}

type toolRefSet struct {
	set   map[toolRefKey]ToolReference
	order []toolRefKey
	err   error
}

func (t *toolRefSet) List() (result []ToolReference, err error) {
	for _, k := range t.order {
		result = append(result, t.set[k])
	}
	return result, t.err
}

func (t *toolRefSet) Contains(value ToolReference) bool {
	key := toolRefKey{
		name:   value.Named,
		toolID: value.ToolID,
		arg:    value.Arg,
	}

	_, ok := t.set[key]
	return ok
}

func (t *toolRefSet) HasTool(toolID string) bool {
	for _, ref := range t.set {
		if ref.ToolID == toolID {
			return true
		}
	}
	return false
}

func (t *toolRefSet) AddAll(values []ToolReference, err error) {
	if err != nil {
		t.err = err
	}
	for _, v := range values {
		t.Add(v)
	}
}

func (t *toolRefSet) Add(value ToolReference) {
	key := toolRefKey{
		name:   value.Named,
		toolID: value.ToolID,
		arg:    value.Arg,
	}

	if _, ok := t.set[key]; !ok {
		if t.set == nil {
			t.set = map[toolRefKey]ToolReference{}
		}
		t.set[key] = value
		t.order = append(t.order, key)
	}
}
