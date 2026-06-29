# Agent Context: ansible-mmv1

## What This Project Is

`ansible-mmv1` is a **Go code generator** that reads Google Cloud resource definitions
from [Magic Modules (MMv1)](https://github.com/GoogleCloudPlatform/magic-modules) and
produces:

- Python Ansible module files (`output/plugins/modules/<name>.py`)
- Integration test scaffolding (`output/tests/integration/targets/<name>/`)

It is the `ansible` compiler target in the Magic Modules ecosystem - analogous to how
MMv1 generates Terraform provider resources, this tool generates `google.cloud`
Ansible collection modules.

**This is not an Ansible playbook/role/collection repository.** It is a Go tool that
*generates* Ansible modules.

## Repository Layout

```
ansible-mmv1/
├── main.go                     # CLI entry point (cobra + viper); git clone + worker pool
├── config.yaml                 # Primary runtime config (pinned MMv1 commit, product list)
├── go.mod / go.sum             # Go 1.25 module; key deps: cobra, viper, zerolog, go-git
├── overlay/                    # Local YAML + template overrides layered over the MMv1 clone
│   ├── products/               # Per-product/resource YAML overrides (MMv1 layout)
│   │   ├── alloydb/
│   │   ├── cloudbuildv2/
│   │   ├── colab/
│   │   ├── tpuv2/
│   │   └── vertexai/
│   └── templates/ansible/examples/   # 73 Ansible YAML example templates (.tmpl)
├── pkg/
│   ├── api/                    # MMv1 product/resource loading + overlay FS wrappers
│   │   ├── api.go              # Product/Resource structs; AnsibleName() -> gcp_<prod>_<res>
│   │   └── loader.go           # LoadProducts(), OverlayFS, ansibleExampleRedirectFS
│   ├── ansible/                # Module building from MMv1 resource definitions
│   │   ├── module.go           # Module struct + constructor
│   │   ├── argspec.go          # argument_spec=dict(...) Python code generation
│   │   ├── documentation.go    # DOCUMENTATION block builder
│   │   ├── examples.go         # EXAMPLES block (doc vs test variants)
│   │   ├── returns.go          # RETURN block builder
│   │   ├── operation_config.go # CRUD operation configs (URI/verb/timeout)
│   │   ├── options.go          # MMv1 Type -> Ansible Option mapping
│   │   └── utils.go            # YAML serialization, description parsing, ToPythonTpl
│   └── renderer/               # Template execution engine
│       ├── renderer.go         # TemplateData, GenerateCode(), GenerateTests()
│       └── utils.go            # Template function map (indent, sortedKeys, etc.)
├── templates/                  # Go text/template files
│   ├── base/
│   │   ├── fragments.tmpl      # python_file_header, license_notice, autogen_notice
│   │   └── test_fragments.tmpl # network_setup / network_teardown fragments
│   ├── plugins/
│   │   ├── module.tmpl         # Main Python module template (~390 lines)
│   │   └── module_info.tmpl    # Info module variant
│   └── tests/integration/
│       ├── aliases.tmpl
│       ├── defaults/main.yml.tmpl
│       ├── meta/main.yml.tmpl
│       └── tasks/autogen.yml.tmpl
└── magic-modules/              # Gitignored: cloned at runtime from upstream
```

Gitignored paths: `output/` (generated files), `magic-modules/` (cloned repo),
`ansible-mmv1` (binary), `.env`.

## Key Concepts

### Overlay System

The `overlay/` directory mirrors the MMv1 directory layout. At load time, an
`OverlayFS` is constructed that layers overlay files *over* the upstream MMv1 clone.
This allows Ansible-specific customizations without touching upstream YAML.

Overlay YAML files support:
- `_drop: true` - remove an item from a list of maps
- Field merges - override specific fields in nested objects
- Smart matching - items matched by `name` or `id`
- List replacement - scalar lists are fully replaced

Overlay YAML also supports `custom_code` blocks with hooks: `pre_read`, `post_read`,
`pre_create`, `post_create`, `pre_update`, `pre_delete`, `post_delete`, `encoder`,
`decoder`, `custom_import`, `custom_create`, `custom_update`, `custom_delete`.

### Ansible Example Templates

For each MMv1 example, the loader looks for
`overlay/templates/ansible/examples/<name>.tmpl`. If not found, the loader logs a
warning and uses empty content. This redirect is handled transparently by
`ansibleExampleRedirectFS` in `pkg/api/loader.go`.

### Module Naming

Modules are named `gcp_<product>_<resource>` (all lowercase). For example,
`vertexai` + `Dataset` → `gcp_vertexai_dataset`.

### Generation Pipeline

1. `main.go`: parse config/flags → git clone/checkout MMv1 → `api.LoadProducts()`
2. `pkg/api`: build `OverlayFS` → call MMv1 loader with `CompilerTarget: "ansible"`
   → wrap products/resources → load Ansible example templates
3. `pkg/ansible`: `NewFromResource()` builds a `Module` struct (options, docs,
   examples, returns, operation configs, argspec)
4. `pkg/renderer`: render `module.tmpl` → write `.py`; render test templates → write
   integration test tree
5. Optionally run `black` (Python) and `yamlfmt` on generated files

### Worker Pool

Generation is parallelized with a goroutine pool capped at `min(NumCPU, 16)`.
Failures from any worker are captured and the first error is returned after all
workers finish.

## Products Currently Configured

The list of products is currently defined in `config.yaml` the table below should be kept up to date by the agent based on the config file

| Product | Resources |
|---|---|
| `alloydb` | Backup, Cluster, Instance, User |
| `cloudbuild` | Trigger *(tests skipped)* |
| `cloudbuildv2` | Connection, Repository *(tests skipped)* |
| `colab` | NotebookExecution, Runtime, RuntimeTemplate, Schedule |
| `vertexai` | Dataset, DeploymentResourcePool, Endpoint, EndpointWithModelGardenDeployment, FeatureGroup, FeatureGroupFeature, FeatureOnlineStore, FeatureOnlineStoreFeatureview, Featurestore, FeaturestoreEntitytype, FeaturestoreEntitytypeFeature, Index, IndexEndpoint, IndexEndpointDeployedIndex, MetadataStore, RagEngineConfig, ReasoningEngine, Tensorboard |

The upstream MMv1 commit is pinned in `config.yaml` under `git.rev`; `git.pull` is
`false` by default.

## Common Commands

```bash
# Build the binary
go build .

# Run with config file (most common during development)
go run . --config config.yaml

# Skip git clone if already cloned
go run . --config config.yaml --no-git-clone

# Generate a single product, skip formatting
go run . --products vertexai --no-format

# Generate a single resource
go run . --products alloydb --resources cluster

# Skip test generation
go run . --config config.yaml --no-tests

# Debug logging
go run . --config config.yaml --log-level debug

# Run tests
go test ./...
```

Formatters required at runtime: `black` (Python), `yamlfmt`. Use `--no-format` to
skip them if unavailable.

## Important Files for Context

When modifying generation logic, the most relevant files are:

- **`pkg/ansible/module.go`** and **`pkg/ansible/options.go`** - how MMv1 types map to
  Ansible module options
- **`pkg/api/loader.go`** - how the overlay FS and Ansible example redirect work
- **`templates/plugins/module.tmpl`** - the core generated Python module template
- **`overlay/products/<product>/<resource>.yaml`** - per-resource customizations
- **`config.yaml`** - which products/resources are generated and git settings

## Coding Conventions

- License header: Apache 2.0, copyright Red Hat Inc.
- All Go files start with `// Copyright 2025 Red Hat Inc.\n// SPDX-License-Identifier: Apache-2.0`
- Logging via `github.com/rs/zerolog`; use `log.Info()`, `log.Debug()`, `log.Warn()`,
  `log.Fatal()` - never `fmt.Println` for operational messages
- CLI via `cobra`; configuration via `viper` (config file + flags merged); flags bound
  to viper keys via `mustBindPFlag()`
- Errors are wrapped with `fmt.Errorf("context: %w", err)`
- Do not use Unicode em-dashes (`—`) anywhere in the codebase, comments, templates, or documentation (including this file); use a regular hyphen or dash (`-`) instead.
- Generated Python modules include a GPL v3 header + "AUTO GENERATED CODE" warning

## What Is Generated

Each generated Python module (`output/plugins/modules/<name>.py`):
- Defines `DOCUMENTATION`, `EXAMPLES`, `RETURN` Python string literals (YAML content)
- Imports `gcp_utils as gcp` from the `google.cloud` Ansible collection
- Generates nested `class` definitions (inheriting `gcp.Resource`) for nested objects
- Generates `encode()` / `decode()` hooks for custom transformation
- Has a `main()` dispatching create/update/delete with async LRO support

Standard module requirements injected into every `DOCUMENTATION` block:
- `python >= 3.8`
- `requests >= 2.18.4`
- `google-auth >= 2.25.1`

All modules extend the `google.cloud.gcp` documentation fragment.
