// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"fmt"
	"sort"
	"strings"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
	"github.com/rs/zerolog/log"
	"github.com/thekad/ansible-mmv1/pkg/api"
)

// ---------------------------------------------------------------------------
// ArgumentInfoSpec
// ---------------------------------------------------------------------------

// ArgumentInfoSpec is the argument_spec for an info module.
// Module-specific parameters declared here:
//   - one entry per URL-param-only property (e.g. location, cluster_id) — these
//     are required to construct the API list URL.
//   - filters — optional list of filter expression strings forwarded to the backend.
//
// All other auth parameters (project, auth_kind, service_account_*, scopes, etc.)
// are injected automatically by gcp_v2.Module and are covered by the
// google.cloud.gcp documentation fragment; they must not be repeated here.
type ArgumentInfoSpec struct {
	UrlParamOnlyOptions []*Option
}

// NewArgumentInfoSpec creates an ArgumentInfoSpec for the given URL-param-only options.
func NewArgumentInfoSpec(urlParamOnlyOptions []*Option) *ArgumentInfoSpec {
	return &ArgumentInfoSpec{
		UrlParamOnlyOptions: urlParamOnlyOptions,
	}
}

// ToString emits the Python argument_spec=dict(...) snippet.
// All entries are sorted alphabetically; URL-param-only options are interleaved
// with the fixed filters entry.
func (a *ArgumentInfoSpec) ToString() string {
	var b strings.Builder

	b.WriteString("argument_spec=dict(\n")

	// Collect all entry names so we can sort them together.
	type entry struct {
		name  string
		lines []string
	}

	entries := []entry{}

	// URL-param-only properties — required, typed, no default.
	for _, opt := range a.UrlParamOnlyOptions {
		lines := []string{
			"    " + pythonIdentifier(opt.AnsibleName()) + "=dict(\n",
			"        type=" + pythonQuote("str") + ",\n",
		}
		if opt.Required {
			lines = append(lines, "        required=True,\n")
		}
		lines = append(lines, "    ),\n")
		entries = append(entries, entry{name: opt.AnsibleName(), lines: lines})
	}

	// filters — list of filter expression strings joined by AND, optional.
	entries = append(entries, entry{
		name: "filters",
		lines: []string{
			"    filters=dict(\n",
			"        type=" + pythonQuote("list") + ",\n",
			"        elements=" + pythonQuote("str") + ",\n",
			"    ),\n",
		},
	})

	// Sort alphabetically by name.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	for _, e := range entries {
		for _, line := range e.lines {
			b.WriteString(line)
		}
	}

	b.WriteString(")")
	return b.String()
}

// ---------------------------------------------------------------------------
// DocumentationInfo
// ---------------------------------------------------------------------------

// DocumentationInfo is the DOCUMENTATION block for an info module.
// It hardwires a single option (filters) and uses the google.cloud.gcp
// documentation fragment to cover all standard auth parameters.
type DocumentationInfo struct {
	Module           string             `yaml:"module"`
	ShortDescription string             `yaml:"short_description"`
	Description      []string           `yaml:"description"`
	Author           []string           `yaml:"author,omitempty"`
	Requirements     []string           `yaml:"requirements,omitempty"`
	Notes            []string           `yaml:"notes,omitempty"`
	Options          map[string]*Option `yaml:"options,omitempty"`
	DocFragments     []string           `yaml:"extends_documentation_fragment,omitempty"`
}

// NewDocumentationInfo builds the DOCUMENTATION block for an info module.
// urlParamOnlyOptions are merged into Options alongside the fixed filters entry
// so that every argument_spec entry has a corresponding DOCUMENTATION entry.
func NewDocumentationInfo(resource *api.Resource, urlParamOnlyOptions []*Option) *DocumentationInfo {
	notes := []string{
		fmt.Sprintf("API Reference: U(%s)", resource.Mmv1.References.Api),
	}
	for name, guide := range resource.Mmv1.References.Guides {
		notes = append(notes, fmt.Sprintf("%s Guide: U(%s)", name, guide))
	}
	sort.Strings(notes)

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

	return &DocumentationInfo{
		Module:           resource.AnsibleName() + "_info",
		ShortDescription: fmt.Sprintf("List GCP %s resources", resource.FriendlyName()),
		Description:      cleanModuleDescription(resource.Mmv1.Description),
		Author:           []string{"Google Inc. (@googlecloudplatform)"},
		Requirements:     standardModuleRequirements,
		Notes:            notes,
		Options:          options,
		DocFragments:     []string{"google.cloud.gcp"},
	}
}

// ToString serialises the documentation block to a YAML string.
func (d *DocumentationInfo) ToString() string {
	return ToYAML(d)
}

// ---------------------------------------------------------------------------
// ReturnInfo
// ---------------------------------------------------------------------------

// ReturnInfo is the RETURN block for an info module.
// It always documents exactly two values: changed (always false) and items
// (a list of zero or more resources matching the supplied filters).
type ReturnInfo struct {
	ResourceKind string // e.g. "AlloyDB.Cluster"
}

// NewReturnInfo creates a ReturnInfo for the given resource.
func NewReturnInfo(resource *api.Resource) *ReturnInfo {
	return &ReturnInfo{
		ResourceKind: resource.Parent.Mmv1.Name + "." + resource.Mmv1.Name,
	}
}

