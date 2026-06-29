// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package ansible

import (
	"fmt"
	"strings"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
	"github.com/rs/zerolog/log"
	"github.com/thekad/ansible-mmv1/pkg/api"
)

// InfoModule is the top-level data structure passed to module_info.tmpl.
// It is constructed independently of Module - there is no shared state between
// the two and InfoModule carries no CustomCode, no nested options, no operation
// configs, and no CRUD logic.
type InfoModule struct {
	Name              string
	Resource          *api.Resource
	MinVersion        string
	DocumentationInfo *Documentation
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

// UrlParamOnlyProperties returns all url-param-only properties from the MMv1 resource,
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
