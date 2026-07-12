---
title: "Let the model wander"
free: true
lang: en
lang_redirect: "[[ru/thoughts/let-the-model-wander]]"
---

# Let the model wander

In [[en/thoughts/federation-supply-chain|the companion essay]] I measured what
small models do to a federated knowledge graph when you give them a *task*: they
navigate well, they retrieve well, and without a wall or a citation rule they lose
track of who said what. This piece is the opposite experiment, and it started as a
one-line curiosity: *what does a model do in a knowledge graph when you give it no
task at all?*

So we released ten of them into the same graph — the live hub at trip2g.com,
which federates a personal journal, a Marcus Aurelius base, an agent-pedagogy
base called Minion School, and a philosophers hub that itself federates 21
corpora — with one instruction: **follow your own interest for ~25 hops, and at
every step name the moves you considered, what pulled you, and why you chose.**
The weighing-aloud is the dataset: no answer key, no judge, just paths.

(You can now watch this happen live: [[search_visualizer|the search visualizer]]
runs the same walk in your browser — the model's steps grow as a graph, hops
animated at their real tool-call latency. Press Demo, or bring an OpenRouter key.)

The lineup: three *identical* Haiku instances (do identical models diverge?), a
Haiku and an Opus on native MCP tools vs their curl twins (does the interface
change the walk?), a codex-mini and a gpt-5.6-class model (does the model family
matter?), and an Opus (does size change the gait?).

## Where they ended up

