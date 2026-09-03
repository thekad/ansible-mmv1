# Agent Context: ansible-mmv1

## What This Project Is

`ansible-mmv1` is a **Go code generator** that reads Google Cloud resource definitions
from [Magic Modules (MMv1)](https://github.com/GoogleCloudPlatform/magic-modules) and
produces:

- Python Ansible module files (`output/plugins/modules/<name>.py`)
- Integration test scaffolding (`output/tests/integration/targets/<name>/`)

It is the `ansible` compiler target in the Magic Modules ecosystem - analogous to how
MMv1 generates Terraform provider resources, this tool generates Ansible collection
modules for whichever collection points its `--templates`/`--overlay` (or `templates`/
`overlay` config keys) at this engine.

**This is not an Ansible playbook/role/collection repository.** It is a standalone Go
engine that *generates* Ansible modules. See ADR 0003
(`docs/adr/0003-decouple-templates-from-engine.md`) for the decision to decouple the
engine from any single collection's content.

**Two-repo model:** the production `templates/` and `overlay/` content (Google
branding, `gcp_v2` module_utils imports, per-resource YAML overrides, Ansible sample
templates) now lives in the target `google.cloud` collection repository, not here.
This repo ships only:
- The Go engine (`pkg/`, `main.go`)
- A generic, `FIXME`-marked **skeleton** `templates/` directory (embedded into the
  binary via `//go:embed`) that documents the template contract and lets
  `go run github.com/thekad/ansible-mmv1@latest` produce *something* even with no
  external templates/overlay configured
- No `overlay/` directory at all (an overlay is optional; if none is found, only
  upstream MMv1 YAML is used)

When developing against real Google Cloud resources, point `--templates`/`--overlay`
(or the `templates`/`overlay` config keys) at a checkout of the collection repo that
holds the production content.

## Repository Layout

```
ansible-mmv1/
├── main.go                     # CLI entry point (cobra + viper); git clone + worker pool; go:embed skeleton templates
├── mmv1-config.yaml            # Primary runtime config (pinned MMv1 commit, product list)
├── go.mod / go.sum             # Go 1.26 module; key deps: cobra, viper, zerolog, go-git
├── docs/adr/                   # Architecture Decision Records (design decisions, migration plans)
├── pkg/
│   ├── api/                    # MMv1 product/resource loading + overlay FS wrappers
│   │   ├── api.go              # Product/Resource structs; AnsibleName() -> gcp_<prod>_<res>
│   │   └── loader.go           # LoadProducts(), OverlayFS, ansibleExampleRedirectFS
│   ├── ansible/                # Module building from MMv1 resource definitions
│   │   ├── module.go           # Module struct + constructor
│   │   ├── infomodule.go       # InfoModule struct + constructor; loads overlay/info/ customizations
│   │   ├── argspec.go          # argument_spec=dict(...) Python code generation
│   │   ├── documentation.go    # DOCUMENTATION block builder
│   │   ├── examples.go         # EXAMPLES block (doc vs test variants)
│   │   ├── returns.go          # RETURN block builder
│   │   ├── operation_config.go # CRUD operation configs (URI/verb/timeout)
│   │   ├── options.go          # MMv1 Type -> Ansible Option mapping
│   │   └── utils.go            # YAML serialization, description parsing, ToPythonTpl
│   └── renderer/               # Template execution engine
│       ├── renderer.go         # TemplateData (fs.FS-backed), GenerateCode(), GenerateTests()
│       └── utils.go            # Template function map (indent, sortedKeys, etc.)
├── templates/                  # Skeleton Go text/template files (embedded via go:embed; FIXME-marked)
│   ├── base/
│   │   └── fragments.tmpl      # python_file_header, license_notice, autogen_notice
│   ├── plugins/
│   │   ├── module.tmpl         # Skeleton Python module template
│   │   └── module_info.tmpl    # Skeleton info module variant
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

**Note:** this repo does not ship an `overlay/` directory (see ADR 0003). The
production `overlay/` content lives in the target `google.cloud` collection repo;
point `--overlay` (or the `overlay` config key) at a checkout of that repo during
development. If no overlay directory is found, generation proceeds using upstream
MMv1 YAML only - this section documents the overlay *mechanism*, which applies to
whichever overlay directory is configured.

Overlay YAML files support:
- `_drop: true` - a sentinel value (not `true` as a boolean flag) used to blank a
  *scalar* `custom_code` string field (e.g. `encoder: _drop`); checked at
  template-render time via `strEq ... "_drop"`. It does **not** remove items from
  list-of-maps fields such as `examples:`/`samples:`/`properties:` - there is no
  mechanism to delete a list item once declared upstream. See
  `docs/adr/0002-examples-to-samples-migration.md` for the full investigation.
- Field merges - override specific fields in nested objects
- Smart matching - list-of-maps items (e.g. `examples:`, `samples:`, `properties:`)
  are matched against the base by their `name` field; a match gets its fields
  merged in, a non-match gets appended as a new item
- List replacement - scalar lists (e.g. `[]string` fields like `scopes`) are fully
  replaced

Overlay YAML also supports `custom_code` blocks with hooks: `pre_read`, `post_read`,
`pre_create`, `post_create`, `pre_update`, `pre_delete`, `post_delete`, `encoder`,
`decoder`, `custom_import`, `custom_create`, `custom_update`, `custom_delete`.

### Ansible Sample Templates

All Ansible-specific sample content is declared through each resource's own
`overlay/products/<product>/<Resource>.yaml` `samples:` block using an `ansible_`-
prefixed name that can never collide with anything upstream declares. This makes our
overlay content immune to whichever key (`examples:` vs `samples:`) a given upstream
resource happens to use. See `docs/adr/0002-examples-to-samples-migration.md`.

**Naming and placement rules - follow these for all new content:**

- Every sample and step name we declare is prefixed with `ansible_`. Names are
  never rendered into output, but the prefix guarantees no collision with
  upstream.
- After the `ansible_` prefix, names carry a **phase prefix** that classifies the
  step, followed by the resource and variant (the **product name is stripped** -
  the containing directory already encodes it):
  - `ansible_doc_<resource>[_<variant>]` - documentation-only examples rendered
    into the module's `EXAMPLES` block (e.g. `ansible_doc_backup`,
    `ansible_doc_trigger_filename`).
  - `ansible_setup_<name>` - prerequisite steps run before the main test, emitted
    into the test `block:` preamble (e.g. `ansible_setup_cluster`).
  - `ansible_test_<resource>` - the main integration-test body + assertions
    (e.g. `ansible_test_backup`).
  - `ansible_teardown_<name>` - cleanup steps emitted into the test `always:`
    section (e.g. `ansible_teardown_cluster`).

  `pkg/ansible/examples.go`'s `ToString(phase)` partitions steps by these
  prefixes. Steps with an unrecognized prefix default to the `test` phase for
  backward compatibility. In `doc` mode, phase filtering is skipped (all steps of
  the doc samples render).
- **Doc-only products** - those with `skip-tests: ["*"]` in `mmv1-config.yaml`
  (currently `cloudbuild` and `cloudbuildv2`) generate no integration tests, so
  they use **only** the `ansible_doc_` phase. The `setup`/`test`/`teardown` phases
  apply only where integration tests actually run (e.g. `alloydb`).
- Always use `samples:` in overlay YAML. Never use `examples:`. The `examples:` key
  is a legacy MMv1 concept; our overlay no longer uses it.
- Resource-specific templates live at
  `overlay/templates/ansible/samples/services/<pkg>/ansible_<name>.tmpl`.
- Shared/reusable step templates (e.g. network + VPC peering setup reused across
  multiple resources) live at
  `overlay/templates/ansible/samples/common/ansible_<name>.tmpl` and are referenced
  via an explicit `config_path` on the step:
  ```yaml
  steps:
    - name: ansible_setup_network
      config_path: templates/ansible/samples/common/ansible_setup_network.tmpl
    - name: ansible_test_my_resource
  ```
- Simple (doc-only) samples have one step sharing the sample's name. Test samples
  have multiple steps: shared/setup steps first, the resource's own `ansible_test_`
  step in the middle, teardown steps last. `Sample.Name` matches the resource's own
  step name.
- Sample/step templates are always written **flush-left** (no leading indentation);
  the renderer handles indentation into the generated `block:`/`always:` structure.
- Parameterized steps can pass values into a shared template via `vars:` on the
  step, referenced in the template as `{{index $.Vars "name"}}`. **Always use the
  no-space form** `{{index $.Vars "name"}}` - MMv1's `validateRegexForContents`
  (`magic-modules/mmv1/api/resource/step.go`) matches that exact spelling to enforce
  "every referenced var must be declared on the step". The spaced form
  `{{ index $.Vars "name" }}` still renders but **silently bypasses** that check, so
  a missing `vars:` declaration goes undetected and produces broken output. During
  test rendering MMv1 rewrites each declared var to a `%{name}` placeholder;
  `pkg/ansible/examples.go`'s `substituteTestVars` converts those back to the literal
  value.

`ansibleExampleRedirectFS` in `pkg/api/loader.go` handles two path patterns:

- Auto-derived terraform paths (`templates/terraform/samples/services/<pkg>/<name>.tf.tmpl`)
  are redirected to `overlay/templates/ansible/samples/services/<pkg>/<name>.tmpl`.
- Paths that already point into `templates/ansible/samples/` (explicit `config_path`
  values, including `common/` shared templates) are read directly from the overlay FS
  without redirection.

If a template is not found, the loader logs a warning and uses **empty** content
(no fallback to the raw Terraform template). Empty content is filtered out
downstream by `pkg/ansible/examples.go`'s `ToString()`.

### Module Naming

Modules are named `gcp_<product>_<resource>` (all lowercase). For example,
`vertexai` + `Dataset` → `gcp_vertexai_dataset`.

### Generation Pipeline

1. `main.go`: parse config/flags → git clone/checkout MMv1 → `api.LoadProducts()`
2. `pkg/api`: build `OverlayFS` → call MMv1 loader with `CompilerTarget: "ansible"`
   → wrap products/resources → load Ansible example templates
3. `pkg/ansible`: `NewFromResource()` builds a `Module` struct (options, docs,
   examples, returns, operation configs, argspec); `NewInfoFromResource()` builds an
   `InfoModule` struct, optionally loading a customization file from `overlay/info/`
4. `pkg/renderer`: render `module.tmpl` → write `.py`; render `module_info.tmpl` →
   write `_info.py`; render test templates → write integration test tree
5. Optionally run `black` (Python) and `yamlfmt` on generated files

### Worker Pool

Generation is parallelized with a goroutine pool capped at `min(NumCPU, 16)`.
Failures from any worker are captured and the first error is returned after all
workers finish.

## Products Currently Configured

The list of products is currently defined in `mmv1-config.yaml` the table below should be kept up to date by the agent based on the config file

| Product | Resources |
|---|---|
| `alloydb` | Backup, Cluster, Instance, User |
| `cloudbuild` | Trigger *(tests skipped, info skipped)* |
| `cloudbuildv2` | Connection, Repository *(tests skipped)* |
| `colab` | NotebookExecution, Runtime, RuntimeTemplate, Schedule |
| `vertexai` | Dataset, DeploymentResourcePool, Endpoint, EndpointWithModelGardenDeployment *(info skipped)*, FeatureGroup, FeatureGroupFeature, FeatureOnlineStore, FeatureOnlineStoreFeatureview, Featurestore, FeaturestoreEntitytype, FeaturestoreEntitytypeFeature, Index, IndexEndpoint, IndexEndpointDeployedIndex *(info skipped)*, MetadataStore, RagEngineConfig *(info skipped)*, ReasoningEngine, Tensorboard |

The upstream MMv1 commit is pinned in `mmv1-config.yaml` under `git.rev`; `git.pull` is
`false` by default.

## Common Commands

```bash
# Build the binary
go build .

# Run with config file (most common during development)
go run . --config mmv1-config.yaml

# Skip git clone if already cloned
go run . --config mmv1-config.yaml --no-git-clone

# Generate a single product, skip formatting
go run . --products vertexai --no-format

# Generate a single resource
go run . --products alloydb --resources cluster

# Skip test generation
go run . --config mmv1-config.yaml --no-tests

# Debug logging
go run . --config mmv1-config.yaml --log-level debug

# Run tests
go test ./...
```

Formatters required at runtime: `black` (Python), `yamlfmt`. Use `--no-format` to
skip them if unavailable.

## Important Files for Context

When modifying generation logic, the most relevant files are:

- **`pkg/ansible/module.go`** and **`pkg/ansible/options.go`** - how MMv1 types map to
  Ansible module options
- **`pkg/ansible/infomodule.go`** - how info modules are built and how customization
  files are loaded
- **`pkg/api/loader.go`** - how the overlay FS and Ansible example redirect work
- **`templates/plugins/module.tmpl`** - the core generated Python module template
- **`templates/plugins/module_info.tmpl`** - the generated info module template
- **`overlay/products/<product>/<resource>.yaml`** - per-resource customizations
- **`overlay/info/<product>/<resource>.yaml`** - per-resource info module customization
  files (NOT processed by MMv1; contains `custom_code` with `pre_read`, `post_read`,
  and `custom_import` hooks injected into the info module's `main()`)
- **`mmv1-config.yaml`** - which products/resources are generated and git settings

## Architecture Decision Records

Significant design decisions and migration plans are recorded as ADRs under
`docs/adr/`, with an index and status legend in `docs/adr/README.md`. Consult
these before undertaking any major refactor of the overlay system, the MMv1
dependency version, or the examples/samples template layout - they capture
non-obvious mechanics (e.g. exactly how MMv1's YAML merge behaves) that would
otherwise need to be re-derived from the vendored `magic-modules` source each time.

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

Each generated info module (`output/plugins/modules/<name>_info.py`):
- Lists resources via the collection URL using the `list` API
- Accepts a `filters` parameter for server-side filtering
- May inject `pre_read` / `post_read` / `custom_import` blocks from
  `overlay/info/<product>/<resource>.yaml` if a customization file exists
- Has a `main()` that calls `info.list()` and returns results unchanged (unless a
  `post_read` block modifies them)

Standard module requirements injected into every `DOCUMENTATION` block:
- `python >= 3.8`
- `requests >= 2.18.4`
- `google-auth >= 2.25.1`

All modules extend the `google.cloud.gcp` documentation fragment.
