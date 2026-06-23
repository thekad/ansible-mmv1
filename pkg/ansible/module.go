// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"fmt"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	mmv1resource "github.com/GoogleCloudPlatform/magic-modules/mmv1/api/resource"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
	"github.com/rs/zerolog/log"
	"github.com/thekad/ansible-mmv1/pkg/api"
)

type Module struct {
	Name             string
	Resource         *api.Resource
	MinVersion       string
	Options          map[string]*Option
	Documentation    *Documentation
	Returns          *ReturnBlock
	Examples         *Examples
	ArgumentSpec     *ArgumentSpec
	OperationConfigs *OperationConfigs
	Dependency       *Dependency
}

// NewFromResource creates a new Module from an API Resource
// The rule of thumb for this constructor is to build the options, examples,
// returns, and operation configs from the Mmv1 API Resource object, and then
// build the rest of the members based off the options.
func NewFromResource(resource *api.Resource) *Module {
	// Always define standard options for GCP resources
	standardOptions := map[string]*Option{
		"state": {
			Name: "state",
			Description: []string{
				"Whether the resource should exist in GCP.",
			},
			Type:    TypeStr,
			Default: "present",
			Choices: []string{"present", "absent"},
		},
	}

	options := NewOptionsFromMmv1(resource.Mmv1)
	for name, option := range standardOptions {
		options[name] = option
	}
	m := &Module{
		Name:             resource.AnsibleName(),
		Resource:         resource,
		Options:          options,
		Examples:         NewExamplesFromMmv1(resource.Mmv1),
		Returns:          NewReturnBlockFromMmv1(resource.Mmv1),
		OperationConfigs: NewOperationConfigsFromMmv1(resource.Mmv1),
	}
	argOpts := m.ArgumentOptions()
	m.Dependency = &Dependency{
		MutuallyExclusive: translateMmv1Conflicts(argOpts),
		RequiredTogether:  translateMmv1RequiredWith(argOpts),
		RequiredOneOf:     translateMmv1AtLeastOneOf(argOpts),
	}

	argumentOptions := m.ArgumentOptions()
	for name, option := range standardOptions {
		argumentOptions[name] = option
	}

	log.Info().Msgf("creating documentation for %s", resource.AnsibleName())
	m.Documentation = NewDocumentationFromOptions(resource, argumentOptions)

	log.Info().Msgf("creating argument spec for %s", resource.AnsibleName())
	m.ArgumentSpec = NewArgSpecFromOptions(argumentOptions, m.Dependency)

	return m
}

func (m *Module) String() string {
	return m.Resource.AnsibleName()
}

func (m *Module) ExcludeDelete() bool {
	return m.Resource.Mmv1.ExcludeDelete
}

// CustomCode returns the custom code (if any) defined in the API Resource YAML file
func (m *Module) CustomCode() mmv1resource.CustomCode {
	return m.Resource.Mmv1.CustomCode
}

func (m *Module) UrlParamOnlyProperties() []*mmv1api.Type {
	return google.Select(m.Resource.Mmv1.AllUserProperties(), func(p *mmv1api.Type) bool {
		return p.UrlParamOnly
	})
}

func (m *Module) BaseUrl() string {
	productVersions := m.Resource.Parent.Mmv1.Versions
	for _, version := range productVersions {
		if version.Name == m.Resource.MinVersion() {
			return ToPythonTpl(version.BaseUrl)
		}
	}

	return ""
}

func (m *Module) ModuleClass() string {
	return google.Camelize(m.Resource.Parent.Mmv1.Name, "upper")
}

func (m *Module) Kind() string {
	return fmt.Sprintf("%s#%s", google.Camelize(m.Resource.Parent.Name, "lower"), google.Camelize(m.Resource.Mmv1.Name, "lower"))
}

func (m *Module) Scopes() []string {
	return m.Resource.Parent.Mmv1.Scopes
}

func (m *Module) GetAsync() *mmv1api.Async {
	return m.Resource.Mmv1.GetAsync()
}

func (m *Module) ProductName() string {
	return m.Resource.Parent.Mmv1.Name
}

func (m *Module) AllMmv1Options() map[string]*Option {
	opts := make(map[string]*Option)
	for name, option := range m.Options {
		// we only care about options that have an mmv1 attached to them
		if option.Mmv1 == nil {
			continue
		}
		opts[name] = option
	}
	return opts
}

func (m *Module) OutputOptions() map[string]*Option {
	outputOptions := map[string]*Option{}
	for name, option := range m.AllMmv1Options() {
		if option.Output {
			outputOptions[name] = option
		}
	}

	return outputOptions
}

func (m *Module) ArgumentOptions() map[string]*Option {
	argumentOptions := map[string]*Option{}
	for name, option := range m.AllMmv1Options() {
		if option.Output || option.OutputOnly() {
			continue
		}
		argumentOptions[name] = option
	}
	return argumentOptions
}

func (m *Module) InputOptions() map[string]*Option {
	inputOptions := map[string]*Option{}
	for name, option := range m.AllMmv1Options() {
		if option.Output || option.Virtual || option.ClientSide || option.UrlParamOnly() {
			continue
		}
		inputOptions[name] = option
	}
	return inputOptions
}

func (m *Module) AllNestedOptions() map[string]*Option {
	nestedOptions := make(map[string]*Option)

	// Start with top-level options
	for _, option := range m.AllMmv1Options() {
		collectNestedOptions(option, nestedOptions)
	}

	return nestedOptions
}

// collectNestedOptions recursively collects all nested object options
func collectNestedOptions(option *Option, result map[string]*Option) {
	// If this option is a nested object or a list of nested objects, add it to the result
	if option.IsNestedObject() || option.IsNestedList() {
		result[option.ClassName()] = option
	}

	// Recursively check suboptions
	if option.Suboptions != nil {
		for _, suboption := range option.Suboptions {
			collectNestedOptions(suboption, result)
		}
	}
}
