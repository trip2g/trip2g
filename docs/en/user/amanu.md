---
title: "Call transcripts from Amanu"
free: true
lang_redirect: "[[ru/user/amanu]]"
---

[Amanu](https://amanu.me/) records and transcribes calls on your own computer. A small script sends each finished transcript to your trip2g knowledge base, with nothing to press after a call.

Step-by-step setup with the files to download: [trip2g.github.io/amanu](https://trip2g.github.io/amanu/) (in Russian). Source: [github.com/trip2g/amanu](https://github.com/trip2g/amanu).

### What you get

After each call a note appears in the `calls` folder, named after the date and time of the meeting. It holds the transcript and, when Amanu made one, the summary. The note is searchable at once, including through the [[mcp|MCP server]].

No audio is uploaded. Only text reaches the knowledge base.

### What you need

| What | Where to get it |
|---|---|
| Windows 11 24H2 or later | required by Amanu itself |
| The endpoint | `https://your-site/_system/graphql` |
| An API key | **Administration → API keys** |

An API key carries administrator rights over all content. Keep the settings file like a password and revoke a lost key in the same section.

### Setup

1. Install Amanu from [amanu.me](https://amanu.me/). In the first-run window choose local transcription.
2. Turn Summary off unless you connect a model for it. While Amanu waits for a summary nobody can produce, the script is never run.
3. Put `amanu-to-trip2g.ps1` and a settings file `amanu-to-trip2g.json` into one folder:

   ```json
   {
     "endpoint": "https://your-site/_system/graphql",
     "apiKey": "your key",
     "folder": "calls"
   }
   ```

4. In Amanu open Settings → Advanced → "Run after each meeting". Program: `powershell.exe`. Arguments, one per line:

   ```
   -NoProfile
   -ExecutionPolicy
   Bypass
   -WindowStyle
   Hidden
   -File
   C:\Users\YOUR_NAME\amanu-trip2g\amanu-to-trip2g.ps1
   {session}
   ```

5. Close the settings window. Amanu saves this field only when the window is closed.

### Checking it

Make a call longer than a minute. Once Amanu finishes transcribing, the note appears in `calls`.

The script keeps a log beside itself, `amanu-to-trip2g.log`. An `OK` line means the note was sent. A `FAIL` line says what stopped it.

### How it works

Amanu can run a program after each meeting and hands it the meeting folder. The script reads `transcript.md` and `summary.md` from it, builds one note and sends it with the `pushNotes` mutation, authorised by the `X-API-Key` header. It is the same call the [[cli|CLI]] and the Obsidian plugin make. More on the API: [[graphql]].

### Limits

- Windows only. There is no macOS script yet.
- No retry. A meeting that ends with no internet connection is not sent, and the log keeps a `FAIL` line.
- With local transcription the speakers are labelled only as "me" and "them".