// ToString serialises the return block to a YAML string.
func (r *ReturnInfo) ToString() string {
	returns := map[string]*ReturnAttribute{
		"changed": {
			Description: "Whether any changes were made (always false for info modules).",
			Returned:    "always",
			Type:        ReturnTypeBool,
		},
		"resources": {
			Description: fmt.Sprintf(
				"List of %s resources matching the supplied filters. "+
					"May be empty, contain a single resource, or multiple resources.",
				r.ResourceKind,
			),
			Returned: "always",
			Type:     ReturnTypeList,
			Elements: ReturnTypeDict,
		},
	}
	return ToYAML(returns)
}

// ---------------------------------------------------------------------------
// InfoModule
// ---------------------------------------------------------------------------

// InfoModule is the top-level data structure passed to module_info.tmpl.
// It is constructed independently of Module — there is no shared state between
// the two and InfoModule carries no CustomCode, no nested options, no operation
// configs, and no CRUD logic.
type InfoModule struct {
	Name              string
	Resource          *api.Resource
	MinVersion        string
	DocumentationInfo *DocumentationInfo
	ReturnInfo        *ReturnInfo
	ArgumentInfoSpec  *ArgumentInfoSpec
	OperationConfigs  *OperationConfigs
	CollectionKey     string
}

// filterOptionsInUrl returns only those options from the given map whose
// placeholder appears in the URL path. Both the MMv1 camelCase property name
// (e.g. {location}, {cluster}) and its Ansible snake_case equivalent
// (e.g. {instance_id}) are tested, since MMv1 authors use either form.
// Both consumers of the returned slice (ArgumentInfoSpec.ToString and
// NewDocumentationInfo) sort independently, so input order does not matter.
func filterOptionsInUrl(options map[string]*Option, urlPath string) []*Option {
	var filtered []*Option
	for _, opt := range options {
		if strings.Contains(urlPath, "{"+opt.Name+"}") ||
			strings.Contains(urlPath, "{"+opt.AnsibleName()+"}") {
			filtered = append(filtered, opt)
		}
	}
	return filtered
}

// NewInfoFromResource constructs an InfoModule from an API resource.
func NewInfoFromResource(resource *api.Resource) *InfoModule {
	log.Info().Msgf("creating info module for %s", resource.AnsibleName())

	opConfigs := NewOperationConfigsFromMmv1(resource.Mmv1)

	// Strip any query string from the collection URL so that query-parameter
	// placeholders (e.g. ?instanceId={instance_id} in alloydb.Instance) are
	// not mistaken for path placeholders when filtering options.
	collectionUrl, _, _ := strings.Cut(opConfigs.CollectionUrl, "?")

	// Convert all url-param-only properties to Option objects, then keep only
	// those whose placeholder appears in the collection URL path.
	// Resource-id params (e.g. clusterId) appear in the self-link URL only and
	// must not be included in the info module argument spec.
	urlParamOnlyProps := google.Select(resource.Mmv1.AllUserProperties(), func(p *mmv1api.Type) bool {
		return p.UrlParamOnly
	})
	urlParamOnlyOptions := filterOptionsInUrl(
		convertPropertiesToOptions(urlParamOnlyProps, nil, false, false),
		collectionUrl,
	)
	log.Debug().Msgf("info module %s: collection path %q, url-param-only options: %v",
		resource.AnsibleName(), collectionUrl, func() []string {
			names := make([]string, len(urlParamOnlyOptions))
			for i, o := range urlParamOnlyOptions {
				names[i] = o.AnsibleName()
			}
			return names
		}())

	return &InfoModule{
		Name:              resource.AnsibleName() + "_info",
		Resource:          resource,
		DocumentationInfo: NewDocumentationInfo(resource, urlParamOnlyOptions),
		ReturnInfo:        NewReturnInfo(resource),
		ArgumentInfoSpec:  NewArgumentInfoSpec(urlParamOnlyOptions),
		OperationConfigs:  NewOperationConfigsFromMmv1(resource.Mmv1),
		CollectionKey:     resource.Mmv1.CollectionUrlKey,
	}
}

// String returns the module name, used as the output filename stem.
func (m *InfoModule) String() string {
	return m.Name
}

// Scopes returns the OAuth scopes required by the resource.
func (m *InfoModule) Scopes() []string {
	return m.Resource.Parent.Mmv1.Scopes
}

// ProductName returns the GCP product name (e.g. "AlloyDB").
func (m *InfoModule) ProductName() string {
	return m.Resource.Parent.Mmv1.Name
}

// Kind returns the resource kind string used to identify the resource in the API
// (e.g. "alloydb#cluster").
func (m *InfoModule) Kind() string {
	return fmt.Sprintf(
		"%s#%s",
		google.Camelize(m.Resource.Parent.Name, "lower"),
		google.Camelize(m.Resource.Mmv1.Name, "lower"),
	)
}

// AllUserProperties returns all user-facing properties from the MMv1 resource,
// used by the template to enumerate URL-param-only properties.
func (m *InfoModule) UrlParamOnlyProperties() []*mmv1api.Type {
	return google.Select(m.Resource.Mmv1.AllUserProperties(), func(p *mmv1api.Type) bool {
		return p.UrlParamOnly
	})
}

func (m *InfoModule) BaseUrl() string {
	productVersions := m.Resource.Parent.Mmv1.Versions
	for _, version := range productVersions {
		if version.Name == m.Resource.MinVersion() {
			return ToPythonTpl(version.BaseUrl)
		}
	}

	return ""
}
