# Script 03: Watch your agents work

**Length:** 85 seconds  
**Target viewer:** Developer or power user who runs agents against their knowledge bases and has no visibility into what the agent actually reads  
**One idea:** Seeing your agent work in real time is a control feature, not a spectacle. You can stop a mistake before it lands.

---

## Shot list note
Browser screen recording: trip2g's live activity view showing multiple open bases. Agent is mid-run. Show it navigating files, reading, then making an edit. Show the human stepping in and stopping it. No cuts to face cam. The UI tells the story.

---

## Script

| Time | VISUAL / ON-SCREEN | VOICEOVER |
|------|--------------------|-----------|
| 0:00 | Browser: trip2g open, three bases visible in sidebar: `personal`, `work`, `reading-group`. An agent task is running. | "Three bases open. An agent is running against all of them. Here's what it's actually doing." |
| 0:07 | Live trace panel on screen: `reading: personal/projects/q3-plan.md` → `reading: reading-group/shared/annotated-papers.md` → cursor moves. | "It's reading my Q3 plan. Now it's in the shared reading group folder. I can see exactly which file it's in." |
| 0:15 | Agent opens a new note, starts writing. Title: `q3-synthesis.md`. First line appears live. | "Now it's writing a synthesis note. I can watch the lines appear." |
| 0:22 | User pauses, points to a line on screen. The agent wrote something wrong: a date from the wrong project. | "There. That date is from the wrong project. I can see that right now, before it saves." |
| 0:28 | User clicks "pause agent" button. Agent stops mid-sentence. User edits the line. Clicks "resume". | "Pause. Fix it. Resume. The mistake never made it to disk." |
| 0:36 | Text on screen: `this isn't a demo. it's a control plane` | "Most agent setups are a black box. You fire a task, come back, and either it worked or it didn't. You debug after the fact." |
| 0:44 | Split: left shows a chat interface with a spinner, no visibility. Right shows trip2g live trace, readable and stoppable. | "trip2g makes the agent's path through your knowledge visible. Not for show. Because write access to 50,000 files is a real thing." |
| 0:54 | Zoom in: per-base access indicators. `personal` base shows a lock icon next to two subfolders. | "You set what each agent can reach. This agent has read access to the shared folder. My private research folder isn't on its path. The lock is in the trace." |
| 1:04 | Agent finishes. `q3-synthesis.md` saved. Log shows which bases it read, which it skipped. | "When it's done, the log tells you every base it touched and every base it didn't. That's the audit trail." |
| 1:12 | End card. trip2g logo. `trip2g.com` | "Self-host: trip2g.com. MIT license." |

---

**CTA:** `trip2g.com` (self-host or try the sandbox)
