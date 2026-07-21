# Review Remediation Plan

This document captures the post-review work required before `gofiledialog` is
used as a shared component across projects. Complete the phases in order: the
first three fix correctness and resource-lifecycle defects that can affect host
applications in production.

## 1. Make dialog completion reliable (P1) - Completed

**Problem:** Closing an Open, Save, or Folder window through the window manager
does not invoke `SetOnChosen`. Explicit action paths invoke the callback
directly, so a callback can also be susceptible to duplicate completion when
multiple UI actions overlap.

**Status:** Completed on 2026-07-13. Open, Save, and Folder dialogs now route
explicit completion and title-bar/window close through an idempotent completion
guard. Window close flushes settings and completes as cancellation, while stale
completion paths such as overwrite confirmation cannot deliver a second result.

**Work:**

- Give each dialog one idempotent `finish(paths, err)` path, guarded by
  `sync.Once` or an equivalent completion state.
- Have `win.SetOnClosed` both flush settings and complete an unfinished dialog
  as cancellation (`nil, nil`).
- Route buttons, Escape, file activation, overwrite confirmation, and external
  window close through the same completion path.
- Ensure a confirmation callback cannot complete a dialog that has already
  been cancelled or closed.

**Verification:** headless Fyne tests for explicit cancellation, title-bar
close, selection, Save overwrite confirmation, and exactly-once callback
delivery.

## 2. Add thumbnail worker shutdown (P1) - Completed

**Problem:** Each `Browser` starts up to eight thumbnail workers, with no way
to stop them when the browser/dialog closes. Repeated dialogs leak goroutines
and retain their cache state for the lifetime of the process.

**Status:** Completed on 2026-07-13. `thumbnailer.Close()` now marks the
thumbnailer closed, wakes idle workers, drops queued and in-flight callbacks,
and waits for workers to exit. `Browser.Close()` owns thumbnail shutdown and
dialog close handlers invoke it for Open, Save, and Folder dialogs. Requests
made after close are ignored, and work finishing after close cannot update UI
callbacks.

**Work:**

- Add `thumbnailer.Close()` with a closed state, cancellation signal, and
  `sync.WaitGroup`.
- Change worker waiting so queued workers wake and exit during close.
- Clear queued/in-flight requests and ensure their callbacks cannot update
  disposed UI cells.
- Add `Browser.Close()` and invoke it from the dialog close handler.
- Define behavior for requests made after close (normally: ignore them).

**Verification:** create and close browsers repeatedly, wait for shutdown, and
assert that worker counts return to baseline. Test that a thumbnail completing
after close cannot update the view.

## 3. Make persistent settings atomic and serialized (P1) - Completed

**Problem:** The shared default settings file is written directly. Timer-based
and flush-based writes can overlap; multiple dialogs or processes can also
write concurrently. A reader can observe partial JSON and fall back to
defaults, losing user settings.

**Status:** Completed on 2026-07-13. `debouncedSaver` now serializes timer and
flush writes and ignores stale timer callbacks after a flush. The default
`fileStore` now serializes saves per settings path, coordinates cross-process
writes with a short-lived lock file, writes through a synced temporary file,
and atomically replaces `settings.json`. On write failure, the previous valid
settings file is left in place. Cross-process behavior is documented as
shared, lock-coordinated, and last-save-wins.

**Work:**

- Serialize `debouncedSaver` writes so a timer callback and `Flush` never call
  `Store.Save` concurrently.
- Persist through a unique temporary file in the settings directory, close and
  sync it, then atomically rename it over `settings.json`.
- Add a lock keyed by settings path for in-process dialogs.
- Decide and document cross-process semantics. Because the default store is
  advertised as shared between apps, use an OS-level lock or make the default
  store app-scoped instead.
- Preserve the last valid file on write failure and expose failures through an
  optional persistence-error callback (or clearly document best-effort
  behavior).

**Verification:** stress test concurrent `Save`/`Flush` calls, test simulated
write failure, and repeatedly parse the final settings JSON. Add a
multi-process integration test if cross-process sharing remains supported.

## 4. Align the supported Go version (P1 release blocker) - Completed

**Problem:** `README.md` says Go 1.22+, while `go.mod` declares Go 1.26.1.

**Status:** Completed on 2026-07-13. Go 1.22 is the declared supported
minimum: `go.mod` now uses `go 1.22`, matching the README. CI now runs
`go test ./...` and `go vet ./...` on Windows, macOS, and Linux for Go 1.22.x
and the current stable Go release.

**Work:**

