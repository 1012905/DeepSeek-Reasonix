---
status: active
owner: @esengine
backup: @SivanCola
reviewed: 2026-10-17
---

# Unified project sidebar — design QA

Date: 2026-10-17, Asia/Singapore.

## Visual sources and scope

Source visual truth: selected generated design, third displayed image:
`selected-design.png` (retained in the private QA artifact set).

Later user references supersede its standalone open-project row and chevrons:
`zcode-project-list.png` and `zcode-fold-toolbar.png` in the private QA artifact set.

Scope: the existing production frontend's sidebar and its project/session actions.
The main conversation, native titlebar and account UI retain their existing owners.

## Browser evidence

Implementation screenshot:
`projects-expanded.png` in the private QA artifact set.

Source and implementation full frames are both 1487 × 1058 pixels.
Implementation CSS viewport is 1487 × 1058, with reported devicePixelRatio 1.
No raster scaling or image replacement was used in the accepted comparison.

The implementation uses the existing 316px sidebar and supplied product fonts and logo.

User-provided partial screenshots have unspecified density; their hierarchy and
interaction requests, rather than their raw pixel measurements, define the changes.

State: light theme, local and SSH projects in one list, project children expanded.
The development fixture supplies different conversation titles from the concept.

The full frame confirms alignment, hierarchy, selection, toolbar positions and footer.

The toolbar, project glyphs, metadata and selected rows were also inspected at native
pixels and through computed DOM geometry. A separate cropped image was unnecessary.

A capture that altered the viewport was rejected and removed before comparison.

Additional accepted screenshots:

- `projects-open-dialog.png`: 1280 × 720 viewport; common local/SSH location picker.
- `projects-narrow.png`: 1280 × 720 viewport; sidebar resized to its 232px floor.

These screenshots are in the same artifact directory as the implementation frame.

## Findings and comparison history

All findings reproduced in the scoped sidebar audit have been fixed.

- Machine parent rows were removed. Local and SSH projects now share level 1;
  conversation rows share level 2. Host plus root identifies remote fold state.
- The open-project row moved to the heading's persistent plus, as requested.
  Its modal retains a visible location choice and a focused, cancellable first action.
- Separate project chevrons were removed after the ZCode reference was supplied.
  Heading clicks and tree arrow keys retain folding and parent navigation.
- Collapse/expand-all was added beside the project label. It controls both project
  sources, changes its label/icon, and is disabled while search reveals matches.
- Local-only filter totals were removed rather than presented as unified totals.
  Canonical loaded/total counts remain accessible without visible count badges.
- Recent previews show five sessions, preserve an older active/opening session,
  and expose remaining records through the existing bounded pagination path.
- Narrow-width title truncation preserves full project paths in hover text and
  does not overlap actions. A settled 232px measurement showed no row overflow.

Post-fix screenshots, DOM state and interaction tests substantiate these resolutions.
Differences from the original concept are the user's later requested refinements,
the existing product's width/font scale, and development fixture content.

## Fidelity surfaces

- Typography: supplied Inter/Noto Sans SC stack; 13px project titles, 12px session
  titles, 11px secondary metadata. Muted project headings and darker session text
  follow the later ZCode reference. Long titles truncate inside their own grid track.
- Spacing: common folder/title/action grid, compact 34px rows, shallow session
  indentation, no standalone arrow column, fixed toolbar hit areas.
- Tokens: existing warm neutral surfaces, semantic muted text, blue selection
  and focus tokens. Selected and inactive rows were inspected separately.
- Assets: original repository wordmark and icon component. The diagonal glyphs
  reuse official Tabler paths with provenance recorded in `styles/LAYOUT.md`.
- Copy: named open-project, new-conversation and fold actions have distinct scopes.
  Time labels use canonical last-active timestamps and are absent when unknown.
  Timestamp rendering is covered by the preview interaction tests; the browser
  fixture omits those timestamps, so its screenshot does not invent dates.

## Validation and limits

