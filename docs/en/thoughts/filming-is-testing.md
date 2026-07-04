---
title: "Filming the demo is a test"
free: true
lang_redirect: "[[ru/thoughts/filming-is-testing]]"
---

*Recording a clean video of a feature forces you to walk its entire happy path on the real product. The bugs that surface are the ones that otherwise wait for a live demo in front of a client.*

We set out to record an onboarding video: download the starter vault, install the plugin, save a note, watch it appear on the site. Nothing exotic, the exact path every new user takes. The recording took days longer than planned, and almost none of the delay was about video. Take after take failed because the product, on the real production system, kept refusing to do what every test said it could do. That turned out to be the value. Filming is a rehearsal of your demo, and it catches demo-day bugs while they still cost a re-shoot instead of a client's trust.

> **Video: coming soon.** The finished recording will be embedded here.

<!-- TODO: replace the callout above with ![[<file>.mp4]] when the render is done -->

### The class of bug the camera finds

Every bug the camera caught had the same shape: it sat right on the happy path, and it was invisible to unit tests because no single unit was wrong. Only the seams were.

A few, at the level of principle rather than changelog:

- The installer artifact that real users download had quietly drifted behind the code. Each part was tested; the packaged combination was not, because packaging happened once and nobody re-checked it. On camera, a save was supposed to publish itself. It didn't.
- The live instances hadn't actually moved to the new build, and requests failed in the middle of the rollout. Visible only on the system clients actually use, because staging never rolls out the way production does.
- Resetting the scene between takes revealed that hiding content and showing it again left it hidden. A state transition no test had ever asked about. Who hides things and brings them back? Anyone staging a demo. Then, eventually, some user.
- The signal we used to confirm "the deploy landed" measured something else entirely. We were reading a frontend cache marker and calling it a deployment check.
- The editor's autosave on switching files didn't reliably push the edit. A timing detail you only notice when you sit and watch the screen, waiting for a change that doesn't arrive.

None of these are deep. Any of them dies in five minutes once seen. The problem is being seen: each one hides below the resolution of a test suite and above the patience of a checklist. There is exactly one place where they all show up at once, and until now that place was the live demo.

### A rehearsal you can afford to fail

A recording has the same properties as a demo in front of a client: real product, the whole path, no skipping the boring parts, an audience that notices everything. With one difference. When a take fails, you fix the bug and press record again. When a demo fails, you explain yourself to someone who was ready to pay.

It is also stricter than a QA checklist. A checklist tolerates "works, with a caveat". A take doesn't; the caveat stays in the frame. You can't annotate your way around a site that never appeared.

### The rig, in principle

The recording itself is automated. Walking the same path by hand stops teaching you anything around the second time. Without turning this into a how-to, the ideas:

- The rig runs on Linux. The impression so far is that Linux gives more flexible control over windows than the alternatives, which matters when a script needs to place them, resize them, and capture the result. An impression, not a benchmark.
- It all runs on plain Xorg with no GPU acceleration, in the VM and on the server alike, and it has already proven itself there. Filming automation needs no special hardware.
- The final picture is framed in a macOS-style window anyway. Where you record and what the viewer sees are separate decisions.
- The browser can be driven two ways: through the DevTools protocol, or by simulating clicks. Both work, and they fail differently, which is occasionally useful in itself.
- Click animations are baked into the code, so the viewer sees the cursor move and press. A recording where things happen without visible cause reads as editing, not as a demo.
- Today the whole rig is driven by Claude Code, locally, on my machine, and we are packaging Hermes, the open agent we run ourselves, with the same skills. None of it is tied to one model: whoever can read the script and press the buttons can hold the camera.

Not step by step. But the idea is clear.

### How the agent does it

One run looks like this. A graphical session comes up on Xorg, without a GPU. The agent starts our stack and the clients, the browser and Obsidian, both with a remote-debugging port open. From there it drives them over the DevTools protocol: it places and resizes the windows, clicks, types, and waits for the change to reach the site. A cursor with a click animation is drawn on top, so every action has a visible cause in the frame. ffmpeg captures all of it from the X display. In post, the frame is wrapped in a macOS-style window, narration is laid over it, and the segments are stitched into one video.

### From learning to a packaged agent

The rig did not start as code. It started as an agent working out the machine on its own: place the windows, drive the browser, launch the app, walk the flow. That first pass is slow and it burns tokens, because the agent reasons through every step live.

Once a step is understood, it moves into code. The agent stops re-deriving how to click a button it has clicked a hundred times, the run gets cheaper and repeatable, and the judgment stays with the agent while the mechanics harden underneath it. Today Claude Code drives the rig and teaches it the steps; we are packaging those same skills into [Hermes](https://hermes-agent.nousresearch.com), an open agent we run ourselves. The end of that road is Hermes as the release agent: it runs on every release and produces the fresh "what's new" demo on its own.

Because it is an agent, one is not the limit. You can run ten of them at once on the same happy path, each walking it a little differently: a different order, different timing, different small choices. A single scripted take sees one path through the product. Ten agents see ten, and between them they knock loose the bugs that only appear when the steps do not happen in the order the script assumed. Not clever. Just many eyes doing the boring walk at the same time, which is one honest use of brute force.

### The output is the point

Ordinary QA produces a report that ages into a backlog. This produces the video you wanted for the landing page anyway, and the testing came free with it. Or count it the other way: the testing was real, and the marketing material came free. It adds up in both directions, which is rare.

There is a house rule behind this: [[en/thoughts/universality-tax|build when the content pulls]], and here the content did the pulling. We needed a video; the video needed the happy path to be true; making it true was the work. [[en/thoughts/dogfooding|Dogfooding]] in its strictest form: your own product, on production, in one unbroken take, with the tape running.
