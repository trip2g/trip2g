---
title: "A federation of gardens"
free: true
lang: en
lang_redirect: "[[ru/thoughts/federation-of-gardens]]"
---

# A federation of gardens

**The short version.** By the criteria of the movement's own historian, trip2g.com
is a digital garden: topography instead of a timeline, notes that keep growing
after publication, imperfect work in public, on infrastructure I own. That part
is a self-assessment, and I'll flag where it flatters us. The more interesting
part is what the 2020 garden wave never built: gardens stayed islands, and the
streams won partly because of it. Federation — one protocol, walls between bases,
agents as visitors — is the "between gardens" layer that history points toward
and stops short of.

## The checklist, honestly

Maggie Appleton's
[A Brief History & Ethos of the Digital Garden](https://maggieappleton.com/garden-history)
traces the idea from Mark Bernstein's *Hypertext Gardens* (1998) through Mike
Caulfield's garden-and-stream talk (2015) to the 2020 wave, and distills it into
patterns: topography over timelines, continuous growth, imperfection in public,
personal and experimental, diverse media, independent ownership. I went through
the list against the actual site, not the pitch.

**Topography over timelines.** The thoughts section has no dates and no feed.
`What is a knowledge graph` links out to `three_zones`, `why_not_book`,
`digital_sovereignty`, `contentless_cms` — you enter anywhere and follow the
links, which is the note's own thesis demonstrated by its markup. And Caulfield's
garden/stream split isn't a metaphor here; it's literally the site map. The
changelog (`en/changelog`) is the stream — dated releases, newest on top. The
notes are the garden — undated, revisable, linked by association. Same vault,
two temporal regimes.

**Imperfection in public.** This is where the evidence embarrassed me a little,
in the right way. The site serves `plans/2026-04-27-mcp-federation.md` — a raw
implementation plan, stage-1 MVP scoping and all. It serves a design doc for
frontmatter patches with the implementation plan still in it. The roadmap page
says, in its own subtitle, "what works, what's partial, what's planned" — which
is Appleton's epistemic disclosure done as a feature list. None of this was
published to look like gardening. It's just what falls out when the site *is*
the working vault: there is no export step at which you'd sweep the rough parts
under the rug.

**Continuous growth, owned soil.** Every note is a Markdown file in a vault I
control, on a server I run, and the two research essays in this series were
revised in public as the runs broke — the broken runs stayed in the repo. The
bilingual pairs (`en/user/federation.md` / `ru/user/federation.md`, each pointing
at its twin) are the same garden bed grown in two languages, which I haven't
seen on Appleton's list but feels native to the ethos: one idea, tended twice.

What *doesn't* fit: no growth-stage markers, no seedling/evergreen epistemology
on individual notes, and most thoughts pieces read more finished than a garden
note should. We disclose status at the roadmap level, not the note level. Fair
is fair.

## The part that isn't a garden

Here's the tension I'd rather hold than hide. The hub at `/en/hub` federates
about twenty-five knowledge bases, and most of them — the twenty-two philosopher
corpora — were not grown. They were *processed*: a pipeline segmented the books,
extracted concepts, planted the wikilinks, at industrial scale and machine
speed. When we released bored models into the graph
([[en/thoughts/let-the-model-wander]]), one of them looked around and read the
place not as a garden but as a system for refining thought — and it wasn't
wrong. A garden grows; a refinery processes. Appleton's ethos is personal, slow,
hand-tended, a little chaotic. A corpus stamped out by a recipe is none of those
things, however well-made. Calling the whole site a garden would be claiming
the wrong virtue.

But this is exactly where the federation framing earns its keep. The hub doesn't
pretend the beds are one garden. Each base keeps its own `kb_id`, its own owner,
its own boundary — the hand-grown journal next to the machine-built Nietzsche
corpus, walls intact. The honest description isn't "a garden" but "a garden
*district*": some plots tended by hand, some planted by machine, and the map
tells you which is which. The dishonest version would be the flat pile — where
everything blends and provenance evaporates, which is precisely the failure mode
the [[en/thoughts/federation-supply-chain]] experiments measured.

There's a sharper version of this criticism, and it came from the models
themselves. When we asked two large models to read the federation's *philosophy*
from its artifacts, both noticed the same thing about the planting: the current
shelf is Stoics, moralists and success literature — thinkers who slice cleanly
into principles and A→B links — and it has no Plato, no Kant, no women, no
critical tradition. One of them called it a *monoculture of the extractable*: the
harvester shapes the field, and whatever resists being cut into reusable units
simply doesn't get planted. That's a fair hit, and I'd rather own it than argue.
Two honest replies. First: the recipe is the monoculture, not the protocol —
nothing in the federation requires a corpus to speak in principle-and-chain
grammar; that's one base's build method, and a dialogue-shaped or system-shaped
corpus can federate with its own internal form. Second: this is an open district.
We'll plant some of the missing beds ourselves — and the rest is an invitation.
Bring your own corpus: your Kant, your Arendt, your tradition, grown in your
format, on your soil. The gate protocol is the whole point — we'd rather federate
your garden than approximate it.

## Gardens with gates

Appleton notes, almost in passing, that gardens remain "rather solo affairs" —
kept on the open web in the hope of future interconnection that hadn't arrived.
That line is the loose thread of the whole history. The 2015-2020 wave rebuilt
the personal wiki beautifully and never built the *between*. Gardens didn't cite
each other's beds; RSS withered; there was no way to let a visitor — human or
otherwise — walk your garden without handing them the keys or dumping your notes
into someone's silo. So the streams kept winning, because streams at least
connected people, even if they connected them into mush.

Federation is my attempt at the missing layer, and it's small on purpose: one
MCP endpoint per garden, a hub that routes by `kb_id`, walls that keep every
retrieved passage attached to the base it came from, and the option to expose a
subgraph — a few beds, not the whole plot — to a peer or an agent. A visitor can
search my garden, read a note, follow its links, and every fruit they carry out
keeps the name of the bed it grew in. That's the attribution result from the
supply-chain essay wearing garden clothes: models navigate a merged pile just
fine; what they lose is *whose idea this was*. The wall is not a better search
box. It's the property line that makes visiting possible at all.

And the visitors are already stranger than the 2020 gardeners imagined. When we
let models wander the graph with no task, they didn't skim — they diverged,
formed theories about the place, and got pulled to whatever the gardener
actually cared about. A garden used to be something you kept for yourself and
the occasional human reader. Now it's terrain that agents inhabit, and the
gates matter more, not less.

A garden feeds the person who keeps it. A federation of gardens can feed a
community — neighbors, partners, agents passing through — on one condition,
the same one the whole research series kept converging on:
every fruit keeps the name of its garden.
