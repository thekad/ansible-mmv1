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

// Package-level documentation constants shared by all module constructors.
var (
	standardAuthors      = []string{"Google Inc. (@googlecloudplatform)"}
	standardDocFragments = []string{"google.cloud.gcp"}
)

// Documentation represents the complete module specification, used for both
// regular modules and info modules.
type Documentation struct {
	// Module name - must match the filename without .py extension
	Module string `yaml:"module"`

	// Short description displayed in ansible-doc -l
	ShortDescription string `yaml:"short_description"`

	// Detailed description - string or list of strings
	Description []string `yaml:"description"`

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
func NewDocumentationFromOptions(resource *api.Resource, options map[string]*Option) *Documentation {
	return &Documentation{
		Module:           resource.AnsibleName(),
		Author:           standardAuthors,
		ShortDescription: fmt.Sprintf("Creates a GCP %s.%s resource", resource.Parent.Mmv1.Name, resource.Mmv1.Name),
		Description:      cleanModuleDescription(resource.Mmv1.Description),
		Options:          options,
		Requirements:     standardModuleRequirements,
		Notes:            buildResourceNotes(resource.Mmv1),
		DocFragments:     standardDocFragments,
	}
}

// NewDocumentationInfo builds the DOCUMENTATION block for an info module.
// urlParamOnlyOptions are merged into Options alongside the fixed filters entry
// so that every argument_spec entry has a corresponding DOCUMENTATION entry.
func NewDocumentationInfo(resource *api.Resource, urlParamOnlyOptions []*Option) *Documentation {
	options := map[string]*Option{
		"filters": {
			Description: []string{
				"A list of filter expression strings used to filter the resources returned by the API.",
				"Each string is a filter expression (e.g. C(some_field = \"SOME_VALUE\")).",
				"Multiple expressions are combined with a logical AND.",
				"Refer to the filter topic documentation U(https://cloud.google.com/sdk/gcloud/reference/topic/filters).",
				"Refer to the IAP-160 filter syntax documentation U(https://google.aip.dev/160).",
			},
			Type:     TypeList,
			Elements: TypeStr,
			Required: false,
		},
	}

	// Merge URL-param-only options so each has a DOCUMENTATION entry.
	// URL path parameters are always scalar strings regardless of their MMv1 type,
	// so force TypeStr and clear any list/nested-object metadata.
	for _, opt := range urlParamOnlyOptions {
		docOpt := *opt // shallow copy — don't mutate the shared Option
		docOpt.Type = TypeStr
		docOpt.Elements = ""
		docOpt.Suboptions = nil
		options[opt.AnsibleName()] = &docOpt
	}

	return &Documentation{
		Module:           resource.AnsibleName() + "_info",
		ShortDescription: fmt.Sprintf("List GCP %s resources", resource.FriendlyName()),
		Description:      cleanModuleDescription(resource.Mmv1.Description),
		Author:           standardAuthors,
		Requirements:     standardModuleRequirements,
		Notes:            buildResourceNotes(resource.Mmv1),
		Options:          options,
		DocFragments:     standardDocFragments,
	}
}

func cleanModuleDescription(description string) []string {
	var cleanLines []string
	for _, line := range strings.Split(description, "\n") {
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
