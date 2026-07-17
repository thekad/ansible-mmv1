# ADR 0002: Migrate Ansible Overlay Content from `examples:` to `samples:`

**Status:** Proposed (2026-07-16)

## Motivation

Every time the pinned upstream `magic-modules` commit is bumped, any resource that
migrates from the legacy `examples:` YAML key to the native `samples:` key changes
which physical directory our Ansible-specific override templates must live in
(`overlay/templates/ansible/examples/` vs
`overlay/templates/ansible/samples/services/<pkg>/`), because our redirect logic in
`pkg/api/loader.go` derives the target path from the Terraform `ConfigPath` MMv1
computes, which differs by format. This has already required two full
reclassification passes in this repo's history (see
`docs/adr/0001-mmv1-upgrade-plan.md`), and will keep recurring indefinitely as upstream
continues its own multi-year `examples:` -> `samples:` migration, resource by resource.

This document describes how to make `ansible-mmv1`'s own overlay content
**immune** to upstream's format choice, permanently, by declaring 100% of our
Ansible-specific sample content through our own `samples:` overlay blocks, using a
name prefix that is structurally guaranteed to never collide with anything upstream
declares. Once complete, this repo never needs to inspect or care whether a given
upstream resource uses `examples:` or `samples:` again.

## Background: why we can't just delete/blank upstream's declarations

Investigated the actual MMv1 merge implementation
(`magic-modules/mmv1/api/product.go`, functions `Merge`/`DeepMerge`):

- `examples:`/`samples:` are lists of structs, merged by matching on the `Name`
  field. An override item whose name matches a base item gets field-merged into it;
  an override item with a new name gets **appended**. There is no delete/remove
  path anywhere in `DeepMerge`.
- The `_drop: true` convention mentioned in `AGENTS.md` **does not apply to list
  items**. It is a template-render-time sentinel (`strEq $.CustomCode.X "_drop"` in
  `templates/plugins/module.tmpl`/`module_info.tmpl`) that only works on scalar
  `custom_code` string fields (e.g. `encoder: _drop`). Using it on a list item
  would fail YAML's strict `KnownFields(true)` decoding.
- `examples: []`/`examples: null` in an override is indistinguishable from the Go
  zero-value and is skipped entirely by `Merge`'s emptiness check - a no-op.
- Full list replacement only happens automatically for **scalar** lists (e.g.
  `[]string` fields like `scopes`), or trivially when the base list is empty.
  Neither applies to `examples:`/`samples:`, which are lists of structs.

**Conclusion:** we cannot make upstream's own declared examples/samples disappear.
But we don't need to - see the next section.

## The mechanism: parallel, uniquely-named `samples:` entries

Our own overlay's `samples:` block and upstream's native `samples:` block are
merged as one combined list (by `Name`). If we declare our own sample with a name
that will *never* collide with anything upstream would plausibly use (guaranteed
via a distinctive prefix), our entry is simply **appended** alongside upstream's,
with zero interaction between the two:

- Our sample's step gets a `ConfigPath` default of
  `templates/terraform/samples/services/<pkg>/<our-step-name>.tf.tmpl`, which our
  redirect maps to `overlay/templates/ansible/samples/services/<pkg>/<our-step-name>.tmpl`.
  Since we provide that file, it renders with our real content.
- Upstream's own native sample's step still defaults to its own
  `.../<their-name>.tf.tmpl` path. Since we never provide a matching override file
  for *their* name, our redirect FS returns empty content for it (current,
  reverted-to behavior - no error, no placeholder text). Empty content is already
  filtered out by the existing `len(content) <= 1` check in
  `pkg/ansible/examples.go`'s `ToString()` ("skipping empty sample").

Net effect: upstream's declaration exists in the in-memory data model but is
functionally invisible in generated output - the equivalent of "blanking" it,
achieved without any deletion capability. This also works identically for
resources where upstream still uses the legacy `examples:` key today (the
`DeepMerge` early-exit `if arr1.Len() == 0 { arr1.Set(arr2); return }` means our
`samples:` override becomes the *entire* samples list outright when upstream's own
native `samples:` list for that resource is empty, which is the case for every
resource still on `examples:` upstream).

Confirmed this is 100% mechanically safe:
- `Sample.Validate()`/`Step.Validate()` only require a non-empty `Name` on each -
  no other required fields.
