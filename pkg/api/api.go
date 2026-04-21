// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
)

// Product is a representation of a directory in the mmv1/products directory
// from the magic-modules clone e.g. mmv1/products/<product>/product.yaml
type Product struct {
	Name string
	File string
	Mmv1 *mmv1api.Product
}

// AnsibleName will return a properly formatted Ansible name for the given product
func (p *Product) AnsibleName() string {
	return fmt.Sprintf("gcp_%s", google.Underscore(p.Name))
}

// Resource is a representation of a file found in the products directory
// from magic-modules clone e.g. mmv1/products/<product>/<resource>.yaml
type Resource struct {
	Name   string
	File   string
	Mmv1   *mmv1api.Resource
	Parent *Product
}

func (r *Resource) AnsibleName() string {
	return fmt.Sprintf("%s_%s", r.Parent.AnsibleName(), google.Underscore(r.Name))
}

// MinVersion will return the minimum version supported by the given resource
func (r *Resource) MinVersion() string {
	if r.Mmv1.MinVersion == "" {
		return "ga"
	}
	return r.Mmv1.MinVersion
}

func (r *Resource) Versions() []string {
	versions := []string{}
	for _, version := range r.Parent.Mmv1.Versions {
		versions = append(versions, version.Name)
	}
	return versions
}
