// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package api

// AnsibleExamplesDir is the path legacy example templates live, mirroring templates/terraform/examples.
const AnsibleExamplesDir = "templates/ansible/examples"

// AnsibleSamplesDir is the path sample templates live, mirroring templates/terraform/samples.
const AnsibleSamplesDir = "templates/ansible/samples"

const (
	terraformExamplesDir      = "templates/terraform/examples/"
	terraformSamplesDir       = "templates/terraform/samples/"
	terraformExampleSuffix    = ".tf.tmpl"
	ansibleExampleSuffix      = ".tmpl"
	compilerTargetAnsible     = "ansible"
	productsDirPrefix         = "products/"
	defaultResourceMinVersion = "ga"
)