- 328 tests passed across 54 related UI/style test files after the workflow audit.
- The audit added 15 regression cases, including controlled async ordering.
- `pnpm build` passed, including TypeScript compilation.
- `make check` passed without widening a ratchet baseline.
- Browser checks passed for unified local/SSH expansion, collapse-all and its
  reversal, the location picker, keyboard parent navigation, and narrow resizing.
- Palette search found an SSH conversation and opened its owning host in MockHub.
- The independent verification tab reported no console errors or warnings.
- Temporary viewport overrides were cleared and the preview remains available.

The initial visual pass used fixtures and boundary tests. The subsequent real
SSH run and its additional repairs are recorded in [ssh-qa.md](ssh-qa.md).
No model request, native Electron package, publication or CI run was used as
evidence. Full accessibility compliance is not claimed.

## Workflow audit — 2026-10-17

The audit reproduced the following defects in deterministic component or window
tests before applying the repairs. Some were inherited by the unified flow;
they are not all regressions introduced by the visual changes.

| Priority | Trigger and symptom | Repair owner and verdict |
| --- | --- | --- |
| P1 | A local project and an SSH project share a root path; a local open reuses the remote blank or idle pane. | `useOpenPane` restricts transcript reuse to local runtimes. Fixed. |
| P1 | An earlier SSH request finishes after a newer local or SSH selection and takes focus back. | `useOpenRemotePane` uses the shared session-switch ticket across both sources. Fixed. |
| P2 | Revisit an SSH session while its first open is pending; a second proxy pane is created. | `useOpenRemotePane` shares the pending request by host, workspace and transcript. Fixed. |
| P2 | A running SSH pane has no history row yet and disappears from Running. | `remoteProjects` includes runtime-only rows before filtering projects. Fixed. |
| P2 | A remote transcript pinned by its kernel is absent from Pinned. | `remoteProjects` reads the canonical pin and preserves pin ordering. Fixed. |
| P2 | A remote transcript is archived elsewhere while its pane remains open; All reconstructs it as a missing row. | Fallback rows are compared with the unfiltered history, preserving archive visibility. Fixed. |
| P2 | Collapse all omits a remote project represented only by an open pane. | The toolbar enumerates the same runtime-backed project roots as the list. Fixed. |
| P2 | Neither source matches a query or filter; a mounted remote component suppresses the empty message. | `Workspaces` checks the shared remote projection instead of component presence. Fixed. |
| P3 | Filtering out loaded archived rows reports them as older, unloaded history. | Older counts are derived before display filtering. Fixed. |
| P3 | Escape in the SSH directory picker invokes the outer and inner dismissal handlers. | Only the active location stage handles dismissal. Fixed. |

Production reachability follows the existing contracts: a remote open publishes
its runtime before a first saved turn; local and remote roots may have identical
spelling; pins and archive state are read from the far kernel's `/tree` response.

Rapid selection and repeated opens are reproduced with controllable promise
completion, not timing sleeps. No private kernel fields or production stores
were modified to construct the reproductions.

Additional browser evidence: `flow-empty-filter.png` in the artifact directory
shows feedback for an empty unified filter. Browser interaction also confirmed
search feedback and collapse-all including a newly opened remote pane.

Cancelling the SSH directory picker restored focus to the open-project button. No browser
console errors or warnings were recorded.

Final audit results: 328/328 related tests passed; TypeScript and the production
build passed; `make check` and `git diff --check` passed. The fixture and boundary
tests do not establish real SSH transport or native Electron behavior.

The final machine-readable result is saved as `tests-flow-final.json` in the
artifact directory. The ten reproduced issues above are resolved; no publication
or kernel protocol change was made.

## Implementation checklist

- [x] Apply the selected unified list and later user refinements.
- [x] Preserve source-aware actions, search and bounded pagination.
- [x] Verify browser geometry, keyboard behavior and primary controls.
- [x] Complete focused regressions, production build and repository checks.

## Follow-up polish

At the minimum sidebar width, remote labels and reserved actions leave less room
for long project names. Full hover text remains available; moving the source label
to a second line is an optional P3 refinement, not a required acceptance change.

## ZCode session selection refinement