- YAML strict decoding operates per-file; override files are always expected to be
  partial, this is normal.
- Sample/step names are **never rendered into generated output** - verified no
  template (`templates/plugins/module.tmpl`, `templates/tests/integration/tasks/autogen.yml.tmpl`)
  references `.Name` on an Examples/Sample/Step object. All visible "name:" fields
  in generated docs/tests come from literal content inside the `.tmpl` files
  themselves. This means the migration is purely organizational and must produce
  **zero difference** in generated output.

## Naming convention

Every sample/step we declare through our own overlay is prefixed with `ansible_`,
otherwise preserving the existing descriptive name unchanged:

```
alloydb_backup_basic        -> ansible_alloydb_backup_basic
vertex_ai_dataset            -> ansible_vertex_ai_dataset
cloudbuild_trigger_filename  -> ansible_cloudbuild_trigger_filename
```

Rules:
- One sample = one step, sharing the identical name (no multi-step Ansible tests
  today; if that ever changes, revisit this convention).
- Template filenames match the new name exactly, e.g. `ansible_vertex_ai_dataset.tmpl`.
- Templates live under `overlay/templates/ansible/samples/services/<pkg>/`, where
  `<pkg>` is the lowercase product directory name (matches MMv1's own
  `packageName := filepath.Base(filepath.Dir(r.SourceYamlFile))` derivation).
- The legacy `overlay/templates/ansible/examples/` directory is retired entirely
  once migration is complete.

## Field mapping

Audited every one of the 25 resource YAML files with an existing overlay
`examples:` block; **100% of them only use `name`, `exclude_test`, and
`exclude_docs`** - no usage anywhere of `vars`, `resource_id_vars`,
`test_vars_overrides`, `test_env_vars`, `primary_resource_id`, `min_version`, or
`bootstrap_iam`. The mapping is therefore trivial:

| `Examples` (legacy) field | `Sample`/`Step` (native) field |
|---|---|
| `name` | `Sample.Name` **and** `Sample.Steps[0].Name` (identical value) |
| `exclude_test` | `Sample.ExcludeTest` (same field name) |
| `exclude_docs` | `Sample.ExcludeBasicDoc` (renamed) |

Example transformation:
```yaml
# before
examples:
  - name: "alloydb_backup_basic"
    exclude_test: true
  - name: "alloydb_backup_basic_test"
    exclude_docs: true
```
```yaml
# after
samples:
  - name: ansible_alloydb_backup_basic
    exclude_test: true
    steps:
      - name: ansible_alloydb_backup_basic
  - name: ansible_alloydb_backup_basic_test
    exclude_basic_doc: true
    steps:
      - name: ansible_alloydb_backup_basic_test
```

## Complete per-resource scope

**25 resources - pure rename** of existing overlay `examples:` entries into
equivalent `samples:` entries (no new entries needed, every currently-authored
`.tmpl` file already has a shadowing overlay declaration):

`alloydb/Backup`, `alloydb/Cluster`, `alloydb/Instance`, `alloydb/User`,
`colab/NotebookExecution`, `colab/Runtime`, `colab/RuntimeTemplate`,
`colab/Schedule`, `vertexai/DeploymentResourcePool`, `vertexai/FeatureGroupFeature`,
`vertexai/FeatureGroup`, `vertexai/FeatureOnlineStoreFeatureview`,
`vertexai/FeatureOnlineStore`, `vertexai/FeaturestoreEntitytypeFeature`,
`vertexai/FeaturestoreEntitytype`, `vertexai/Featurestore`,
`vertexai/IndexEndpointDeployedIndex`, `vertexai/Index`, `vertexai/ReasoningEngine`,
`vertexai/Tensorboard`

