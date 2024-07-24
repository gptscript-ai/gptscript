package openapi

import (
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type OperationList struct {
	Operations map[string]Operation `json:"operations"`
}

type Operation struct {
	Description string `json:"description,omitempty"`
	Summary     string `json:"summary,omitempty"`
}

const NoFilter = "<none>"

func List(t *openapi3.T, filter string) (OperationList, error) {
	operations := make(map[string]Operation)
	for _, pathItem := range t.Paths.Map() {
		for _, operation := range pathItem.Operations() {
			var (
				match bool
				err   error
			)
			if filter != "" && filter != NoFilter {
				if strings.Contains(filter, "*") {
					var filters []string
					if strings.Contains(filter, "|") {
						filters = strings.Split(filter, "|")
					} else {
						filters = []string{filter}
					}

					match, err = MatchFilters(filters, operation.OperationID)
					if err != nil {
						return OperationList{}, err
					}
				} else {
					match = operation.OperationID == filter
				}
			} else {
				match = true
			}

			if match {
				operations[operation.OperationID] = Operation{
					Description: operation.Description,
					Summary:     operation.Summary,
				}
			}
		}
	}

	return OperationList{Operations: operations}, nil
}

func MatchFilters(filters []string, operationID string) (bool, error) {
	for _, filter := range filters {
		match, err := filepath.Match(filter, operationID)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}
