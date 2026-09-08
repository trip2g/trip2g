---
title: "The job ships as a note"
free: true
lang_redirect: "[[ru/thoughts/ingest-as-a-note]]"
---

Your call recordings live at the recorder — mine at Krisp: its account, its rules, its search. trip2g pulls them out into a vault you own: the verbatim transcript, one note per call, search on top. What does the pulling is not a service and not a plugin. It is a markdown note. The frontmatter is the configuration, the body is a fenced python block, and that block is the job. You version it, edit it and sync it like any other note, and the recorder's token sits sealed in its own frontmatter.

Below: the anatomy of that note, the trap that cost me two calls, and the quote cutter — the move that takes away from the model the one job it lies about.

## The role note

In [[en/user/fleet|fleet]] a role is a note. An LLM role's body is an instruction; a code role's body is a program. Here is the ingest role's header, trimmed to what matters:

```
---
description: "Krisp meetings -> verbatim transcript notes (cron, no LLM)"
fleet_id: codellm
mode: cron
cron_schedule: "*/15 * * * *"
write_patterns: ["transcripts/**", "state/krisp-ingest.md"]
attach_notes: ["state/krisp-ingest.md"]
unseal: [krisp_token]
krisp_token: "sealed:v1:…"
timeout_seconds: 600
min_seconds: 60
max_per_run: 3
---
```

`fleet_id` names the fleet that picks the role up, and it also decides the kind of run: behind `codellm` the body executes as a program. `mode: cron` and `cron_schedule` set an alarm clock — the role wakes itself instead of waiting for a note to change. `write_patterns` are the globs it cannot step outside. `attach_notes` preloads notes into the delivery. `timeout_seconds` bounds the run; the default is 300. Everything below that line is mine: the minimum call length, how many recordings to take per run. The role reads its own settings out of its own header.

The body is a python block, and that block is the program. Four calls lead out of it:

```python
cfg = fleetkit.frontmatter()   # the role's own settings
sec = fleetkit.secrets()       # what was opened for this run

changes.append(fleetkit.note(path, meta, body))
fleetkit.emit(changes)
```

The code does not write the vault. It builds a list of notes and returns it; fleet writes them, after checking every path against `write_patterns`. A role given `transcripts/**` will not touch `roles/`, however hard the python tries. The boundary is not the code's good manners, it is a line in the note header you can read with your eyes.

## The secret lives in the note

The recorder's token is needed by the code and by nobody else. In the header it looks like this:

```
unseal: [krisp_token]
krisp_token: "sealed:v1:…"
```

The value was sealed by the agent's own form at `/_system/codellm/seal`, behind the admin login. The key lives in codellm's environment and nowhere else: not in the note, not in the repository, not on my laptop — and not in the sandbox where the code runs, either. A note carrying a secret stays an ordinary note. You sync it to Obsidian, open it on a phone, put it in a backup. What is inside is ciphertext. Only the process it is addressed to can open it, and only the fields listed in `unseal`. The details are in [[en/user/codellm-secrets|Secrets for code roles]].

## The queue is a note too

The role remembers how far it got in `state/krisp-ingest.md`: one `since_ms` field in the frontmatter. The note arrives in the run through `attach_notes`, and goes back out through the same `fleetkit.note()` at the end.

There is no state store. The vault is the queue. You can open the cursor, see where ingest stands, and rewind it by editing a number by hand. I needed that sooner than I expected.

## The trap: the transcript arrives after the recording

A recorder transcribes a call after the call has ended. A run that lands inside that window sees the recording — and an empty tree of utterances. My role dutifully marked such a call empty and moved the cursor past it. The call was then lost for good: the next run only takes what is newer than the cursor, and the transcript appeared behind it.

That is how two September calls disappeared. Both got their transcripts later, 62 and 48 utterances, and without rewinding the cursor by hand I would never have known.

A cursor in a queue like this is not "how far I have read". It is a watermark of settled recordings. Settled means written, or a demo, or shorter than `min_seconds`. A call with no utterances is not settled: the watermark stops there and the next run tries again. And after `empty_grace_hours`, a recording that never got a transcript stops holding the queue — otherwise one dead row blocks everything behind it.

## What comes out

One note per call, `transcripts/YYYY-MM-DD-<id8>.md`, verbatim, never edited afterwards. It is the source: everything else links to it, nothing rewrites it.

