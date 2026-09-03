// Copyright 2025 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0

package renderer

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
	"github.com/thekad/ansible-mmv1/pkg/ansible"
)

type TemplateData struct {
	templateFS               fs.FS
	OutputFolder             string
	ModuleDirectory          string
	IntegrationTestDirectory string
	OverWrite                bool
}

func NewTemplateData(templateFS fs.FS, outputFolder string, overWrite bool) *TemplateData {
	absOutputFolder, err := filepath.Abs(outputFolder)
	if err != nil {
		log.Panic().Err(err)
	}
	return &TemplateData{
		templateFS:               templateFS,
		OutputFolder:             absOutputFolder,
		ModuleDirectory:          path.Join(absOutputFolder, "plugins", "modules"),
		IntegrationTestDirectory: path.Join(absOutputFolder, "tests", "integration", "targets"),
		OverWrite:                overWrite,
	}
}

func (td *TemplateData) executeTemplate(templateName string, input any) (bytes.Buffer, error) {
	contents := bytes.Buffer{}

	tpls := []string{
		"base/fragments.tmpl",
		templateName,
	}

	// Create template first
	tmpl := template.New(filepath.Base(templateName))

	// Create function map with special function exec that will have access to the template object
	funcs := funcMap(td.templateFS)
	funcs["exec"] = func(templateName string, data interface{}) (string, error) {
		var buf strings.Builder
		err := tmpl.ExecuteTemplate(&buf, templateName, data)
		if err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	// Add the function map to the template and then parse files
	tmpl = tmpl.Funcs(funcs)
	tmpl, err := tmpl.ParseFS(td.templateFS, tpls...)
	if err != nil {
		return contents, err
	}

	if err = tmpl.ExecuteTemplate(&contents, filepath.Base(templateName), input); err != nil {
		return contents, err
	}

	return contents, nil
}

func (td *TemplateData) writeFile(filePath, templateName string, input any) error {
	log.Debug().Msgf("writing file: %s", filePath)
	if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	if fileExists(filePath) {
		if !td.OverWrite {
			return fmt.Errorf("file already exists: %s", filePath)
		} else {
			log.Warn().Msgf("overwriting file: %s", filePath)
		}
	}

	contents, err := td.executeTemplate(templateName, input)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filePath, contents.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}

func (td *TemplateData) GenerateInfoCode(module *ansible.InfoModule) error {
	if err := os.MkdirAll(td.ModuleDirectory, 0755); err != nil {
		return fmt.Errorf("error creating info module directory: %v", err)
	}

	moduleFile := path.Join(td.ModuleDirectory, fmt.Sprintf("%s.py", module))

	if err := td.writeFile(moduleFile, "plugins/module_info.tmpl", module); err != nil {
		return fmt.Errorf("error generating info module file: %v", err)
	}

	return nil
}

func (td *TemplateData) GenerateCode(module *ansible.Module) error {
	if err := os.MkdirAll(td.ModuleDirectory, 0755); err != nil {
		return fmt.Errorf("error creating module directory: %v", err)
	}

	moduleFile := path.Join(td.ModuleDirectory, fmt.Sprintf("%s.py", module))

	if err := td.writeFile(moduleFile, "plugins/module.tmpl", module); err != nil {
		return fmt.Errorf("error generating module file: %v", err)
	}

	return nil
}

func (td *TemplateData) GenerateTests(module *ansible.Module) error {
	directories := []string{
		td.IntegrationTestDirectory,
		path.Join(td.IntegrationTestDirectory, module.Name),
		path.Join(td.IntegrationTestDirectory, module.Name, "defaults"),
		path.Join(td.IntegrationTestDirectory, module.Name, "meta"),
		path.Join(td.IntegrationTestDirectory, module.Name, "tasks"),
	}
	for _, directory := range directories {
		log.Debug().Msgf("creating integration test directory: %s", directory)
		if err := os.MkdirAll(directory, 0755); err != nil {
			return fmt.Errorf("error creating integration test directory: %v", err)
		}
	}

	testFiles := []string{
		path.Join(td.IntegrationTestDirectory, module.Name, "aliases"),
		path.Join(td.IntegrationTestDirectory, module.Name, "defaults", "main.yml"),
		path.Join(td.IntegrationTestDirectory, module.Name, "meta", "main.yml"),
		path.Join(td.IntegrationTestDirectory, module.Name, "tasks", "autogen.yml"),
	}
	for _, testFile := range testFiles {
		log.Debug().Msgf("creating integration test file: %s", testFile)
		templateName := path.Join("tests/integration", strings.TrimPrefix(testFile, filepath.Join(td.IntegrationTestDirectory, module.Name))+".tmpl")
		if err := td.writeFile(testFile, templateName, module); err != nil {
			return fmt.Errorf("error creating integration test file: %v", err)
		}
	}

	return nil
}

// fileExists returns true if the given path exists, false otherwise
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