- Choose the real supported minimum Go version.
- If Go 1.22 is intended, lower `go.mod` only after confirming all APIs and
  dependencies build there.
- Otherwise correct the README, release notes, and support policy.
- Add CI for every supported Go version on Windows, macOS, and Linux.

**Verification:** clean builds and tests in the full CI matrix, including a
fresh consumer module using `go get`.

## 5. Restrict New Folder to a single folder name (P2) - Completed

**Problem:** The New Folder form accepts paths such as `..\\outside` and joins
them to the current directory, permitting folder creation outside the visible
location.

**Status:** Completed on 2026-07-13. New Folder input is now validated as a
single folder name before any filesystem operation. Empty names, leading or
trailing whitespace, `.`, `..`, absolute paths, path separators, control
characters, and platform-invalid names are rejected. The destination is built
from the validated name and checked to remain inside the browser's current
directory.

**Work:**

- Reject empty values, `.`, `..`, absolute paths, path separators, and
  platform-invalid name characters.
- Construct the destination from a validated basename only.
- Check that the cleaned destination remains inside `Browser.CurrentDir()`.
- Report existing-directory and permission errors clearly.

**Verification:** table-driven tests for traversal, absolute paths, separators,
valid Unicode names, and already-existing directories.

## 6. Implement or remove parent-window semantics (P2) - Completed

**Problem:** Constructors accept `parent fyne.Window` and document centering
against it, but ignore the parameter and always center on screen.

**Status:** Completed on 2026-07-13. Fyne's public `fyne.Window` interface
does not expose portable window position/move APIs needed for reliable
parent-relative placement. The current API keeps the `parent` parameter for
compatibility with Fyne-style dialog helpers, but GoDoc and README now state
that gofiledialog creates top-level windows and centers them on screen.

**Work:**

- Prefer centering relative to a non-nil parent, with screen-centering only
  when there is no parent.
- Apply the behavior consistently to Open, Save, and Folder dialogs.
- If Fyne cannot support this reliably across platforms, remove the parameter
  in the next major API version and correct the documentation now.

**Verification:** unit-test a positioning helper and manually verify placement
with multiple windows on supported platforms.

## 7. Make filters independent of their labels (P2) - Completed

**Problem:** Filter selection uses the visible label as a map key. Duplicate
labels silently overwrite one another and select the wrong filter.

**Status:** Completed on 2026-07-13. Fyne's `widget.Select` identifies options
by display string, so duplicate visible labels cannot be made reliable while
preserving labels exactly. Open and Save dialogs now reject duplicate filter
labels during construction with a clear error, including duplicate labels
generated from unnamed filters.

**Work:**

- Associate selector options with a stable index or unique internal ID rather
  than their visible text.
- Alternatively, reject duplicate labels during dialog construction with a
  useful error.
- Preserve display labels exactly as supplied by the caller.

**Verification:** test duplicate named filters, unnamed filters that normalize
to the same label, and repeated switching between filters.

## 8. Harden API contracts and test coverage (P2) - Completed

**Status:** Completed on 2026-07-13. Settings persistence is now explicitly
documented as best-effort: load failures fall back to defaults and save
failures are ignored so UI interaction is never broken by a settings backend.
`Browser.SetSort` now ignores unknown column IDs instead of persisting invalid
sort state. Additional coverage was added for corrupt settings, custom store
save failures, symlink directories, missing directory read errors, malformed
thumbnail files, oversized image headers, and dialog lifecycle behavior. Local
validation passed `go test ./...`, `go vet ./...`, and focused
`go test -race .`; the supported-platform matrix is covered by the CI workflow
added in phase 4.

**Work:**

- Decide whether custom `Store.Load` and `Store.Save` failures are returned,
  reported asynchronously, or intentionally ignored; make code and docs agree.
- Validate invalid public inputs such as unknown `ColumnID` values passed to
  `Browser.SetSort`.
- Add coverage for symlink directories, unreadable directories, corrupted
  settings, malformed/oversized image headers, and persistence migration.
- Run `go test -race ./...`, `go vet ./...`, and platform CI before each
  release.
- Add an integration test suite that exercises real dialog lifecycle behavior,
  which is not covered by the current mostly unit-level tests.

## Completion order

1. Dialog completion semantics
2. Thumbnail worker lifecycle
3. Atomic and serialized settings persistence
4. Go-version contract and CI
5. New Folder validation
6. Parent-window behavior
7. Filter identity
8. Broader API hardening and release validation

Do not cut a reusable release until phases 1 through 4 are complete and the
race detector plus the supported-platform CI matrix are green.
