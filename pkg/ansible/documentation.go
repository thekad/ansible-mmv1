// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"fmt"
	"sort"
	"strings"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	"github.com/thekad/ansible-mmv1/pkg/api"
)

// Documentation represents the complete module specification, used for both
// regular modules and info modules.
type Documentation struct {
	// Module name - must match the filename without .py extension
	Module string `yaml:"module"`

	// Short description displayed in ansible-doc -l
	ShortDescription string `yaml:"short_description"`

	// Detailed description - string or list of strings
	Description []string `yaml:"description,omitempty"`

	// Author information - string or list of strings
	Author []string `yaml:"author,omitempty"`

	// Module options
	Options map[string]*Option `yaml:"options,omitempty"`

	// Requirements for the module to work
	Requirements []string `yaml:"requirements,omitempty"`

	// Notes about the module
	Notes []string `yaml:"notes,omitempty"`

	// DocFragments are fragments of shared documentation that will be included in the documentation
	DocFragments []string `yaml:"extends_documentation_fragment,omitempty"`
}

// buildResourceNotes constructs the sorted notes slice (API reference + guide
// links) that both module constructors emit.
func buildResourceNotes(mmv1 *mmv1api.Resource) []string {
	if mmv1 == nil {
		return nil
	}
	notes := []string{
		fmt.Sprintf("API Reference: U(%s)", mmv1.References.Api),
	}
	for name, guide := range mmv1.References.Guides {
		if name == "" || guide == "" {
			continue
		}
		notes = append(notes, fmt.Sprintf("%s Guide: U(%s)", name, guide))
	}
	sort.Strings(notes)
	return notes
}

// NewDocumentationFromOptions creates a new Documentation from a resource and options.
func NewDocumentationFromOptions(resource *api.Resource, options map[string]*Option, authors []string, docFragments []string) *Documentation {
	return &Documentation{
		Module:           resource.AnsibleName(),
		Author:           authors,
		ShortDescription: fmt.Sprintf("Manages a %s resource", resource.FriendlyName()),
		Description:      cleanModuleDescription(resource.Mmv1.Description),
		Options:          options,
		Requirements:     standardModuleRequirements,
		Notes:            buildResourceNotes(resource.Mmv1),
		DocFragments:     docFragments,
	}
}

// NewDocumentationInfo builds the DOCUMENTATION block for an info module.
// urlParamOnlyOptions are merged into Options alongside the fixed filters entry
// so that every argument_spec entry has a corresponding DOCUMENTATION entry.
func NewDocumentationInfo(resource *api.Resource, urlParamOnlyOptions []*Option, authors []string, docFragments []string) *Documentation {
	options := map[string]*Option{
		"filters": {
			Type:     TypeList,
			Elements: TypeStr,
			Required: false,
		},
	}

	// Merge URL-param-only options so each has a DOCUMENTATION entry.
	// URL path parameters are always scalar strings regardless of their MMv1 type,
	// so force TypeStr and clear any list/nested-object metadata.
	for _, opt := range urlParamOnlyOptions {
		docOpt := *opt // shallow copy - don't mutate the shared Option
		docOpt.Type = TypeStr
		docOpt.Elements = ""
		docOpt.Suboptions = nil
		options[opt.AnsibleName()] = &docOpt
	}

	return &Documentation{
		Module:           resource.AnsibleName() + "_info",
		ShortDescription: fmt.Sprintf("List %s resources", resource.FriendlyName()),
		Description:      cleanModuleDescription(resource.Mmv1.Description),
		Author:           authors,
		Requirements:     standardModuleRequirements,
		Notes:            buildResourceNotes(resource.Mmv1),
		Options:          options,
		DocFragments:     docFragments,
	}
}

func cleanModuleDescription(description string) []string {
	var cleanLines []string
	for line := range strings.SplitSeq(description, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return cleanLines
}

// ToString serialises the documentation block to a YAML string.
func (d *Documentation) ToString() string {
	return ToYAML(d)
}
