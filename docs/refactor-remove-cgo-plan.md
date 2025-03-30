# Refactoring Plan: Replace CGO SQLite, Update Release Process (Based on CBZOptimizer Example)

**Goal:** Replace the CGO-dependent `mattn/go-sqlite3` library with the pure Go `ncruces/go-sqlite3` library, remove the need for CGO, update `.goreleaser.yml` to handle builds, Docker images (AMD64/ARM64), source archives, Syft SBOMs (for archives and source), and Cosign signing (for checksums and Docker images), and update the GitHub release workflow accordingly.

**Branch:** `refactor/remove-cgo`

**Target Platforms:** Linux AMD64, Linux ARM64 (Binaries and Docker Images)

---

## Phase 1: Code and Dependency Updates

1.  **Create Branch:** `refactor/remove-cgo`.
2.  **Replace SQLite Driver:** Update `go.mod`, run `go mod tidy`.
3.  **Code Adjustments (Verification):** Search for `sql.Open("sqlite3", ...)` usage, review `internal/database/`, compile (`go build ./...`).
4.  **Local Testing:** Run tests (`go test ./...`).

---

## Phase 2: Build Configuration Updates

5.  **Update `.goreleaser.yml`:**
    - Set `version: 2`.
    - Set `project_name: night-routine`.
    - Add `release:` section defining `github.owner: belphemur` and `github.name: night-routine`.
    - **(Optional but Recommended):** Add `changelog:` configuration similar to the example if desired.
    - **Builds Section:**
      - Keep `id: night-routine`.
      - Set `main: ./cmd/night-routine`.
      - Set `goos: [linux]`.
      - Set `goarch: [amd64, arm64]`.
      - Set `env: [CGO_ENABLED=0]`.
      - Set `mod_timestamp: '{{ .CommitTimestamp }}'`.
      - Add `flags: [-trimpath]`.
      - Update `ldflags: [-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{ .CommitDate }}]`.
    - **Checksum Section:** Ensure `checksum: { name_template: "checksums.txt" }` exists.
    - **Source Section:** Add `source: { enabled: true }`.
    - **GoMod Section:** Add `gomod: { proxy: true }`.
    - **SBOMs Section:**
      - Use two blocks with unique IDs, targeting `archive` and `source` artifacts, relying on the default Syft integration.
        ```yaml
        sboms:
          - id: archive # Default ID for archive SBOMs
            artifacts: archive
          - id: source # Unique ID for source SBOM
            artifacts: source
        ```
    - **Dockers Section:**
      - Use two blocks, one for `amd64` and one for `arm64`.
      - Set `use: buildx`.
      - Set `goos: linux`. Set `goarch` appropriately for each block.
      - Define `image_templates` for `ghcr.io/belphemur/night-routine` including `{{ .Version }}` and `latest` tags suffixed with `-amd64` or `-arm64`.
      - Include `build_flag_templates` with `--platform`, OCI labels (`org.opencontainers.image.*`), and `--pull`.
      - Add `extra_files: [configs/routine.toml]` to both docker blocks if this file needs to be included in the image.
        ```yaml
        # Example for amd64 block
        dockers:
          - image_templates:
              - "ghcr.io/belphemur/night-routine:{{ .Version }}-amd64"
              - "ghcr.io/belphemur/night-routine:latest-amd64"
            use: buildx
            goos: linux
            goarch: amd64
            build_flag_templates:
              - "--pull"
              - "--platform=linux/amd64"
              - "--label=org.opencontainers.image.created={{.Date}}"
              # ... other labels from example ...
            extra_files: # Add this if needed
              - configs/routine.toml
          # ... similar block for arm64 ...
        ```
    - **Docker Manifests Section:** Define `docker_manifests:` for `{{ .Version }}` and `latest` tags, combining the platform-specific images.
    - **Signs Section:** Add `signs:` block to sign the `checksum` artifact using `cosign sign-blob` with keyless signing (`COSIGN_EXPERIMENTAL=1`, `--yes`).
      ```yaml
      signs:
        - cmd: cosign
          env: [COSIGN_EXPERIMENTAL=1]
          certificate: "${artifact}.pem"
          args:
            [
              "sign-blob",
              "--output-certificate=${certificate}",
              "--output-signature=${signature}",
              "${artifact}",
              "--yes",
            ]
          artifacts: checksum
          output: true
      ```
    - **Docker Signs Section:** Add `docker_signs:` block to sign the Docker `images` using `cosign sign` with keyless signing.
      ```yaml
      docker_signs:
        - cmd: cosign
          env: [COSIGN_EXPERIMENTAL=1]
          artifacts: images
          output: true
          args: ["sign", "${artifact}", "--yes"]
      ```
6.  **Update `build/Dockerfile`:**
    - Remove CGO-specific dependencies and build flags.
    - Simplify `go build` command.
    - Ensure the final stage (`gcr.io/distroless/static-debian12:nonroot`) is appropriate.
    - Verify the `COPY` command for `configs/routine.toml` if needed and not handled by GoReleaser's `extra_files`.

---

## Phase 3: Workflow and Documentation Updates

7.  **Update `.github/workflows/release.yml` (Combined Approach - Corrected):**
    - Define `permissions:` including `contents: write`, `id-token: write`, `packages: write`, and `attestations: write`.
    - Include steps for:
      - `actions/checkout@v4` (with `fetch-depth: 0`).
      - `actions/setup-go@v5`.
      - `sigstore/cosign-installer@v3.8.1`.
      - `anchore/sbom-action/download-syft@v0.18.0`.
      - `docker/setup-qemu-action@v3`.
      - `docker/login-action@v3` (to `ghcr.io`).
      - `goreleaser/goreleaser-action@v6` (id: `goreleaser`) with `args: release --clean` and `GITHUB_TOKEN` env var.
    - **Add Post-GoReleaser Steps for Attestation:**
      - Add a step (id: `digest`) to get the manifest list digest using `docker buildx imagetools inspect ghcr.io/${{ github.repository }}:${{ fromJSON(steps.goreleaser.outputs.metadata).tag }} --format '{{json .Manifest.Digest}}'`. (Note: Ensure the output is correctly captured, e.g., using `echo "manifest_digest=$(...)" >> $GITHUB_OUTPUT`).
      - Add the `actions/attest-build-provenance@v2` step, using `subject-name: ghcr.io/${{ github.repository }}:${{ fromJSON(steps.goreleaser.outputs.metadata).tag }}` and `subject-digest: ${{ steps.digest.outputs.manifest_digest }}` (adjust output name based on the digest step).
8.  **Update Documentation (`docs/goreleaser-implementation-plan.md`):**
    - Remove or replace this file.

---

## Phase 4: Final Testing and Merge

9.  **Local GoReleaser Test:** `goreleaser release --snapshot --clean`. Check `dist` folder for binaries, checksum, signature, SBOMs (archive+source), source tarball.
10. **Workflow Testing:** Push branch, create test tag. Monitor action run. Verify:
    - Successful GoReleaser execution.
    - Docker images (AMD64/ARM64) pushed and signed in GHCR.
    - Multi-arch manifest list created in GHCR.
    - Binaries, checksum, signature, SBOMs (archive+source), source tarball attached to the draft release.
    - Attestations created and pushed for the manifest list.
11. **Code Review and Merge:** PR, review, merge `refactor/remove-cgo`.
