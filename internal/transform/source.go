package transform

import (
	"fmt"
	"strings"

	"gitops.szakallas.eu/plugins/internal/utils"
	"sigs.k8s.io/kustomize/kyaml/resid"
	kyaml_utils "sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// SourceSelector is the source of the replacement transformer.
type SourceSelector struct {
	// A specific object to read it from.
	resid.ResId `json:",inline,omitempty" yaml:",inline,omitempty"`

	// Structured field path expected in the allowed object.
	FieldPath string `json:"fieldPath,omitempty" yaml:"fieldPath,omitempty"`
}

func (s *SourceSelector) String() string {
	if s == nil {
		return ""
	}
	result := []string{s.ResId.String()}
	if s.FieldPath != "" {
		result = append(result, s.FieldPath)
	}
	return strings.Join(result, ":")
}

// SelectSourceNode finds the node that matches the selector, returning
// an error if multiple or none are found
func SelectSourceNode(nodes []*yaml.RNode, selector *SourceSelector) (*yaml.RNode, error) {
	var matches []*yaml.RNode
	for _, n := range nodes {
		ids, err := utils.MakeResIds(n)
		if err != nil {
			return nil, fmt.Errorf("error getting node IDs: %w", err)
		}
		for _, id := range ids {
			if id.IsSelectedBy(selector.ResId) {
				if len(matches) > 0 {
					return nil, fmt.Errorf(
						"multiple matches for selector %s", selector)
				}
				matches = append(matches, n)
				break
			}
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("nothing selected by %s", selector)
	}

	source := matches[0]

	fieldPath := kyaml_utils.SmarterPathSplitter(selector.FieldPath, ".")

	rn, err := source.Pipe(yaml.Lookup(fieldPath...))
	if err != nil {
		return nil, fmt.Errorf("error looking up replacement source: %w", err)
	}
	if rn.IsNilOrEmpty() {
		return nil, fmt.Errorf("fieldPath `%s` is missing for replacement source %s", selector.FieldPath, selector.ResId)
	}

	return rn, nil
}
