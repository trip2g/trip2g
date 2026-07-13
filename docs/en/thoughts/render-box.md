---
title: "The render box: how our demo-video agent got a remote machine and a job queue"
free: true
lang_redirect: "[[ru/thoughts/render-box]]"
---

# The render box: how our demo-video agent got a remote machine and a job queue

> 🎬 **The video this essay is about:** [English](https://youtu.be/u9DqHVndXv4) · [Russian](https://youtu.be/jqC9Q1i5O50)

**TL;DR.** We wanted an AI agent to produce a polished demo video for an open-source project on its own: record the scenes, voice them, montage them, verify the result. Recording inside the agent's local sandbox failed in five distinct ways, so we moved all heavy work to a cheap remote VM the agent drives over SSH, and when we wanted many agents sharing that VM, we put single-node Nomad in front of it as a resource-aware queue. The pattern that fell out is general: the agent orchestrates locally, a remote SSH box does the muscle, a queue keeps the box from melting.

## The goal

The task sounded simple. Take an open-source project with a one-command install, and have an agent produce the demo video end to end: script it, record terminal and browser scenes, generate the voiceover, cut everything together in Remotion, and check its own output frame by frame. No human at the editing desk.

The montage part worked early. Recording turned out to be the fight.

The companion essay, [[en/thoughts/filming-is-testing|Filming the demo is a test]], argues that recording a demo is itself a test of the product. This one is about the machine that makes the recording run.

## Five ways local recording died

The agent runs in a sandbox on a dev laptop. Each of these failures cost real diagnostic time, because each one looks like something else at first.

**1. `sleep` returns exit 144.** The sandbox kills foreground sleeps. Every screen-recording script on Earth is built out of "start capture, sleep, type command, sleep, stop capture," and all of those scripts silently break. Workarounds exist (pause inside `expect`, which runs outside the sandbox, or poll with `until xdpyinfo; do :; done`), but you only find them after staring at exit code 144 for a while and wondering what signal 16 even is.

**2. The 48-byte video file.** Repeated runs leave zombie `Xvfb` and `xterm` processes behind. The next `Xvfb :97` conflicts with the corpse of the previous one, and `ffmpeg x11grab` happily writes out a valid MP4 container with zero frames in it. 48 bytes. Roughly every second recording attempt failed this way until we learned to `pkill -9` the zombies and remove `/tmp/.X97-lock` before every single take.

**3. kitty renders nothing.** We wanted a pretty terminal. kitty needs OpenGL and comes up blank under Xvfb. Fell back to xterm with DejaVu Sans Mono, which is fine and boring and works.

**4. Chrome recordings collide.** Two browser scenes recorded in parallel share one singleton Chrome profile. They fight over it. One recording wins, the other captures a profile-lock error dialog.

**5. Electron paints nothing under bare Xvfb.** The strangest one. Obsidian (Electron) launches under Xvfb, creates a window, and the window never paints. The grab is blank. Electron apparently waits for a window manager event that bare Xvfb never sends. Run dwm on the display first and it paints fine. Nothing in any error log points you toward this; you get there by elimination.

Any one of these is a footnote. Five of them together, each surfacing as "the video file is wrong" with no error message, is a verdict on the environment. We were not debugging our pipeline anymore. We were debugging the sandbox, and the sandbox was not going to change.

## The pivot: a videobox

So the heavy work left the laptop. We provisioned a dedicated VM (Hetzner cpx32: 4 vCPU, 8 GB, a few cents per hour, Terraform + Ansible so it is reproducible and disposable) and gave the agent SSH access. The agent stays home and orchestrates. The box records and renders.

This dissolved every one of the five failures at once, which is the part worth noticing:

- Normal bash on a normal machine. `sleep` sleeps.
- A clean machine with no zombie accumulation. Grabs are stable; the 48-byte lottery ended.
- Its own disk. Remotion copies a multi-gigabyte `public/` directory into `/tmp` on every bundle, which was choking the shared dev machine and is a non-event on the box.
- Displays `:90` through `:99`, so parallel captures get their own X server and their own browser profile instead of fighting over one.

We also stopped grabbing X11 where we could avoid it. Terminal scenes are now recorded deterministically with asciinema + agg, which replays a scripted session into video with crisp text and, unexpectedly, perfect color emoji. Browser scenes use Chrome driven over CDP. Electron scenes still need Xvfb, plus dwm, for the reason above.

## The dwm patch, and why small code is an agent superpower

That fifth failure (Electron never painting under bare Xvfb) got fixed by dwm, and the fix is a small story worth telling because it inverts everything above.

dwm is the suckless dynamic window manager: a deliberately tiny tiling WM in C. The core is around 2,300 lines; the whole build we run, IPC patch included, is under 6,000. Small enough to read end to end in one sitting. When you put a real window manager over the display, Obsidian's window gets mapped and painted immediately and the blank grab problem is gone.

We run an IPC-patched build that exposes a Unix socket at `/tmp/dwm.sock`, so window management can be driven programmatically. A thin script, `dwm-stage.mjs`, talks to that socket to bring the recording stage up on a chosen display. That is what makes a window manager scriptable for headless recording instead of something you click at.

Here is the part that stuck with me. Patching dwm was trivial for the agent, and the reason is size. The entire program fits in the context window at once, so the agent reads all of it, understands all of it, and there is no framework ceremony sitting between intent and behavior. It works simply and like clockwork. Compare that to the sprawl the agent had been fighting for days: Electron's invisible dependence on a window-manager event, Chrome's singleton profile, Remotion copying gigabytes to `/tmp`, each an abstraction whose failure mode lives somewhere you cannot see. A few thousand lines of legible C is the opposite kind of object. The agent can hold the whole thing in its head, which is exactly the property that makes flaky abstractions expensive and small tools cheap.

A personal note, kept honest: the owner has run dwm as his daily driver for over ten years. That is a personal habit, not a recommendation, and he would not tell anyone else to switch to it. It just happened to be the right small tool within reach, and its smallness turned out to be the whole point.

## The manifest, or how the agent knows a scene is good

Moving compute did not solve verification, so the pipeline grew a structure for it. A video is scenes; a scene is micro-scenes; each micro-scene declares a typed, replayable `record_steps` list: `bash`, `launch`, `click`, `wait`, `capture`. Every step carries a `check` postcondition, and the micro-scene carries an `accept` criterion for the final frame.

The payoff is in the failure mode. The build extracts one screenshot per scene into a checklist, and a vision agent judges each frame against its `accept` criterion. When something is wrong, the failing step's `check` names it: "expected the sync output to contain 'Pushed N notes'" points at one step of one micro-scene, not at a vaguely bad final video. You re-record that micro-scene by id and leave the rest alone. Debugging a video stops being archaeology.

## The JSON is the actual system

Everything above runs off one file. The scenario is a single declarative JSON manifest: scene, then micro-scenes, and each micro-scene carries its `record_steps`, its `audioText`, and its `accept` criterion. The video, the audio slicing, the recording plan, the test spec, all of it is a projection of this one file. If I had to point at the real thesis of this project, I would point at the JSON, not the box or the queue.

Four things follow from making data the core, and each one is a concrete win rather than an aesthetic preference.

The agent reads and writes it trivially. It is just data, no framework, so an agent ingests the whole scenario in one go and knows exactly what the video is supposed to be. This is the same property that made dwm easy: small and legible enough to hold whole.

It lints. Structure, required fields, referenced assets, step types, timings that do not add up, all of it validates statically before a single second of render burns. Most "bad video" bugs never reach the render stage because they are schema errors, and schema errors are cheap to catch.

It makes "what's missing" obvious. The manifest declares every scene and every asset it needs. Diff declared against present and you get an exact worklist: this clip is not recorded, that VO slice is not generated. Regeneration is targeted by id instead of "redo the whole video," which is what makes an hour-long render loop tolerable.

The acceptance checklist falls out for free. Because each micro-scene already declares an `accept` criterion and a timing, code walks the manifest and emits a per-scene checklist: timestamp, prompt, and an auto-extracted screenshot, which the vision agent then verifies. The same file is both the build input and the test spec. You do not write the tests separately; they are a view of the thing you are building.

A second language made this concrete. We shipped the video in Russian first, then wanted English. Re-voicing was the easy half: regenerate the per-scene voiceover with an English voice and re-render. But several scenes had Russian baked into the pixels, not the audio: the documentation pages the browser walks through, the Obsidian note being edited ("Привет из Obsidian!"), the published site's home page, the little macOS window-title labels. Those had to be re-recorded in an English context, with English docs, an English note, an English site. The manifest is what made this a targeted job instead of a reshoot. Keyed by scene id, it told us exactly which scenes carried language in their visuals and needed a fresh take, and which were language-agnostic and could be left alone: the terminal install, the admin panel. A second language turned out to be a diff over the manifest, swapping audio everywhere and re-recording only the handful of scenes whose visuals actually spoke.

The through-line is the same one dwm illustrated from the other direction. A legible declarative data file is to this pipeline what a small C codebase is to window management: small enough to hold whole, so humans and agents both reason about it directly. Data is the contract; the code just executes it. And it lines up with the spine of the whole setup. The agent orchestrates, the box computes, the queue schedules, but the thing the agent actually manipulates, the object it thinks in, is this JSON.

## Then dozens of agents, hence Nomad

One agent and one box is just SSH. But once renders are a remote call, more agents start making that call, and a Remotion render will cheerfully eat all four cores for minutes. Three concurrent renders means everything is slow and one of them probably OOMs the box. You need admission control: run at most N, queue the rest.

We evaluated HashiCorp Nomad as a single-node batch queue, server and client in one process, `raw_exec` driver, and validated it live on a throwaway cpx32 (the test VM existed for 24 minutes and cost about €0.02). The mechanism is almost embarrassingly simple. Each render job declares `resources { cpu = 3500 }`; the node offers 8000 MHz; therefore two run at once and the rest sit `pending` with an explicit reason: `Dimension "cpu" exhausted on 1 nodes`. We dispatched six jobs and watched them drain in clean waves of two, with zero to one second between a slot freeing and the next job starting. No poller, no cron, the scheduler just re-evaluates blocked allocations when capacity frees.

One caveat deserves plain words: with `raw_exec`, resource figures are scheduling reservations, not enforcement. There is no cgroup. While two renders "reserving" 3500 MHz each were running, both ffmpegs were at ~180% CPU and the box sat at 95% overall. On a dedicated render box that is what you want (a render should use the whole machine; the queue exists to stop more from piling on), but it means the `memory` figure must be sized to a render's real peak, not to a wished-for cap, or two jobs will land and OOM together.

The agent-facing interface is one helper. `render-enqueue <id> "<cmd>"` over SSH dispatches, polls the allocation until it terminates, streams the logs back, and exits with the render's real exit code. Two Nomad footguns are buried inside it so callers never meet them: `nomad alloc logs -f` can hang forever after the task completes (we had three SSH sessions stuck that way), and Nomad's runtime `${...}` interpolation rejects bash's `${VAR:-default}` inside inline scripts. The queue survives daemon restarts, running ffmpegs included: Nomad reattaches to its executors, and an ffmpeg that started before `systemctl restart nomad` finished with exit 0 after it. Idle footprint is about 118 MB RSS and ~1% CPU. The whole thing shipped as an Ansible role plus two small scripts.

Could we have used something lighter? A flock-based FIFO or GNU parallel caps concurrency, but gives no persistent queue across reboots, no per-job logs or status, and DIY exit-code plumbing. Our own in-app Go queues (goqite/backlite) would have meant writing a submit API and worker on the box. Nomad was one apt package and ~50 lines of config for scheduling by resources, per-job logs, exit codes, and restart survival. For many external agents over SSH, it was less code, not more.

## What comes next, honestly

Today the render boxes are CPU dev VMs, rented for roughly an hour and destroyed. The obvious next step is the same pattern on a GPU host: hardware-accelerated encode, faster Remotion renders, room for ML steps like forced alignment in the same queue. We have not built that. The queue semantics would not change at all, which is a point in the design's favor, but it is an aspiration, not a result.

## The pattern

Strip the video specifics and what remains is this: the agent is an orchestrator, a remote SSH-accessible box is the heavy-compute substrate, and a resource-aware queue is the scheduler between them. Heavy work, flaky work, and parallel work all have the same property: they get worse inside an agent's sandbox and better on a dedicated machine. The agent does not need local compute. It needs a reliable way to say "run this, tell me the exit code," and to wait its turn when the machine is busy.

We got there backwards, one dead end at a time, and each dead end paid for the next piece: the sandbox failures bought the box, the box bought parallel agents, parallel agents bought the queue. That is usually how infrastructure happens.
