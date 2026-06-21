# Script 04: Share a slice

**Length:** 75 seconds  
**Target viewer:** Obsidian user collaborating with a partner, colleague, or client. Needs selective sharing without giving up the whole vault.  
**One idea:** Give a partner's agent access to one folder and revoke it on camera. The rest of your vault stays air-gapped.

---

## Shot list note
Screen recording: trip2g access control panel. Show the folder tree, grant access to one folder, show the partner's agent querying it successfully, then revoke. Show the agent's next query fail cleanly. Real UI, real terminal output.

---

## Script

| Time | VISUAL / ON-SCREEN | VOICEOVER |
|------|--------------------|-----------|
| 0:00 | trip2g access panel open. Folder tree on left: `personal/`, `research/shared-papers/`, `finance/`, `clients/acme/`. | "This is my vault. Four folders. I want my research partner's agent to see one of them, `shared-papers`, and nothing else." |
| 0:09 | Click `research/shared-papers/`. Toggle: `access: none → circle-read`. A peer node selector appears. Select `partner-node`. | "I set that folder to circle-read and select her node. That's it. The rest of the vault doesn't change." |
| 0:18 | Terminal: partner's agent runs `search "protein folding review 2024"`. Result returns from `shared-papers/`. | "Her agent can now query that folder. Gets a result back from `protein-folding-review-2024.md`." |
| 0:26 | Same agent tries `search "acme client brief"`. Response: `access denied: clients/acme/ not in peer grant`. | "It asks about my client folder. Access denied. That folder was never in the grant. The gate held." |
| 0:34 | Text on screen: `the notes didn't move · no upload · no copy` | "Her notes are on her machine. Mine are on mine. The shared-papers folder didn't get copied anywhere. The query routed through and the result came back." |
| 0:42 | Back to access panel. Click `research/shared-papers/`. Toggle back: `circle-read → none`. | "Project's over. I revoke access." |
| 0:48 | Terminal: partner's agent tries the same search again. Response: `access denied: grant revoked`. | "Her agent tries the same query. Gone." |
| 0:55 | Text: `Notion gives you all-or-nothing · trip2g gives you a folder` | "Every shared workspace I've used forces me to choose: invite someone into everything, or copy a subset by hand. This is a third option." |
| 1:03 | End card. trip2g logo. `trip2g.com` | "Open source. Self-hosted. trip2g.com." |

---

**CTA:** `trip2g.com` (self-host or try the sandbox)
