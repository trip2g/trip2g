---
title: "One writer per base: a Confluence alternative without a permission matrix"
description: "Why every knowledge base in trip2g has exactly one writer, and how a team space is assembled from federated bases instead of a permission matrix over one shared wiki. Notion and Confluence permissions, Federated Wiki, and where the idea falls short."
free: true
lang: en
lang_redirect: "[[ru/thoughts/one-writer-per-base]]"
---

# One writer per base

**In short.** trip2g publishes Obsidian vaults as websites and serves them to AI agents over MCP. This essay is about using it as a Confluence alternative with no permission matrix. Every knowledge base in it has one writer: its owner, a person or that person's agent. Everyone else reads. A team space is several such bases, each with its own owner, and one endpoint that searches across all of them. That is what federation means here. Most of the permission settings you fight in Notion or Confluence then have nothing left to decide. Who may edit a note follows from whose base it is. If you find a mistake in someone else's note, you message the owner instead of editing their base. This is a single-writer model for a federated knowledge base. It has a precedent that did not take off, and it costs something real. Both are below.

## How shared wikis rot

Every team wiki I have used started the same way. One space, everybody can edit, nobody thinks about access. Then a contractor joins, and a page about salaries needs hiding. Then a client gets a guest account. Then one subpage needs to be stricter than its parent.