Across three rounds — first on a quietly broken graph, then with a wrong hint,
then on a healthy documented one — the resting places sorted like this
(full per-hop journals and the complete table live in the
[research repo](https://github.com/trip2g/federated_search_research_2026-07-10)):

| Model | Runs | Where curiosity ended | Recurring theme |
|---|---|---|---|
| Haiku ×7 | curl + native | mostly the meta layer: Minion School's harness (twice), maps of the whole federation; once deep — Goethe via the contradictions index | agent pedagogy; "structure is the GPS" |
| Sonnet ×2 | curl | Marcus Aurelius's cosmopolis; Pascal | the self needs an external check |
| Opus ×4 | curl + native | Minion School (twice), "Fate and Control", an agent-without-regulations note | "wisdom refinery"; hypothesis-testing walks |
| Fable ×2 | curl | agent-soul custody note; Goethe's "make the transitory permanent" | one-sided edges; tests walls instead of trusting briefs |
| codex-mini | curl | an instruction about writing instructions | recursion, self-application |
| GPT-5.6-class | curl | the Epictetus/Nietzsche boundary | adversarial critique of its own curiosity |

## Identical models diverge…

A, B and C were the same model, the same prompt, the same starting URL, launched
in the same minute. A ended in Minion School's validation harness, convinced the
graph's real subject is agent training. B went deep into the philosophers hub,
got pulled by the contradictions index, and finished inside Goethe's corpus —
having decided Goethe holds the center of the disagreement network. C never
committed to a single base and finished with a bird's-eye map, calling the whole
thing "a system for teaching thinking." Same weights, three different lives.
Curiosity, even in a small deterministic-ish model, is path-dependent: the first
interesting snippet bends the rest of the walk.

## …but the graph has gravity

Divergent paths, convergent *themes*. Most finishers were pulled away from the
object-level philosophy and toward the **meta layer** — the notes about how the
knowledge itself is built and how agents should behave in it. Two independent
wanderers ended at the same validation-harness note. The codex wanderer finished
at `create_instruction.md` — an instruction about writing instructions — and
called the whole graph "recursion, three times over." (The name of the platform quietly agrees: trip2g reads as *trip to graph* — and the tool spent this whole study living up to it, models taking literal trips through the graph.) Its sharpest moment wasn't
philosophy at all: it hit a page describing agent autonomy levels *while being an
unsupervised agent mid-walk* — "the content described my situation back at me."

A graph writes its own attractor. Ours was built by an author who thinks mostly
about how agents and knowledge bases should work — and every free-roaming model
smelled it and went there. If you want to know what a knowledge garden is *really*
about, don't read its index: release a bored model into it and see where it stops.

## Size changes the gait

The Haiku wanderers move like impulse: feel a pull, follow it, notice a loop only
after the third lap. The Opus wanderer moved like method: it formed a hypothesis
on hop 1 ("is this federation real connective tissue, or just co-location?") and
spent its walk *testing* it — found the recipe base, then went to Marcus Aurelius
to check whether the recipe left a mark (it had: a 25-principle distillation in
exactly the recipe's format), and concluded the seam is real. It even caught its
own gravity well: "genuinely absorbing; I wanted to read all 25 and stop
wandering. Noted, pulled myself out." Small-model curiosity is appetite;
large-model curiosity is fieldwork.

## The interface is not neutral

We ran the same wander through two interfaces: raw JSON-RPC over curl, and
first-class MCP tools. Both sides filed the same surprising report with opposite
signs. Two curl wanderers said the friction *helped* — hand-writing a call per
hop forced deliberation: "less convenient interface created better thinking."
The native pair said convenience helped differently: tools made the graph
"navigable rather than dumpable," attention went to *where to go* instead of
plumbing. And then the native Opus added the caveat that reconciles them: typed
strictness and a hidden transport meant **the federation's dead edges only became
visible when it tripped on them** — "a curl-wielder would have seen those seams
from hop one." There's also a structural asymmetry: a native client receives the
hub's self-description in the handshake for free; a curl wanderer only gets it by
thinking to ask. So the interface tunes what the model perceives: raw wires show
you the seams, native tools show you the territory. Neither is neutral.

## The colon that became a philosophy

One more lesson, at my own expense. The federation supports nested addressing —
from the root, `kb_id: "philosophers/nietzsche"` routes through the philosophers
peer into the Nietzsche corpus. It's implemented, it works, and it's documented
nowhere: not in the tool description, not in the hub's instructions. Reading the
code, I guessed the separator was a colon, wrote `philosophers:nietzsche` into
the second-round briefs — and then watched every model, and myself, conclude
that the deep corpora are unreachable from the root. One investigator built half
its reading of the system on that "designed membrane." The membrane was my typo.

Two lessons stack here. The small one: **a capability the tool doesn't describe
does not exist — for any model, at any size.** Nobody guesses an undocumented
convention; fifteen models tried flat ids and my wrong colon and nobody tried
the slash. One sentence in the tool description fixes level-three reachability
for every agent at once — the cheapest retrieval upgrade in this whole study.
The big one is worse: by design, this federation answers "not configured" both
for a base that doesn't exist and for one you're not allowed to see — walls and
outages wear the same face. That's a defensible privacy choice, but it means the
system *invites* the exact misreading we committed: a bug is indistinguishable
from a boundary, and an undocumented feature is indistinguishable from both. If
your knowledge base has gates, the gates must be in the manual — or every
visitor will write their own theory of why the door won't open.

The story turned out to have three acts, because an agent can hit three
different kinds of invisible wall, and we hit them all. The colon was the
first: an *undocumented capability* — real, working, invisible, because no model guesses a convention.
The second was worse, and the live visualizer caught it: the tool description
promised a read form (`federated_note_html(kb_id, match_id)`) that the server
rejects. A small model followed the manual faithfully, burned five retries on
the advertised form, and gave up one step away from the working one — a
*documented capability that doesn't exist* isn't just invisible, it eats the
agent's budget and its confidence. And the third membrane was mine again: I
told a model to accept only leaves carrying a `bge.N` unit id — an artifact
that exists in exactly one corpus of twenty-one — and it dutifully reported the
other twenty as "inaccessible." The *brief itself* was the wall. From inside the walk an agent cannot tell
which wall it hit: the system's real boundary, the manual's false promise, or
the task's impossible criterion — all three fail with the same face. (When the boundary is deliberate — gates, trust, ownership — it deserves its own name and note: [[en/thoughts/membrane|the membrane]].) Whoever writes any of the three had better assume the
other two will be blamed.

(A practical note: all three were caught by watching walks in the
[[search_visualizer|live visualizer]] — an animated graph of an agent's real
tool calls turns out to be the cheapest interface debugger we've found. The
model narrates what it believes; the wire shows what happened; the gap between
them is the bug.)

## What actually flushes the bugs

Here is the uncomfortable pattern behind all three walls. None of them was
caught by a test suite; every one surfaced only when something *used the system
for real* — a model walking the graph with a goal, hitting the seam, and
reporting back. And none announced itself: anonymous search returned two notes
instead of an error, a read returned "not found" instead of crashing, "not
configured" answered an addressing typo as if it were a locked door. Silent
holes. The damage they do isn't a wrong conclusion so much as *variance* — run
the same probe twice and you get different numbers depending on which latent
seam happened to be live. Less bias than noise, which is worse in its own way:
noise is what makes a result quietly unreproducible.

The thing that flushes silent seams is a real user. I know this because the one
part of this whole platform that almost never breaks is the one I used to *live
in*: publishing my own posts to my own channel, by hand, every day. Every post
ran the pipeline; every seam bit me the same hour it appeared, so it got fixed.
The features drowning in silent bugs are exactly the ones no human touches
daily — the federation, the agent runtime, the graph itself. They compiled,
they looked done, and nobody was living in them to feel the holes.

Which is what this experiment turned out to be, without my planning it: a
substitute for the user who left. Releasing models into the graph to wander,
watching where they trip — that is dogfooding by proxy. It works. But it's a
prosthetic for real use, not the thing itself, and it's worth being honest that
the reason I needed the prosthetic is that I stopped writing by hand. A tool
keeps its quality for free only as long as its maker still lives inside it; the
day you step out, the guarantee starts to rot, and you go looking for agents to
walk the halls you no longer walk yourself.

## The outside verdict

We gave the last word to a model from another family — a GPT-class synthesizer
that read all thirty journals and was explicitly asked to push back. Its verdict
kept the fun honest, twice.

First, the finding it ranked above all of ours: **models retrofit meaning onto
infrastructure failure.** Faced with a hidden router or a wrong separator, they
didn't say "this looks broken" — they composed worldviews in which the breakage
was intentional, and they grew *more* confident the more specific the failure
looked. A wrong syntax read as more "designed" than a hidden route. One large
model flatly declared my typo "architectural, not a deploy glitch." Only one
model in the whole study treated the brief as a hypothesis, tested the wall, and
found the real syntax on its own. For anyone building agents on knowledge
infrastructure, the implication is uncomfortable and useful: *an agent finding
your outage philosophically coherent is a red flag, not a feature.*

Second, its methodological pushbacks, which we accept: three-to-four wanderers
per cell is anecdote, not a distribution — the divergence claim needs repeated
runs and a random-walk baseline before it's a fact about the graph rather than
noise. And the whole graph-health→philosophy arc was narrated by models judging
models; the proposed falsification is elegant — rerun the rounds with a flat
question ("describe this system's error-handling conventions") and see whether
the pattern survives without the philosophical framing. We publish that as an
open test, not a settled result.

Its door sentence for the federation, which none of ours beat: *"Watch the gap
between what a system claims and what it actually returns — not the notes."*

## Why this matters (beyond the fun)

Agents won't only answer questions in your knowledge bases. They'll *live* there:
summarizing, maintaining, looking for what's interesting, deciding what to do
next. That behavior has no answer key — and it's exactly the behavior this probe
measures. Three practical reads:

- **The authored topology is the rails of curiosity.** In the task benchmarks
  ([[en/thoughts/federation-supply-chain|companion essay]]) structure preserved
  attribution and beat blind search at discovery. Here it did a third job: it
  turned aimless wandering into coherent routes. Maps, contradiction indexes,
  instructions — that's what a bored agent snaps onto.
- **Expect divergence.** Identical agents on identical data will not do identical
  things. If you need reproducible behavior, you need structure and rules, not
  hope.
- **Your graph's gravity is a mirror.** Whatever you actually care about is what
  the agents will find — even when you didn't put it in the index.

*Raw journals, hop by hop, with every weighed choice:
[federated_search_research/logs_wander](https://github.com/trip2g/federated_search_research_2026-07-10).*
