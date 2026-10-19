---
title: "Data mesh for documentation: a federated knowledge base in four principles"
description: "Zhamak Dehghani's four data mesh principles in plain words, mapped onto knowledge bases in trip2g: domain ownership, knowledge as a product, a self-serve platform, federated governance. Plus Backstage TechDocs, the known criticism, and where the analogy flatters."
free: true
lang: en
lang_redirect: "[[ru/thoughts/data-mesh-for-knowledge]]"
---

# Data mesh for knowledge

**In short.** trip2g publishes Obsidian vaults as websites and lets AI agents search across many knowledge bases over MCP. The problem it deals with, one shared pile that nobody owns, is the same one data engineering hit with the central data lake, a few years before the rest of us. Its answer is called data mesh: stop pouring everything into one central lake, let each domain own and publish its own data, and hold it together with a shared platform and a few standards everybody follows. A federated knowledge base is the same move applied to documentation and notes. Below are the four data mesh principles in plain words, what each one looks like in trip2g today, the criticism data mesh has earned, and the places where my analogy is kinder to trip2g than the facts.

## Where data mesh comes from

Zhamak Dehghani, then at Thoughtworks, published [How to Move Beyond a Monolithic Data Lake to a Distributed Data Mesh](https://martinfowler.com/articles/data-monolith-to-mesh.html) on martinfowler.com in May 2019. Her diagnosis of the central data lake was blunt: "over promised and under realized". A central team owned the pipelines and sat far from the domains that produced the data, so the people who knew the data were not the people who looked after it.

In December 2020 she named [four principles](https://martinfowler.com/articles/data-mesh-principles.html), and in 2022 O'Reilly published her book, *Data Mesh: Delivering Data-Driven Value at Scale*. Swap "dataset" for "knowledge base" and many paragraphs of the first article reads as a description of a company wiki.

## The four principles, in plain words

**1. Domain ownership.** Dehghani's wording: "decentralization and distribution of responsibility to people who are closest to the data". The team that produces something looks after it. Not a central data team, not a wiki gardener.

**2. Data as a product.** Data is "treated as a product, and the consumers of that data should be treated as customers". The 2019 article lists what that means: a data product is discoverable, addressable, trustworthy, self-describing, interoperable and secure.

**3. Self-serve platform.** A shared platform gives every domain "a high-level abstraction of infrastructure" so that a team can publish a data product without building storage and pipelines from scratch.

**4. Federated computational governance.** Some decisions are local to each domain, some are global. The global ones are agreed by a governance group made of domain and platform people, and then the platform enforces them automatically instead of relying on everyone remembering. The point of the global part is "interoperability and a compounding network effect through discovery and composition of data products".

## The same four, for knowledge bases

**Ownership** is the rule from [[en/thoughts/one-writer-per-base]]: each base has one writer, whoever actually knows the subject. The platform team writes the platform base, HR writes HR's. The [[en/thoughts/federation-supply-chain|supply-chain essay]] gives the reason in numbers: on one merged pile, models found the right ideas and attached them to the wrong author.

**Knowledge as a product.** Going through Dehghani's list against what trip2g ships:

- *Discoverable.* A base is listed through a KB-note, an ordinary note whose body tells the agent "Use when: … Don't use when: …". The agent reads it in search results and decides whether to go there. The [[en/user/hub|public hub]] is a page of such notes.
- *Addressable.* Each base has one address, `/_system/mcp`, and a short `kb_id` that a query can target.
- *Self-describing.* A base can publish author instructions (`instructions()`, and `federated_instructions(kb_id)` from the other side), and each subgraph carries a one-line "What this subgraph is" that a peer reads beside the name.
- *Interoperable.* One protocol, MCP, with the same handful of methods on every base. What runs behind it is the owner's business: a peer only has to answer those methods, so it can be another trip2g instance, an adapter in front of some other store, or a different search engine.
- *Secure.* A peer key starts with access to nothing, the owner grants it [[en/user/subgraphs|subgraphs]], and revoking it takes one button.
- *Trustworthy.* This is the weak one, see below.

The product framing is also literal here: a base can be sold by subscription, which is the [[en/thoughts/knowledge-as-a-service]] idea.

**Self-serve platform.** One trip2g instance is the whole platform for one base: a single binary on SQLite, sync from Obsidian or a git push, hybrid search, an MCP endpoint and a website out of the box. A team does not ask a central group for a pipeline. It runs an instance, or someone runs one for it.

**Federated governance.** What is global is what the code enforces on every call between bases. The set of federated methods, all of them reads. Signed requests with a shared key. The same "not configured" answer for a base that doesn't exist and one you may not see. Key rotation with a five-minute grace window. Then a couple of defaults each instance may change: a fan-out depth cap of 3 so peers with peers don't loop, and a two-second timeout per peer. What is local is everything else: what the base contains, how notes are structured, which subgraphs exist, who gets which ones.

The difference from Dehghani's version is who sets the global rules. In data mesh a governance group of the domains decides them. Here they are whatever [[en/user/federation|the federation code]] does, and the only group that decided them is whoever wrote that code, me. For a protocol between strangers that is probably right. For a company it means the shared rules stop at access and routing, which comes back below.

## A precedent from documentation: Backstage TechDocs

Data mesh is about data, so here is a precedent from documentation itself. (An older one, Ward Cunningham's Federated Wiki, is in [[en/thoughts/one-writer-per-base]].) Spotify's [TechDocs](https://backstage.io/docs/features/techdocs/), built into Backstage, is "docs-like-code": engineers write Markdown in the same repository as the service, and Backstage renders it and makes it findable from the service's page in the catalog. Spotify reports more than 5,000 documentation sites on it.

That is ownership plus a portal. The team owns its docs because they live in its repository, and the portal pulls them together for readers. trip2g can do exactly this inside one instance: several repositories sync into one site, each tagged with its own subgraph through `--meta subgraph=…` ([[en/user/advanced|docs]]).

Federation differs in where the pulling-together happens. TechDocs builds every team's docs into one portal. A trip2g hub doesn't copy anything: it holds routing notes, sends the question out at read time, and each answer comes back from the base that owns it. The portal approach is easier to search and to style. Federation keeps each base on its owner's server with its owner's access rules, which matters once one of the bases belongs to a different company.

## The criticism, which applies here too

Data mesh has had a rough few years. In its 2022 Hype Cycle for Data Management, Gartner placed it at the innovation trigger and labelled it "obsolete before plateau", with low market penetration. The report is paywalled; I am going by [Atlan's summary](https://atlan.com/gartner-data-mesh), which also cites Gartner's 2021 Data and Analytics Governance Survey: only 18% of organisations had enough governance maturity to adopt it. Dehghani disagreed with the verdict, and the same summary records that too. The recurring complaints: every team needs data skills it may not have, domain boundaries are hard to draw in a real organisation, and the platform it needs is a large project by itself.

My short version of all that: without a strong platform and agreed standards, a mesh is just a central swamp split into twenty smaller swamps. That transfers to knowledge without edits. Give every team its own base and no habit of describing it, and you get twenty small wikis rotting in parallel, now with no shared search to find the rot. A KB-note description is a thin defense. It works only if someone writes "Use when / Don't use when" honestly and updates it when the base changes.

## Where the analogy flatters trip2g

Data mesh was designed for organisations with dozens of domains and a platform team to build the self-serve layer. trip2g today runs mostly on my own bases and a small public hub. When I write "domains" here I mean a handful of bases, not a company org chart.

*Trustworthy* has no platform-level answer. Dehghani asks a data product for a service level objective on its truthfulness. trip2g has nothing like it. One base in the hub stamps every quote with an authenticity status ([[en/thoughts/federation-supply-chain]] describes it), but that is one owner's practice, not a platform rule. Freshness is not tracked across bases either: a peer that stopped updating a year ago looks exactly like one updated this morning.

Governance is narrow. The "computational" part covers access, protocol and routing. It says nothing about content: there is no shared glossary, no common note schema, no quality check that runs across bases. In data mesh that layer is where interoperability lives. Here it is left to the owners.

The catalog is hand-written. Data mesh assumes a searchable catalog of data products with metadata. In trip2g a base becomes discoverable when someone writes a KB-note about it, and only to those who have that note.

And failure is quiet in one place. A peer that doesn't answer within two seconds is skipped and you get results from the rest. That keeps search fast and makes a slow base invisible rather than broken, which the [[en/thoughts/membrane|membrane]] note warns about from another angle.

None of this kills the analogy. It says which half of data mesh trip2g has built: ownership, addressing, access and a protocol everyone speaks. The other half, shared standards for what is inside a base and how good it is, is still on the owners. If you adopt the pattern, that half is your work.