The role takes the call's instant not from the recorder but from the meeting id: it is a UUIDv7, whose upper 48 bits are milliseconds. Then it shifts to the owner's timezone, taken from the role's own header, because a daily note is a local day and not a UTC one.

This is not a fondness for bit shifts. The recorder's metadata turned out to be unusable as input: across a month picked apart by hand, 36 recordings, the titles were wrong in 29 of them, and the date in the title disagreed with the id in 18 files by a day or two. For a while the vault held three August transcripts under the wrong date. The id is right; the title is not.

## The raw archive is already the product

It is tempting to assume the archive becomes useful once a model has processed it. It does not work that way. I pulled down 340 transcripts covering nine months, and before a single model run I had something I had not had before: search over what was actually said. [[en/user/search|Search]] runs two lanes at once — full text for an exact phrasing, semantic for the meaning. "What did we say about X last quarter" comes back with a timecode, not a list of meetings.

What separates this from the recorder's own search is not quality, it is ownership. The transcripts sit in your vault, next to your other notes, inside the same links. It is a RAG store that belongs to you: an agent walks into it like any other folder, and [[en/user/subgraphs|subgraphs]] decide who may read what. The transcripts carry one subgraph label, the working `roles/` and `state/` another; a frontmatter patch applies the label folder-wide so nobody stamps it note by note.

## The quote cutter

The next stage is a call note a person actually reads: topics, takeaways, quotes as evidence. The obvious move is to hand the model the transcript and ask for a retelling with quotes. I made that move and got three different failures out of one prompt:

- a 117-minute call produced no note at all — a single non-streaming response did not fit inside the timeout;
- 85 KB of transcript came back as 34 KB, quotes silently truncated;
- 54 KB came back as 126 KB, quotes doubled.

The three failures share a cause. The size of the answer grew with the size of the input, and the model was retelling what it was supposed to be copying. No prompt fixes that, because I was asking the model to work as a photocopier.

So the work got split along the line of what each side is good at. The model writes only a boundary map: one `MM:SS | takeaway heading` line per turn in the conversation, plus a title and a two-sentence summary. A couple of kilobytes, whatever the length of the call. The quotes are cut by code, along those timecodes:

```python
for n, (start, tc, head) in enumerate(bounds):
    end = bounds[n + 1][0] if n + 1 < len(bounds) else None
    chunk = [u for u in utt if u[0] >= start and (end is None or u[0] < end)]
```

A quote is now a literal slice of the transcript. It cannot be shortened, doubled or invented, because a slice has no such operations. And the length of the call stops being a risk, because the model's output no longer scales with it.

One way to lie is left: name a timecode that is not in the transcript. Two guards stand against it. The first is that every timecode in the map must exist in the transcript verbatim, and the set of real ones is the utterance times. The second is what to do with a timecode that does not:

```python
near = min(known, key=lambda k: abs(k - s))
if abs(near - s) <= 90:
    snaps.append("%s->%s" % (tc, ntc))
else:
    problems.append("timecode %s is not in the transcript and nothing within 90s" % tc)
```

A boundary is snapped to the nearest real utterance if that one is within 90 seconds, and the snap goes into the finished note's header as `boundary_snaps`. If there is nothing near, the script refuses outright rather than shipping a draft with an invented anchor. The correction is visible to the reader; a silent correction does not exist.

On a 109-minute call this produced 240 quotes, zero of them non-verbatim, and one invented boundary — snapped and marked. The same call under the retelling approach produced no note at all.

What generalizes here is not the cutter, it is the seam. Give the model the judgement: where does the conversation turn. Give code the fidelity: what exactly was said. The seam between them is a timecode, and a timecode can be checked without reading a line. The same move for the second time: [[en/thoughts/krisp-segmentation-nano-vs-mini|the first]] had code compute the spans for the model, this one has it cut the quotes.

## What comes next

The analysis was built alongside, as a skill on [[en/thoughts/hermes-wiki|hermes]], and it runs by different rules: two gates where the owner says who was on the call and what to carry out of it, and an extraction that only mints notes for what he named. That is its own story, and it is not about ingest. What the transcripts grow into after that — call notes, concepts, daily notes — is shown in [[en/user/calls-knowledge-base|Calls into a knowledge base]].

## What this is and isn't

The measurements come from one archive and one recorder: 340 transcripts, of which one month was picked apart by hand. The metadata numbers are about that recorder and that month, not about recorders in general. What generalizes is the shape: a raw verbatim archive is useful before any analysis, and fidelity belongs in code.
