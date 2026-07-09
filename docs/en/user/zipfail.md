---
title: "The ZIP that quietly dropped its .obsidian folder"
free: true
lang_redirect: "[[ru/user/zipfail]]"
---

The [[en/user/onboarding-vault|onboarding vault]] you download from trip2g is a ZIP. For months it extracted cleanly. Then a rushed change swapped how it was built and quietly broke extraction of its `.obsidian/` folder — the terminal `unzip` and macOS's own unarchiver both mangled it. The vault still came out full of notes, just without the folder that holds the sync plugin, so it silently would not sync. It sat in the main branch for five days before I caught it. This is what went wrong, why it took five days, and how it got fixed.

The story has a few turns. The failure was silent — you got a vault that looked fine and simply would not sync. The change that caused it was itself a fix for a different bug. And it turns out to be one line in a six-week, $22k sprint — which raises a harder question about what happens to bug counts when AI writes ten, then a hundred times more code.

### The wall of errors

The onboarding vault ships as a ZIP. One day I extracted one from a terminal, and it shouted:

```
checkdir error:  .../.obsidian exists but is not directory
                 unable to process .obsidian/app.json
```

Every file inside `.obsidian/` failed the same way. The rest of the vault extracted, but the `.obsidian/` folder with the plugins never got created — and without the sync plugin, syncing did not work.

### The root cause: a missing slash

A ZIP stores a folder as its own entry with a trailing slash: `.obsidian/`. The packer — a Python script in `onboarding-vault/generate.sh` — wrote directory entries **without** the slash, as plain `.obsidian`.

Without the slash, that entry is a zero-byte record. Info-ZIP's `unzip` reads it as a regular **file** named `.obsidian`. Then it hits `.obsidian/app.json`, tries to create `app.json` *inside* a thing it already decided was a file, and gives up: `exists but is not directory`.

### How it got here: a fix that shipped a regression

The vault used to be packed by the system `zip` command. That tool writes proper directory entries with trailing slashes, so `unzip` was happy for months — but it doesn't set the ZIP UTF-8 filename flag, so Windows decoded Cyrillic note names with its legacy codepage and mangled them.

So I swapped `zip` for a Python `zipfile` packer to fix exactly that. Python encodes names as UTF-8 and sets the flag automatically, and Windows was happy again. The same change also wrote directory entries without their trailing slash. One fix and one fresh regression, shipped in the same commit.

### Why it took five days

The broken archive was in the main branch for five days. Nothing between commit and download re-extracts it and checks the result — not the build that repacks it, not CI, not me. And the failure is quiet: you still get a vault full of notes, it just has no `.obsidian`, so it silently will not sync.

So it was not that clever tools forgave it — every extractor I tried, terminal and macOS both, got it wrong. It was that few people pull a fresh build in five days, the ones who might have got a vault that looked fine, and nothing on the path unpacks one to check. I caught it when I unpacked one myself on day five.

### The fix

Emit directory entries with a trailing slash. One small change to the packer:

```python
# before
info = zipfile.ZipInfo(str(rel))
# after
info = zipfile.ZipInfo(str(rel) + "/")
```

The Go download path copies archive entries through unchanged, so fixing the source ZIP fixes every vault anyone downloads afterward. Fix: [`c7b305a8`](https://github.com/trip2g/trip2g/commit/c7b305a8a6278834edce617efe5859f0a7cde7e8).

### The guard tests

Two tests watch for this now. A pure-Go one asserts the invariant directly on the generated archive: no plain-file entry may have another entry nested under it — if both `X` and `X/...` exist, then `X` must end in `/`. It is fast and needs nothing external. The second is blunter: it shells out to `unzip`, extracts the archive, and fails on a single warning — the original bug, reproduced end to end. Both fail on the old broken ZIP and pass on the fixed one. Commits: [`85a1d913`](https://github.com/trip2g/trip2g/commit/85a1d913ff1be4b0e516fafa017ee44b8a745b2d), [`0e66cf1b`](https://github.com/trip2g/trip2g/commit/0e66cf1be32b9440c748ea0d2bc62ab39135426e).

### The lesson

Those five days were not a bad model or a broken process. They look more like statistical noise: at this volume of code, some defects slip through no matter what, and this one lived for five days. The guard test above does not erase that background rate — it only lowers it, unpacking and checking the archive on every build instead of waiting for someone to unpack one by hand. That is the whole lesson, and it is smaller than "use better tools": check the artifact you ship, automatically, every time.

### The receipt

This one landed during the great Fable sprint before 7 July — moving fast, cutting corners, fixing one thing and nicking another. The sprint left a paper trail. `bunx ccusage claude weekly` prints what Claude Code burned, week by week:

![[./img/ccusage-weekly.png]]

Read top to bottom, that is the marathon. The weekly bill climbs almost tenfold each week, tops out near $9k for the week of 28 June, then falls by two-thirds the week after — when I stopped sprinting. Six weeks, 26.5 billion tokens, about $22k. Ninety-six percent of it is cache reads: the model re-reading large contexts, not writing new code. This ZIP bug is one line buried in that total.

### The moral

AI plows the whole field. In a week it turns over more ground than I would in a season — which is exactly why there is so much dirt to rake back afterward. The missing slash was one clod. I am out of crunch mode now, and I will be clearing the rest for a while.

### The bigger question, a guess

This bug is not special, and that is the point. At equal volume, AI seems to make about as many mistakes as I do — no better, no worse. What changed is the volume. There is roughly ten times more code than a year ago, and there will be a hundred times more. Hold the error rate steady and multiply the code by a hundred, and you get a hundred times the bugs.

The models learned from human code, written by people who never worked at this scale — where the raw count of defects stayed below anything critical. Apply the same rate to far more output and it crosses that line. So the bar for code quality rises to meet it, and it already is. Companies are starting to turn you down if you do not write with AI: they want the productivity, and they expect the errors kept to something acceptable.

If that holds — if AI writing the same volume produces roughly the same errors a person would — then the speed is real and the discipline has to scale with it: stricter tools, more guard tests, more raking through the field afterward. That is a guess. We will know next year.

You can vibe-code your own trip2g, in your own style — and you will collect just as many bugs, in your own style. It is fun. Not always. The test is in, plenty learned, moving on.
