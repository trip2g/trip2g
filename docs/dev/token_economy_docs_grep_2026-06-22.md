# Token-economy baseline: naive grep approach vs MCP retrieval

**Date:** 2026-06-22

## Methodology

For each question, we simulated what a coding agent without MCP does:
1. Run `grep -rl <terms> docs/ --include="*.md"` to locate candidate files.
2. Open the primary file(s) a reasonable agent would pick from those results. Where
   `docs/en/user/` contained a dedicated user doc AND `docs/dev/` had a dev doc on the
   same topic, we counted both — an agent cannot tell from the filename alone which one
   holds the answer, so it reads both.
3. Token cost = grep output tokens + all file tokens read before a confident answer.
4. Tool call count = every grep call + every file read until confident answer.

**Tokenizer** (same definition used in MCP benchmark for comparability):

```bash
python3 -c "import re,sys; print(len(re.findall(r'\w+|[^\w\s]', open(sys.argv[1], encoding='utf-8').read())))" <file>
# piped content:
echo "$TEXT" | python3 -c "import re,sys; print(len(re.findall(r'\w+|[^\w\s]', sys.stdin.read())))"
```

---

## Results

| # | Question | Files opened | Tokens: grep | Tokens: files | **Total tokens** | Tool calls | Confident? |
|---|----------|-------------|-------------|--------------|-----------------|------------|------------|
| 1 | how do webhooks work | `dev/webhooks.md`, `en/user/webhooks.md` | 992 | 1 289 + 1 823 | **4 104** | 3 | yes |
| 2 | how do i publish a post to telegram | `en/user/telegram.md`, `en/user/publishing.md` | 1 400 | 1 632 + 1 039 | **4 071** | 3 | yes |
| 3 | set up a custom domain for my site | `dev/multidomain.md`, `en/user/multidomains.md` | 1 166 | 2 694 + 1 856 | **5 716** | 3 | yes |
| 4 | how to use multiple languages on my site | `dev/multilang.md`, `en/user/multilingual.md` | 652 | 3 178 + 1 271 | **5 101** | 3 | yes |
| 5 | two way sync between obsidian and the site | `dev/obsidian_sync.md`, `en/user/two-way-sync.md` | 980 | 6 144 + 688 | **7 812** | 3 | yes |
| 6 | what templates are available | `en/user/templates.md`, `en/user/default-template.md` | 2 594 | 2 751 + 4 449 | **9 794** | 3 | yes |
| 7 | accept paid subscriptions and monetization | `en/user/monetization.md` | 1 583 | 525 | **2 108** | 2 | yes |
| 8 | telegram post types and limits | `en/user/telegram.md`, `dev/telegram_bot_vs_userbot.md` | 585 | 1 632 + 1 309 | **3 526** | 3 | yes |

**Sorted totals:** 2 108 · 3 526 · 4 071 · 4 104 · 5 101 · 5 716 · 7 812 · 9 794

**Median tokens-to-answer: 4 603**

**Median tool calls: 3** (Q7 needed only 2; all others needed 1 grep + 2 file reads)

---

## Comparison vs MCP retrieval

| Method | Median tokens to answer | Median tool calls |
|--------|------------------------|-------------------|
| MCP focused-section retrieval | ~200 | 2 (search + note_html) |
| MCP whole-note retrieval | ~2 700 | 2 (search + note_html) |
| **Naive grep + file read (this benchmark)** | **~4 600** | **3** |

The grep approach costs roughly **23× more tokens** than the MCP focused-section read
(4 603 / 200), and about **1.7× more** than reading a whole note via MCP.

Tool call count is almost the same — 3 vs 2. The difference is not in round-trips but in
what each call returns: the MCP grep returns a ranked, focused result (one section from
one note); the file-system grep returns a list of 20–30 filenames with no content, forcing
full-file reads to proceed. The overhead is all token volume, not call count.

---

## Caveats

- **EN + RU duplication inflates grep output.** `docs/en/user/` and `docs/ru/user/` mirror
  the same files. Every topic grep returns both language trees, roughly doubling the list
  the agent must scan. We counted EN files only when the EN file was clearly primary, but
  the noise still shows up in the grep token cost.
- **Whole-file reads dominate.** Grep is cheap (500–2 500 tokens). The dominant cost is
  that a naive agent reads full files — it does not know which section holds the answer, so
  it consumes the entire file. `dev/obsidian_sync.md` alone is 6 144 tokens.
- **Not a worst case.** For Q6 (templates) we counted only 2 user-doc files. An agent that
  also opens `dev/default_template.md` (4 348 tokens) or `dev/layouts.md` (2 731 tokens)
  would spend 16 000+ tokens on that question alone.
- **Q7 is the cheap outlier.** `en/user/monetization.md` is 525 tokens and squarely
  answers the question; a well-named file makes grep effective for this case. It also needed
  only 2 tool calls (1 grep + 1 read) rather than the usual 3.
- **Tool call counts look deceptively close to MCP (3 vs 2).** The key difference is signal
  per call: grep returns filenames only, so all three calls are required just to get to
  readable content. MCP returns a focused answer in the second call.
- **These are realistic, not inflated numbers.** The 23× token ratio is what a real agent
  paying full attention to the grep results would spend.
