---
title: "What a markdown vault is"
free: true
lang_redirect: "[[ru/thoughts/markdown-vault]]"
---

A vault is a folder of markdown notes you navigate as a linked graph. The same thing Obsidian users call a "vault." The one difference: ours is **not tied to Obsidian**.

## The format, not the app

A vault is just markdown files. No Obsidian, no plugins, no lock-in. You can open, read, and edit it with anything: Obsidian, VS Code, Vim, a script. The contract is the format (markdown plus wiki-links), not a particular editor. Obsidian is a great editor, but a vault should not depend on one app and its plugins to be read, served, or moved.

## An ephemeral thing

To trip2g a vault is a portable, reconstructable unit, not files nailed to one machine. A vault can live on a disposable node, sit as a copy in object storage, be restored anywhere, and be federated to a trusted peer. The node is just where the vault happens to run right now; the knowledge lives separately (see [[en/thoughts/surviving-node-loss|on preemptible nodes]]). Kill the node and the vault reassembles on another.

## Hypertext, not just a folder

A vault is not a flat folder: wiki-links `[[like this]]` turn it into **hypertext**, a graph of linked notes you traverse by following links rather than reading top to bottom (see [[en/thoughts/obsidian-and-wikilinks|wiki-links]] and [[en/thoughts/knowledge-graph|the knowledge graph]]). Markdown is the text layer; hypertext is the layer of links over it.

When a vault is written as an **index-first wiki read not by a human in a chat but by an agent over MCP** (search / instructions), it becomes an [[en/user/llm-wiki|LLM Wiki]]: a knowledge artifact the agent navigates, not a transcript. The same folder of markdown, just addressed to an agent.

## The unit of knowledge

A vault can be anything: personal notes, a role's knowledge base (what a video-editor or sales agent knows), a department hub, a company hub. Always the same simple thing: a folder of markdown you moderate and move yourself.
