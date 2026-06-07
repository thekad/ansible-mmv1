// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package api

// AnsibleExamplesDir is the path example templates live, mirroring templates/terraform/examples.
const AnsibleExamplesDir = "templates/ansible/examples"

const (
	terraformExamplesDir      = "templates/terraform/examples/"
	terraformExampleSuffix    = ".tf.tmpl"
	ansibleExampleSuffix      = ".tmpl"
	compilerTargetAnsible     = "ansible"
	productsDirPrefix         = "products/"
	defaultResourceMinVersion = "ga"
)
