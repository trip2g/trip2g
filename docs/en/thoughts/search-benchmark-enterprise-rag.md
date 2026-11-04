---
title: "The hybrid search that wasn't hybrid"
free: true
lang_redirect: "[[ru/thoughts/search-benchmark-enterprise-rag]]"
---

*We ran trip2g's search through a corporate benchmark of 500 questions and found the bug that had kept it at half strength. The fix took the share of questions with the right document in the top ten from 69% to 84%. We also tested whether trip2g can hold a large knowledge base: settings improve memory and startup, but they are not on by default yet.*

For a year I kept finding odd misses in search. You ask in plain language, the document is definitely there, and search doesn't see it. Each case looked minor, and no pattern emerged. To learn whether a company could trust trip2g with its knowledge, we needed a fair exam, not one more hand-picked example.

## The exam

[EnterpriseRAG-Bench](https://onyx.app/enterpriserag-bench) comes from a company called Onyx. It is a made-up company, Redwood Inference, with all of its working chatter and documents: almost 512 thousand files from Slack, email, Jira, Confluence, GitHub, a CRM and meeting transcripts. It has internal code names, outdated pages and nearly identical drafts, like a real company.

The archive comes with 500 questions and correct answers. Some are simple, some avoid the words used in the right document, and for some the answer isn't in the archive at all.

An exam like this matters because RAG answers only from what search found. If the right document didn't surface, the smartest model will miss or confidently make something up. It all starts with search.

We didn't use the whole archive. This run took all 722 documents that hold answers and 50 thousand random ones, about one file in ten. So it can't be compared with the benchmark's leaderboard: the task got easier, because most of the near-identical documents dropped out. The full archive would score lower.

## What the first run showed

| Questions | Right document in the top ten |
|---|---|
| All | 69% |
| Simple | 79% |
| In the asker's own words | 38% |

Questions in the asker's own words were a disaster. That is how people really ask, and search missed six times out of ten. For another 109 of 470 questions the right document wasn't even in the top twenty.

## What was wrong

Two searches work together in trip2g: one by exact words, one by meaning. Their results are merged, so each covers the other's weak spots.

The test showed that the word search barely worked. It required every word of the question to be in the document. A long question almost never meets that: only 59 of 500 questions got any result from it. The rest never reached it, and the whole hybrid rested on the meaning search.

On corporate data the meaning search often misses. Ask about Northbridge Bank and log retention, and it brings another document on the same topic, while the right runbook sits in 75th place. Code names and client names mean little to it, and the word search finds them easily.

A second bug hid behind the first. The title counted separately from the text, and a one-word title looked like a perfect match. In Slack the title is often the channel name, so any question about incidents lifted the `incidents` channel to the top.

## What changed

- A document now needs half of the question's words, not all of them.
- The title counts together with the text.
- Function words like "is" and "как" no longer take part in search and aren't highlighted.

No new components, just a change in search.

## The result

| 50 thousand documents, 500 questions | Before | After |
|---|---|---|
| Right document in the top ten | 69% | **84%** |
| Same, questions in the asker's own words | 38% | **62%** |
| Same, simple questions | 79% | **93%** |
| Right document in first place | 43% | **57%** |
| Questions with the right document outside the top twenty | 109 | **45** |

We also checked real notes, on the [[en/thoughts/search-benchmark-obsidian-help|Obsidian help docs]]. Russian search grew noticeably: the quality score went from 0.689 to 0.751. English grew less: from 0.763 to 0.784.

The fix isn't free. Queries with typos and tricky morphology got slightly worse. On the old bilingual set of 48 notes the order of results got worse, the score fell from 0.926 to 0.879, though it still finds the right documents. Plain "any word will do" crashed that score to 0.626, which is why the bar sits at half. We tuned that bar on these same sets, so the numbers lean a bit optimistic. We didn't separately measure a Russian question whose answer exists only in an English note.

## What about performance

In short: we found some of the problems and know how to work around them, but almost nothing changed by default.

| What | Before | Possible now | On by default |
|---|---|---|---|
| Peak memory at 50 thousand | 23.6 GB | 8–9 GB with the index on disk | No |
| Server start | The index is rebuilt on every start: 204 seconds already at 13 thousand notes | With the index on disk at 50 thousand: 8 minutes the first time, 33 seconds after that | No |
| Preparing notes for meaning search | 3 notes a second | 9 notes a second with the new queue settings | No |
| Database size limit | Upload stopped at 14 thousand notes | The limit can be lifted in settings | No |
| Search time, half of queries are faster | 1.03 seconds | 1.18 seconds | Slightly slower |

Memory was the main finding. The in-memory index took 85% of the memory the program held. Taken straight, the full archive would need about 240 GB, but that's an estimate, not a measurement. With the index on disk the same notes fit in a third of the memory.

Three things are still untested: the full archive of 512 thousand files, behavior under real load, and the quality of the model's answers. We measured only whether search finds the right document.

## Is it ready for a company's knowledge base

Not yet. It went from 69% to 84%, and that's on the easier version of the exam. So about one document in six still misses the top ten, and more than one in three for questions in the asker's own words.

Search can't fix part of that. "Which team got the most follow-up tasks across all incident reviews" has no single document with the answer: that takes counting, not finding. Another part can be fixed: long stretches of text get cut badly, and in sources like meeting transcripts the pieces run many times over the intended size. That's the next task.

The whole run cost about a dollar and a half in embeddings. For that money we learned more about the bug than in a year of chasing single cases.
