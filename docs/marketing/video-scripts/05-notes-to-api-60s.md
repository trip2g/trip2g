# Script 05: From notes to API in 60 seconds

**Length:** 70 seconds  
**Target viewer:** Developer or technical user who wants to point an agent at their local notes without setting up a RAG pipeline  
**One idea:** `mcp:add` your vault and your agent can answer questions grounded in it. No pipeline, no upload, one command.

---

## Shot list note
Screen recording only. Terminal left half, agent chat right half. Run the command, watch the agent's first grounded answer appear. Keep the pacing tight. The whole thing should feel fast.

---

## Script

| Time | VISUAL / ON-SCREEN | VOICEOVER |
|------|--------------------|-----------|
| 0:00 | Terminal. Empty prompt. | "I have a folder of notes: research, meeting logs, project decisions. My agent has never seen any of it. That's about to change." |
| 0:06 | Type: `mcp:add ~/notes/research --name my-research` then hit enter. Output: `scanning... 847 files indexed. MCP endpoint ready at localhost:3701`. | "One command. trip2g scans the folder and exposes it as an MCP endpoint. 847 notes, indexed." |
| 0:16 | Switch to agent chat (Claude or Cursor). Type: `"What's my current hypothesis on the attention mechanism bottleneck?"` | "Now I ask my agent something I wrote down three weeks ago." |
| 0:22 | Agent response appears: grounded answer, cites `notes/research/attention-bottleneck-v2.md`, quotes a specific line. | "It found it. Quoted the exact line. Cited the file." |
| 0:30 | Type another question: `"What did I conclude about the dataset size trade-off?"` Answer comes back citing two files. | "Another question. Two files this time, pulling from both." |
| 0:38 | Show terminal: `localhost:3701`, no external requests in the network log. | "Nothing left the machine. No OpenAI key, no cloud vector store, no data sent anywhere. The endpoint is local." |
| 0:46 | Text on screen: `no pipeline · no upload · no vendor lock-in` | "Most RAG setups ask you to chunk your notes, embed them, upload them to a service, and maintain a sync job. This is a folder and one command." |
| 0:54 | Quick montage: `mcp:add` three more folders: `~/notes/clients`, `~/notes/meetings`, `~/notes/books`. Each one indexes in seconds. | "Add as many as you want. Each one becomes a separate base with its own access rules." |
| 1:02 | End card. trip2g logo. `trip2g.com` | "Self-host it: MIT license. Or try the sandbox at trip2g.com." |

---

**CTA:** `trip2g.com` (self-host or try the sandbox); `mcp:add` your first vault in under a minute
