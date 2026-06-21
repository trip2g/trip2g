# Script 02: Git-conflict horror to trip2g fix

**Length:** 80 seconds  
**Target viewer:** Obsidian-Git user who has hit merge conflicts on system files; knows the pain  
**One idea:** Git guarantees conflicts on files you never touched. trip2g doesn't use Git for sync.

---

## Shot list note
Two-panel screen recording. Left: a terminal showing a Git merge conflict in `workspace.json`. Right: the same edit being made in trip2g with no conflict. Real files, real terminal output. No mockups.

---

## Script

| Time | VISUAL / ON-SCREEN | VOICEOVER |
|------|--------------------|-----------|
| 0:00 | Terminal. `git pull`. Output scrolling. Then: `CONFLICT (content): Merge conflict in .obsidian/workspace.json`. Red text. | "This is a conflict in `workspace.json`. That file records which pane you had open." |
| 0:07 | Zoom in on the conflict markers: `<<<<<<< HEAD`, `=======`, `>>>>>>> origin/main`. The diff is two different `"active"` values. | "Nobody wrote prose here. Two devices recorded their window state at the same time. Git can't merge that automatically." |
| 0:15 | Text on screen: `guaranteed conflict: every time two people sync` | "This isn't a configuration mistake. It's structural. Git was built for deliberate, reviewed commits, not real-time note-taking." |
| 0:22 | Cut to: same vault, trip2g running. Person B opens a note and edits a heading. | "Same vault, trip2g connected. Person B edits the same note." |
| 0:28 | Person A's Obsidian: the change appears. No conflict prompt. No merge screen. | "Person A sees it. No conflict. No merge screen." |
| 0:34 | Terminal on trip2g side: clean log lines. `sync: note updated by peer`. | "trip2g doesn't route your whole vault through Git. It pushes the content changes over a direct socket connection." |
| 0:42 | Show file manager: `.obsidian/workspace.json`, unchanged, no conflict markers, no staged changes. | "`workspace.json` never went near a commit. Neither did any plugin state file. Those stay local." |
| 0:50 | Split screen: left shows the Git terminal with three unresolved conflicts stacked up; right shows trip2g log, clean. | "The alternative is fifteen minutes resolving merges on files no human was supposed to touch. Every session." |
| 1:02 | Text on screen: `open source · self-hosted · one binary` | "trip2g is open source, MIT license. One binary. Your vault stays on your machine." |
| 1:08 | End card. trip2g logo. URL. | "Try it: trip2g.com." |

---

**CTA:** `trip2g.com` (self-host or try the sandbox)