Source: the user-provided ZCode screenshot
showing a neutral selected conversation row
and [ZCode TaskListItem](https://github.com/zai-org/ZCode/blob/main/packages/ui/src/TaskListItem.tsx).

The source selects with `bg-selected`, uses a separate hover fill, and renders
the pin as a pin action/state rather than a universal selection indicator.

Scope is the conversation selection treatment, preserving Reasonix fonts,
sidebar dimensions, metadata and action semantics. Source screenshot density
is unknown; this is a pattern comparison rather than a pixel-exact clone.
The source and all three implementation crops were opened together for review.

Evidence set: `selection/` in the private QA artifacts; images are not committed.

- `light.png`: neutral selected fill, no blue edge or black short bar, matching
  regular title weights for selected and unselected rows; no clipping.
- `dark.png`: the same recipe follows theme ink, with a visible neutral fill.
- `keyboard-focus.png`: the first conversation remains selected while the
  second has a separate keyboard outline. Enter opens the focused conversation.
- `full.png`: the accepted final light frame, with the original conversation
  and theme restored. No new images or copy were introduced.

Browser computed styles confirm 8% ink for selection, 4% for hover, equal
500 title weights in the current Chinese typography setting, no box shadow,
and no selection-generated pseudo-element.

Existing execution markers retain
their execution semantics. The shared rail recipe covers local and SSH rows.
Pointer switching, direction-key focus and Enter activation passed; console
warning/error inspection returned no entries.

This pass uses the existing
development fixture and makes no new claim about real SSH transport.

Validation: 42 tests across four relevant suites passed; the TypeScript and
production build, `make check`, and whitespace checks passed. The build retains
its existing large-chunk advisory. No P0/P1/P2 selection issues remain.

## ZCode hover actions

The same user-provided ZCode reference now also defines the hover action
positions: a leading Pin action and a trailing Archive action. Delete is
available from the conversation menu. This scope retains Reasonix copy and
fonts rather than cloning unrelated ZCode navigation or metadata.

Evidence set: `hover-actions/` in the private QA artifacts; images are not committed.

The source and `local-hover.png`, `dark-hover.png`, and `narrow-hover.png` were
opened together. The source has unknown density; implementation crops are
316px and 232px CSS sidebar widths. Comparison covers action placement,
neutral fills, spacing, typography, icon legibility and unchanged copy.

- Local and SSH rows share the same action components and horizontal grid.
  Title ink remains at x=94 before and during hover. At 232px the title
  truncates without touching either button.
- Pin and unpin preserve the active conversation. SSH pinning, archiving,
  finding the archived transcript and restoring it with Enter were exercised
  through the development fixture. Test pins and archives were restored.
- `remote-keyboard.png` records focus returning to the same Pin button after
  its asynchronous write. Pending writes disable duplicate submission.
- `remote-menu.png` records the keyboard menu with Pin, Archive and Delete.
  Escape returns focus to the conversation row. Hover action visibility also
  follows keyboard focus; touch surfaces reveal idle-row actions directly.

One browser pass found the focused SSH action losing focus while disabled.
The component now restores that original action after the write settles;
the post-fix browser check confirms the focused unpin button and unchanged
active local conversation.

Final screenshots show no unresolved P0/P1/P2
issues. Existing compact metadata remains secondary and does not jump.

The full frontend suite passed 5254 tests across 633 files. After the focus
repair, 13 focused interaction tests passed again. The production build,
the complete Go serve package tests, repository lint and whitespace checks
passed.

The build retains the existing large-chunk advisory.

Remote flag proxy tests verify host routing, explicit false flags, link
release, disabled-host refusal and preservation of the far kernel's typed
errors. No session storage format changed. Existing local endpoint payloads
remain unchanged; new remote routes forward existing far-kernel operations.

Saved explicit off preferences remain off; an unset preference now shows
the hover controls. This pass does not add a new real-SSH transport claim.

Console inspection after a fresh navigation found no current warning/error
entries. Two earlier Vite hot-update duplicate-root messages remain in the
browser's historical log and are separated in `console.json`.

final result: passed

Retrospective: the existing canonical-state and test-contract guidance covers this
change. No additional reusable skill rule was needed.
