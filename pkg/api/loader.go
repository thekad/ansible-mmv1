// Copyright 2026 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strings"

	mmv1api "github.com/GoogleCloudPlatform/magic-modules/mmv1/api"
	"github.com/GoogleCloudPlatform/magic-modules/mmv1/google"
	mmv1loader "github.com/GoogleCloudPlatform/magic-modules/mmv1/loader"
	"github.com/rs/zerolog/log"
)

// ansibleExampleRedirectFS wraps the merged mmv1 FS so that reads of Terraform example
// templates (templates/terraform/examples/<name>.tf.tmpl) are satisfied from Ansible
// examples (templates/ansible/examples/<name>.tmpl) when present.
type ansibleExampleRedirectFS struct {
	inner google.ReadDirReadFileFS
}

func wrapAnsibleExampleRedirect(inner google.ReadDirReadFileFS) google.ReadDirReadFileFS {
	if inner == nil {
		return nil
	}
	return &ansibleExampleRedirectFS{inner: inner}
}

func (a *ansibleExampleRedirectFS) Open(name string) (fs.File, error) {
	return a.inner.Open(name)
}

func (a *ansibleExampleRedirectFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return a.inner.ReadDir(name)
}

func (a *ansibleExampleRedirectFS) ReadFile(name string) ([]byte, error) {
	if alt, ok := terraformExamplesToAnsible(name); ok {
		b, err := a.inner.ReadFile(alt)
		if err == nil {
			return b, nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			// Do not fall back to the Terraform template: TF templates reference
			// `$.Vars` / `$.ResourceIdVars` keys that are not guaranteed to align
			// with how Ansible samples/examples are structured, and reading them
			// can trigger spurious "variable not defined" fatal errors upstream.
			log.Warn().Str("terraform_path", name).Str("ansible_path", alt).
				Msg("example template not found; using empty content")
			return []byte{}, nil
		}
		return nil, err
	}

	if isAnsibleExampleTemplatePath(name) {
		b, err := a.inner.ReadFile(name)
		if err == nil {
			return b, nil
		}
		if errors.Is(err, fs.ErrNotExist) {
			log.Warn().Str("ansible_path", name).
				Msg("example template not found; using empty content")
			return []byte{}, nil
		}
		return nil, err
	}

	return a.inner.ReadFile(name)
}

func isAnsibleExampleTemplatePath(name string) bool {
	examplesMatch := strings.HasPrefix(name, AnsibleExamplesDir+"/") && strings.HasSuffix(name, ansibleExampleSuffix)
	samplesMatch := strings.HasPrefix(name, AnsibleSamplesDir+"/") && strings.HasSuffix(name, ansibleExampleSuffix)
	return examplesMatch || samplesMatch
}

func terraformExamplesToAnsible(terraformPath string) (ansiblePath string, ok bool) {
	// Handle legacy examples: templates/terraform/examples/<name>.tf.tmpl -> templates/ansible/examples/<name>.tmpl
	if strings.HasPrefix(terraformPath, terraformExamplesDir) {
		rest := strings.TrimPrefix(terraformPath, terraformExamplesDir)
		if strings.HasSuffix(rest, terraformExampleSuffix) {
			stem := strings.TrimSuffix(rest, terraformExampleSuffix)
			if stem != "" {
				return path.Join(AnsibleExamplesDir, stem+ansibleExampleSuffix), true
			}
		}
		return "", false
	}
	// Handle new samples: templates/terraform/samples/services/<pkg>/<name>.tf.tmpl -> templates/ansible/samples/services/<pkg>/<name>.tmpl
	if strings.HasPrefix(terraformPath, terraformSamplesDir) {
		rest := strings.TrimPrefix(terraformPath, terraformSamplesDir)
		if strings.HasSuffix(rest, terraformExampleSuffix) {
			stem := strings.TrimSuffix(rest, terraformExampleSuffix)
			// stem is "services/<pkg>/<name>" - preserve the full relative path
			if stem != "" && stem != "." {
				return path.Join(AnsibleSamplesDir, stem+ansibleExampleSuffix), true
			}
		}
		return "", false
	}
	return "", false
}

// LoadProducts loads Magic Modules products. If onlyProductShortNames is non-empty, only
// those products are loaded via LoadProduct; otherwise LoadProducts() loads the
// full catalog. An empty slice means “all products”.
//
// The wrapped FS redirects Terraform example paths to Ansible templates when present
func LoadProducts(mmv1Root, overlayDir, version string, productFilter []string) (google.ReadDirReadFileFS, *mmv1loader.Loader, error) {
	ofs, err := google.NewOverlayFS(overlayDir, mmv1Root)
	if err != nil {
		return nil, nil, err
	}
	ansibleSysFS := wrapAnsibleExampleRedirect(ofs)

	l := mmv1loader.NewLoader(mmv1loader.Config{
		Version:           version,
		BaseDirectory:     mmv1Root,
		OverrideDirectory: overlayDir,
		Sysfs:             ansibleSysFS,
		CompilerTarget:    compilerTargetAnsible,
	})

	if len(productFilter) == 0 {
		l.LoadProducts()
	} else {
		seen := make(map[string]struct{})
		products := make(map[string]*mmv1api.Product)
		for _, raw := range productFilter {
			short := strings.ToLower(strings.TrimSpace(raw))
			if short == "" {
				continue
			}
			if _, dup := seen[short]; dup {
				continue
			}
			seen[short] = struct{}{}

			key := productsDirPrefix + short
			p, err := l.LoadProduct(key)
			if err != nil {
				var verr *mmv1loader.ErrProductVersionNotFound
				if errors.As(err, &verr) {
					log.Warn().Err(err).Str("product", key).Msg("skipping product (version not available)")
					continue
				}
				log.Warn().Err(err).Str("product", key).Msg("failed to load product; skipping")
				continue
			}
			products[key] = p
		}
		if len(products) == 0 {
			return nil, nil, fmt.Errorf("no products could be loaded for requested names %v", productFilter)
		}
		l.Products = products
		log.Info().Strs("products", productFilter).Msg("loaded selected products only")
	}

	if err := l.AddExtraFields(); err != nil {
		return nil, nil, err
	}
	l.Validate()
	return ansibleSysFS, l, nil
}

// ProductKeys returns sorted loader map keys (e.g. "products/cloudbuildv2").
func ProductKeys(l *mmv1loader.Loader) []string {
	keys := make([]string, 0, len(l.Products))
	for k := range l.Products {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ShortName returns the directory segment under products/ for a loader product key.
func ShortName(productKey string) string {
	return strings.ToLower(filepath.Base(productKey))
}

// WrapProduct builds ansible-mmv1's api.Product from a loaded MMv1 product.
func WrapProduct(mmProd *mmv1api.Product, mmRoot string) *Product {
	short := ShortName(mmProd.PackagePath)
	return &Product{
		Name: short,
		File: filepath.Join(mmRoot, mmProd.PackagePath, "product.yaml"),
		Mmv1: mmProd,
	}
}

// WrapResource builds ansible-mmv1's api.Resource.
func WrapResource(mmRes *mmv1api.Resource, parent *Product, mmRoot string) *Resource {
	return &Resource{
		Name:   mmRes.Name,
		File:   filepath.Join(mmRoot, mmRes.SourceYamlFile),
		Mmv1:   mmRes,
		Parent: parent,
	}
}

