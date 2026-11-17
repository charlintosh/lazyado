package models

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	ParamString  = "string"
	ParamNumber  = "number"
	ParamBoolean = "boolean"
)

var scalarParamTypes = map[string]string{
	"":                  ParamString,
	"string":            ParamString,
	"number":            ParamNumber,
	"boolean":           ParamBoolean,
	"filepath":          ParamString,
	"environment":       ParamString,
	"securefile":        ParamString,
	"serviceconnection": ParamString,
}

type PipelineParameter struct {
	Name        string
	DisplayName string
	Type        string
	Default     string
	Values      []string
}

func (p PipelineParameter) Label() string {
	if p.DisplayName != "" {
		return p.DisplayName
	}
	return p.Name
}

type PipelineStageSpec struct {
	Name        string
	DisplayName string
}

func (s PipelineStageSpec) Label() string {
	if s.DisplayName != "" {
		return s.DisplayName
	}
	return s.Name
}

type PipelineSpec struct {
	Parameters []PipelineParameter
	Stages     []PipelineStageSpec
	Fixed      []string
}

func ParsePipelineSpec(text string) (PipelineSpec, error) {
	var doc struct {
		Parameters yaml.Node `yaml:"parameters"`
		Stages     yaml.Node `yaml:"stages"`
	}
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil {
		return PipelineSpec{}, fmt.Errorf("reading the pipeline YAML: %w", err)
	}
	var spec PipelineSpec
	switch doc.Parameters.Kind {
	case yaml.SequenceNode:
		for _, item := range doc.Parameters.Content {
			spec.addParameter(item)
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(doc.Parameters.Content); i += 2 {
			name, value := doc.Parameters.Content[i].Value, doc.Parameters.Content[i+1]
			if value.Kind != yaml.ScalarNode {
				spec.Fixed = append(spec.Fixed, name)
				continue
			}
			spec.Parameters = append(spec.Parameters, PipelineParameter{Name: name, Type: ParamString, Default: value.Value})
		}
	}
	if doc.Stages.Kind == yaml.SequenceNode {
		for _, item := range doc.Stages.Content {
			var stage struct {
				Stage       string `yaml:"stage"`
				DisplayName string `yaml:"displayName"`
			}
			if item.Kind == yaml.MappingNode && item.Decode(&stage) == nil && stage.Stage != "" {
				spec.Stages = append(spec.Stages, PipelineStageSpec{Name: stage.Stage, DisplayName: stage.DisplayName})
			}
		}
	}
	return spec, nil
}

func (s *PipelineSpec) addParameter(node *yaml.Node) {
	var raw struct {
		Name        string    `yaml:"name"`
		DisplayName string    `yaml:"displayName"`
		Type        string    `yaml:"type"`
		Default     yaml.Node `yaml:"default"`
		Values      []string  `yaml:"values"`
	}
	if node.Kind != yaml.MappingNode || node.Decode(&raw) != nil || raw.Name == "" {
		return
	}
	kind, ok := scalarParamTypes[strings.ToLower(raw.Type)]
	if !ok || (raw.Default.Kind != 0 && raw.Default.Kind != yaml.ScalarNode) {
		s.Fixed = append(s.Fixed, raw.Name)
		return
	}
	s.Parameters = append(s.Parameters, PipelineParameter{
		Name:        raw.Name,
		DisplayName: raw.DisplayName,
		Type:        kind,
		Default:     raw.Default.Value,
		Values:      raw.Values,
	})
}
