# Magic-Modules v1 (MMv1) Upgrade Plan

This document outlines the plan to upgrade the `ansible-mmv1` generator to be compatible with a newer version of the upstream `magic-modules` dependency.

**Target Upstream Commit:** `bc6456aef9d0eb887c2d386ff8f0fc94f7b83aff`

**BLOCKER:** The target commit requires **Go >= 1.26.0**. This migration cannot proceed until the execution environment meets this requirement. The current environment is running Go 1.25.

---

## Summary of Breaking Changes

The analysis identified several major breaking changes in the target version:

1.  **Examples to Samples Migration:** The `examples: []` block in resource YAMLs has been replaced by a new `samples: []` block that supports multi-step testing. The upstream repository contains a mix of both old and new formats.
2.  **API Endpoint Construction Refactor:** The use of `BasePath` fields to construct API URLs has been removed in favor of a new `BaseUrl` function-based approach.
3.  **Introduction of `ResourceIdentity`:** A new `ResourceIdentity` feature was added to standardize how resources are identified, which may impact URL construction.

## The Complete Migration Plan

This is a comprehensive, end-to-end plan to perform the upgrade once the Go version blocker is resolved.

### 1. Update Dependency to Target Commit

-   **File:** `go.mod`
-   **Action:** Run `go get github.com/GoogleCloudPlatform/magic-modules/mmv1@bc6456aef9d0eb887c2d386ff8f0fc94f7b83aff` to update the Go dependency.

-   **File:** `config.yaml`
-   **Action:** Edit the `rev` key to `bc6456aef9d0eb887c2d386ff8f0fc94f7b83aff` to ensure the runtime git clone matches the Go module version.

### 2. Update Go Structs for Dual Format Support

-   **File:** `pkg/api/api.go` (or equivalent)
-   **Action:** Modify the core MMv1 resource struct to include **both** the old `Examples` field and the new `Samples` field. This allows the parser to handle both legacy and modern resource definitions from the upstream YAML.

### 3. Implement Backwards-Compatible Processing Logic

-   **File:** `pkg/ansible/examples.go`
-   **Action:** Update the example processing logic to be conditional:
    1.  Check if `resource.Samples` has content. If yes, process it using **new** logic (treating each sample as a single "create" step).
    2.  If `resource.Samples` is empty, fall back to checking `resource.Examples` and processing it with the **existing** legacy logic.

### 4. Update Overlay Template Loading Path

-   **File:** `pkg/api/loader.go`
-   **Action:** Modify the `ansibleExampleRedirectFS` to look for overlay templates in the new path: `overlay/templates/ansible/samples/`. A fallback to the old path will **not** be implemented to keep the logic simple.

### 5. Migrate `ansible-mmv1`'s Own Overlay Files

-   **Action:** Execute shell commands to move all of this project's custom templates to the new location to match the updated loader.
    ```bash
    mkdir -p overlay/templates/ansible/samples/
    mv overlay/templates/ansible/examples/* overlay/templates/ansible/samples/
    ```

### 6. Refactor API Endpoint Construction

-   **File:** `pkg/ansible/operation_config.go`
-   **Action:** Refactor the URL generation logic. Replace the use of the removed `BasePath` fields with the new `BaseUrl` functions provided by the updated MMv1 dependency.

### 7. Targeted Investigation of Resource URL Generation

-   **Files:** `pkg/ansible/operation_config.go`, `pkg/ansible/module.go`
-   **Action:** Perform a focused investigation to see if the `ResourceIdentity` changes impacted any URL generation helpers and apply minimal fixes as needed.

### 8. Final Verification

-   **Action:** After all changes are complete, run `go test ./...` to ensure the migration was successful and that all tests pass.
