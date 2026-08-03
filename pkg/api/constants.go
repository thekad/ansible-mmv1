// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package api

// AnsibleSamplesDir is the root path for all Ansible sample templates.
// Resource-specific templates live under services/<pkg>/; shared/common templates
// live under common/. Both are served directly from the overlay FS via
// isAnsibleExampleTemplatePath without any terraform-path redirection.
const AnsibleSamplesDir = "templates/ansible/samples"

const (
	terraformSamplesDir       = "templates/terraform/samples/"
	terraformExampleSuffix    = ".tf.tmpl"
	ansibleExampleSuffix      = ".tmpl"
	compilerTargetAnsible     = "ansible"
	productsDirPrefix         = "products/"
	defaultResourceMinVersion = "ga"
)
