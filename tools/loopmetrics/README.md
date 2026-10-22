# loopmetrics

Measures repetition in Reasonix runs from the `--trajectory` JSONL stream, so a
claim like "Studio loops more than another agent" or "this prompt change
reduces looping" can be judged by data. Read-only: it never touches the kernel
or a session, and it only reads files it is given.

```bash
python3 tools/loopmetrics/loopmetrics.py run traj-a-1.jsonl [--json]
python3 tools/loopmetrics/loopmetrics.py cell --name A:short-readonly traj-a-*.jsonl
python3 tools/loopmetrics/loopmetrics.py resume first.jsonl second.jsonl
```

Standard library only. Tests: `python3 -m unittest discover -s tools/loopmetrics -t tools/loopmetrics`.

## Pre-registration

Every threshold below was fixed **before** the analyzer was run on any real
trajectory (this file is the first commit of the tool). Sensitivity at the
neighbouring values is reported next to every flagged-rate result, so a number
cannot have been tuned to a finding. Changing a threshold means a new
`schema_version`; version 1 keeps meaning what it says here.

| parameter | value | sensitivity reported at |
| --- | --- | --- |
| M2 shingle length | 8 normalised characters | not varied |
| M2 minimum block length | 40 normalised characters | not varied |
| M2 cross-block Jaccard flag | 0.8 | 0.7 and 0.9 |
| M2 within-block repeat flag | 0.5 | 0.4 and 0.6 |
| M3 look-ahead K | first 10 calls of the continued turn | 5 and 20 |
| M4 reply length cap | 8 normalised characters | not varied |
| minimum repeats for a result | 5 runs per cell | cells below it print "not a result" |

## Judgement reads structure, never wording

Equality of tool name, canonical arguments and result bytes; set overlap of
character shingles; equality of normalised text. No phrase table: words such as
"wait" or "hmm" are not counted, because a word list misfires on real input.

## Population and anchors

The population is a **turn**: the events from one `turn_started` to the next
(or the end of the file). A tool call is identified by its call id; the stream
emits `tool_dispatch` more than once per call, so every count is over unique
ids that also have a `tool_result`. A dispatch with no result (an interrupted
call) is reported as `unresolved` and enters no metric.

## Metrics

**M1 redundant tool calls.** A call is redundant when an earlier call in the
same turn had the same tool name and the same canonical arguments, succeeded,
and returned the same result bytes: the call gathered no new evidence.
Reported per turn as `redundant / unique calls` and `max_run`, the longest
stretch of consecutive calls that match each other that way. Excluded: a retry
after a call that failed (`err` set); a call whose result differs (polling);
anything repeated across a non-read-only call, which clears the history because
the state it read may have changed. Canonical arguments are the parsed JSON with
sorted keys. Sensitivity: `--ignore-arg description` drops a named argument
key from the comparison (a model-written rationale that can differ between two
identical commands); off by default, and both readings are printed when used.

**M2 reasoning repetition.** Over `reasoning` events (not the `message`
event, which repeats the concatenation). A block is normalised by lower-casing
and dropping whitespace, punctuation, symbols and control characters. Blocks
shorter than the minimum are excluded and counted. Within-block repeat is
`1 - unique shingles / total shingles`; cross-block repeat is the largest
Jaccard overlap of the block's shingle set with an earlier block of the same
turn. Reported: median and maximum of both, the share of blocks over each flag,
and reasoning characters and blocks as covariates, so "thinks more" and "thinks
the same thing again" are not read as one thing.

**M3 redo after stop-and-continue.** For a first and a continued trajectory:
among the first K resolved calls of the continued run, the share whose
(name, canonical arguments, result bytes) already completed in the first run.
Anchor: the continued file's first `turn_started`.

**M4 content-free repeats.** Over `text` events of a file, in order, normalised
as above: the longest run of consecutive events with identical normalised text
no longer than the cap. Longer identical texts are not counted here.

**M5 cost covariates.** Model requests, prompt, completion and cache-hit
tokens and cost, summed over `usage` events deduplicated by attempt id.

## What a result needs

A cell is one condition, one arm. Fewer than 5 runs in a cell prints "not a
result". Cells report median, minimum and maximum, not a mean alone, and the
count of runs that failed, timed out or produced no trajectory. A number
without those lines is an observation, not a result.

## What this does not establish

- A trajectory with no loop proves nothing about long sessions that were not
  run; a run's absence from the corpus is not evidence of absence.
- M1 sees repeated calls that read identically. A loop that varies its
  arguments each round (a new search phrase every time) is invisible to it.
- M2 is character overlap: a paraphrased loop is under-counted, and boilerplate
  that legitimately recurs (a fixed checklist) is over-counted.
- Neither metric says why a loop happened.
