---
owner: @esengine
backup: @SivanCola
status: active
reviewed: 2026-10-12
---

# Intelligent UI implementation

[RFC #12584](https://github.com/esengine/DeepSeek-Reasonix/issues/12584) defines the staged design. This first implementation covers desktop recovery and measurement (P0). TUI changes are deferred by the owner. The model-facing `render_surface` tool and typed fields are subsequent work.

## Recovery boundary

The extension UI hub retains at most 64 current publications, keyed by plugin and surface identity. A republish replaces that identity. Snapshots are isolated copies of the already validated and credential-redacted payload. Notifications are transient and are never replayed.

`GET /extensions/surfaces` reads the controller snapshot and the session's existing wire log. A window refresh in the same running generation restores the current payload.

A retired generation, crashed plugin, or restarted host returns identity only with `surface.expired`; the desktop removes old controls and displays the reason. It does not resubmit a form or open a question.

The wire log records only plugin, surface, session, generation and kind. It never records the publication body, field defaults or user answers. No new session schema or store is introduced.

A history predating this implementation has no publication identities to recover. A truncated or unreadable recorded history returns `surface.replay_incomplete`, rather than claiming that no surfaces existed.

A recovery snapshot does not owe entrance motion. Live publications received during a snapshot read take precedence for their identity; results from an obsolete read or unmounted pane are discarded.

This endpoint does not expand execution or approval authority.

## Desktop presentation

Recovered publications use the existing Studio `.flow` presentation. Expiry is a plain `restore-notice` status row. It has no tool gutter, vertical line, card frame or visible internal error code; `data-code` retains the typed identity for inspection and tests.

## Measurement definitions

All counters are local to one mounted desktop pane, exposed on its `PaneReport.interactions`. They are reset when that pane is remounted, are not uploaded, and contain no question, option or answer strings. Local deduplication retains prompt identities only for that pane's lifetime.

| Counter | Counts | Anchor and population | Excludes |
| --- | --- | --- | --- |
| `requests` | First observed structured ask request per prompt id | First request event in this mounted pane; model and external structured questions visible to this pane | Duplicate delivery and questions rebuilt from stored history; this is not the number of provider tool calls |
| `questions` | Question entries in those first requests | The same request population | Duplicate delivery and stored history |
| `options` | Declared option entries in those questions | The same request population | The frontend's added free-text affordance and answer content |
| `answers` | First successful local answer submission per prompt id | Acknowledged submission in this pane | Failed submissions, retries after the same acknowledgment, other devices and inferred outcomes |
| `customAnswers` | Those submissions with a selection absent from the corresponding declared option labels | The same acknowledged-answer population; exact option membership | Answer text, empty skips, and any claim that the model asked the same question again |

These counters measure desktop interaction exposure and local answers. They do not establish task success, user friction improvement, or an account-wide rate. A later fixed-task trajectory experiment must compare user turns to the same completed outcome before reporting a benefit from P1.