**6 resources - mixed** (rename existing shadowed entries, add new entries for
files that currently rely on matching upstream's native name directly):

| Resource | Existing (rename) | New (add) |
|---|---|---|
| `cloudbuildv2/Connection` | `cloudbuildv2_connection_gitlab` | `cloudbuildv2_connection_ghe`, `cloudbuildv2_connection_github` |
| `vertexai/EndpointWithModelGardenDeployment` | `vertex_ai_deploy_test` | `vertex_ai_deploy_basic`, `vertex_ai_deploy_huggingface_model`, `vertex_ai_deploy_psc_endpoint` |
| `vertexai/Endpoint` | `vertex_ai_endpoint_network_test` | `vertex_ai_endpoint_network` |
| `vertexai/IndexEndpoint` | `vertex_ai_index_endpoint_with_public_endpoint` | `vertex_ai_index_endpoint`, `vertex_ai_index_endpoint_test` |
| `vertexai/MetadataStore` | `vertex_ai_metadata_store_test` | `vertex_ai_metadata_store` |

**4 resources - brand new `samples:` block** (currently zero overlay override;
all their templates work today purely by matching upstream's native example name
directly):

| Resource | New entries |
|---|---|
| `cloudbuild/Trigger` | `cloudbuild_trigger_build`, `cloudbuild_trigger_filename`, `cloudbuild_trigger_manual`, `cloudbuild_trigger_pubsub_config`, `cloudbuild_trigger_repo`, `cloudbuild_trigger_webhook_config` |
| `cloudbuildv2/Repository` | `cloudbuildv2_repository_ghe`, `cloudbuildv2_repository_ghe_doc`, `cloudbuildv2_repository_github`, `cloudbuildv2_repository_github_doc`, `cloudbuildv2_repository_gle` |
| `vertexai/Dataset` | `vertex_ai_dataset` |
| `vertexai/RagEngineConfig` | `vertex_ai_rag_engine_config_basic`, `vertex_ai_rag_engine_config_scaled`, `vertex_ai_rag_engine_config_unprovisioned` |

All names above get the `ansible_` prefix applied and the corresponding `.tmpl`
file renamed/relocated as described in the naming convention section. Total: 79
template files across all 35 resource YAML files.

## Execution plan

1. **Snapshot** current `output/` (full `go run` regeneration) for later diffing.
2. **Rewrite overlay YAML files**: for each of the 35 resource files, perform a
   targeted text-level replacement of the `examples:` block with the equivalent
   `samples:` block (or insert a new `samples:` block where none exists). This
   must be surgical - locate and replace only the `examples:`/`samples:` key's
   lines - rather than a full YAML parse+re-serialize, because several of these
   files contain large embedded Python `custom_code` string blocks whose exact
   formatting/comments must be preserved byte-for-byte.
3. **Rename and relocate template files**: `git mv` all 79 `.tmpl` files to
   `overlay/templates/ansible/samples/services/<pkg>/ansible_<old-name>.tmpl`.
4. **Delete** the now-empty `overlay/templates/ansible/examples/` directory.
5. **Simplify the redirect code** in `pkg/api/loader.go`/`pkg/api/constants.go`:
   remove the legacy-`examples:` branch of `terraformExamplesToAnsible`, the
   `AnsibleExamplesDir`/`terraformExamplesDir`/`terraformExampleSuffix` constants,
   and the `examples:`-path check in `isAnsibleExampleTemplatePath` - leaving only
   the `samples/services/<pkg>/` redirect path. No changes needed in
   `pkg/ansible/examples.go` (`NewExamplesFromMmv1` already only reads
   `mmv1.Samples`, not `mmv1.Examples`, from earlier work).
6. **Document the convention** in `AGENTS.md`: add a section describing the
   `ansible_` prefix rule and the "always use `samples:`, never `examples:`"
   policy for all future resource onboarding.
7. **Verify**:
   - `go build/vet/test`
   - Full regeneration; diff against the step-1 snapshot - expect **zero
     differences** (confirms the migration is purely organizational, per the
     "names never render into output" finding above)
   - Python `ast.parse` / YAML `safe_load` syntax checks on all generated files
     (belt-and-suspenders, should be unaffected)
   - Confirm zero remaining references to `overlay/templates/ansible/examples`
     anywhere in code, and that the directory no longer exists

## Risks / open items to watch during execution

- The text-level YAML surgery script must be tested carefully per-file given the
  variety of surrounding content (some files have `custom_code` blocks
  immediately before/after the `examples:` key); a dry-run diff review before
  applying is recommended.
- `ReasoningEngine` has 6 additional example names declared upstream that we have
  never authored Ansible content for (`developer_connect_source`, `image_spec`,
  `byoc`, `psc_interface`, `context_spec`, `granular_ttl`) - out of scope for this
  migration since there's no existing template to migrate; they remain as
  harmless, unrendered upstream-native entries exactly as they are today.
- This migration should produce a functionally silent diff in `output/` - any
  non-empty diff after step 7 indicates a mistake in the mapping and must be
  investigated before proceeding further, not dismissed.