The tools let you do all of this, and that is the trouble. In Notion [a subpage takes the permissions of its parent](https://www.notion.com/help/sharing-and-permissions), and you can go into the subpage and override them. There are six access levels, from Full access down to Can view, two of them only for databases, plus guests from outside the workspace. Every override is a decision someone made once and nobody remembers making.

Atlassian measures where this ends. Its own support page on [Confluence spaces with too many permissions](https://support.atlassian.com/portfolio-insights/docs/confluence-spaces-with-too-many-permissions/) sets a guardrail at 20,000 permissions per space, counts "every individual permission granted to a user or group", and warns that past it page tree navigation, viewing and editing all slow down. Nobody designs twenty thousand permissions into one space; they pile up one reasonable request at a time.

The advice that comes with it is telling. Atlassian suggests groups instead of individual users, new, smaller spaces per team or project, space owners, and periodic cleanup of stale content. Caroline Weber at Honicon, in [Sustainable permission models for Confluence](https://honicon.com/en/?p=12153), describes the state before the cleanup: "a confusing tangle of permissions: too many administrators, unclear responsibilities", where "no one is quite sure who actually has access to what". Her fix is a few clear groups and dedicated spaces instead of page-level restrictions.

And the content rots in parallel. Rencore's [guide to Confluence sprawl](https://hub.rencore.com/guides/govern/confluence/sprawl) is a list of detectors: pages not updated in 365 days, spaces with no activity in 365 days, orphan pages with no parent, empty spaces, labels used once. Every one of those is something that belongs to everybody and therefore to nobody.

Read the remedies together and they point one way: smaller spaces, each with an owner, with people grouped by what they may do. That is a step toward one owner per base. The tools stop halfway, because the shared pile is the product.

## The rule, and what it removes

So I went the rest of the way. A base has one writer. In trip2g that means one vault: notes reach the server from its owner's Obsidian, a git push, the in-browser editor, or an agent holding the owner's API key. Readers can read, and the only thing they can send back is a form submission if the owner put a form on a note. A subscriber's token inherits what that subscriber may read, and the MCP tools a reader can call (`search`, `note_html`, `expand`, `similar`) only read.

A team space is then put together from parts. The platform team keeps its base. HR keeps its own. A partner keeps theirs on their own server. My hub connects to each one through a KB-note, an ordinary note whose frontmatter names the peer's address and whose body says when to use it. The agent asks one endpoint and the hub fans the question out ([[en/user/federation|how federation works]]).

Now look at what is left to configure. The edit column of the matrix is gone: there is one writer per base, so there is nobody to grant edit rights to. What remains is read access, and it is coarse on purpose. The owner groups notes into [[en/user/subgraphs|subgraphs]] and grants each peer key a set of them. A new key starts with access to nothing. Revoking it is one button, and the peer's next call gets a 401.

Across bases the wall is not a setting someone can get wrong in a hurry. Every federated tool reads: `federated_search`, `federated_note_html`, `federated_expand`, `federated_similar`, `federated_instructions`. Even the GraphQL pass-through to a peer validates the query and rejects anything that is not a read. There is no way to write into my base from yours, because no such call exists. Asking "who can edit the platform team's runbook?" has one answer, and you don't need to open a settings page to get it.

"Who is responsible for this page?" gets an answer too. The page lives in someone's base. That someone is responsible. If they leave, their base goes stale as a whole and visibly, instead of leaving five hundred orphans scattered through a shared space.

## A fix is a message

The part people push back on: what if I see a mistake in your note?

Then you tell me. You don't edit my note. In a shared wiki the drive-by fix is the best feature: you see a wrong port number, you change it, done. I am giving that up deliberately, and I want to say plainly that it is a loss. A typo now costs a message and a wait.

What it buys is that my base only says what I said. When an agent quotes the platform runbook, the runbook's owner wrote that sentence. Nobody slipped an edit in while the owner was on holiday. I measured why this matters for agents in [[en/thoughts/federation-supply-chain]]: a model searching a merged pile finds the right material and then attaches it to the wrong author. A base that has one writer keeps the name on the answer.

How the message travels is not settled. Today it is whatever channel you already use with the owner. The closest thing the platform ships is a [[en/user/forms|form on a note]], which stores submissions against that note. I would not call that a review workflow. If you disagree with my note rather than correct it, the honest move is to write your own note in your own base and link to mine. Then both views exist, each under its author's name.

## This was tried in 2011

The idea is not mine. Ward Cunningham, who invented the wiki, presented the [Federated Wiki](https://en.wikipedia.org/wiki/Federated_Wiki) at IndieWebCamp in 2011. Each person runs their own wiki site. You cannot edit someone else's page; you fork it onto your site and change your copy. Jon Udell [described it in InfoWorld in 2015](https://www.infoworld.com/article/2243053/wiki-creator-reinvents-collaboration-again.html): "If I make changes, you may or may not choose to merge those changes back into your site." A conventional wiki, in Mike Caulfield's phrase that Udell quotes, is "a relentless consensus engine". Cunningham's name for the alternative was "a chorus of voices".

It never went mainstream, and the reasons are worth reading before you copy the idea. You needed to run your own server. It was hard to use: one user wrote in 2018 that it ["simply did not function for me"](https://www.kickscondor.com/comments/federated-wiki-and-directories) and called fedwiki.org "really neat but difficult to use". And discovery across sites was thin. Search ran over your current "neighborhood", the sites you had come across in that session, and [Udell explained](https://blog.jonudell.net/2015/01/02/how-federated-wiki-neighborhoods-grow-and-change/) that it "only sees the titles and first paragraphs of the pages in your neighborhood". Merging was manual too: a fork stayed a separate copy until its original owner chose to pull the changes back.

Here is what trip2g does differently on each point, and where it is the same.

**Discovery.** The list of peers is not a browsing session. It is a set of KB-notes in your vault, so it persists, and each one carries a description written for the reader: "Use when: … Don't use when: …". The agent reads that description in search results and decides whether to route there. `federated_search` runs each peer's own search over its whole notes, not over a list of titles, and without a `kb_id` it fans out to every peer you may reach. `federated_instructions` hands the agent the peer's own guidance before it searches. In Cunningham's day the reader was a person clicking between sites. Now it is often an agent over MCP, and an agent is willing to read twenty descriptions and make five calls.

**Merging.** trip2g has no forks at all. You read my base in place, and what you carry out keeps my base's name. I dropped the copy-and-merge machinery rather than solving it. That costs the chorus: Federated Wiki let ten slightly different versions of a page coexist. Here you get one version per owner and a link from your note to mine.

**Running a server.** Still true. Your base is an instance you or someone you trust runs. That is the [[en/thoughts/digital-sovereignty|point of the exercise]], and it is also the barrier Federated Wiki hit.

## Where this is weaker than it sounds

Inside one instance the rule is only a convention. An API key gives admin-level access to the whole instance. You can sync three repositories into one instance with `--meta subgraph=…` ([[en/user/advanced|docs]]), and nothing stops two people from sharing one key. The single writer is enforced by topology only between instances, where the peer can do nothing but read. If you want the guarantee, give each owner their own instance. If one instance is enough, you have a shared wiki with good manners.

Some work is genuinely joint. A spec two people write in the same afternoon, meeting notes, a decision log for a project with no single owner. In this model that document lives in someone's base, and the other person sends comments. It is slower than co-editing, and for those documents a shared editor is the better tool.

And scale. I run this on my own bases and on a small public hub that federates a few others. I have not watched a five-hundred-person company try it, so I can't tell you how the message-to-the-owner habit survives there. My guess is that it is cheaper than a permission audit. That is a guess.

The [[en/thoughts/membrane|membrane]] note covers one more cost: a federated refusal looks the same whether the base doesn't exist or you weren't let in. From outside you see less than in a shared wiki, by design.

The trade, stated once: you lose the drive-by edit and the shared draft. You get bases that each say exactly what their owner said, permissions that can be read in one line, and an answer to "who is responsible?" that you can find without asking around. [[en/thoughts/data-mesh-for-knowledge]] looks at the same move from the data engineering side, where it has a name and several years of criticism.
