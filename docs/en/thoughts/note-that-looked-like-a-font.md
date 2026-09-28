---
title: "The note the server took for a font"
free: true
lang_redirect: "[[ru/thoughts/note-that-looked-like-a-font]]"
---

*What this is: the trip2g server refused an immunology abstract because it decided the abstract was a font. Here is why that happened, and why a check that looked sensible never suited notes in the first place. Read it if you validate uploads by looking at their content.*

We were running a search benchmark on SciFact: 5,183 abstracts of scientific papers. We turned each one into a note and pushed them into trip2g with the ordinary sync. Everything went in except one batch of a hundred. The server answered:

```
Unsupported content type: application/vnd.ms-fontobject
```

`application/vnd.ms-fontobject` is Embedded OpenType, a font format from the Internet Explorer days. The batch held nothing but text about cells and proteins. The server log named the note:

```
---
title: "BASOPHILS AND THE T HELPER 2 ENVIRONMENT CAN PROMOTE..."
---
```

## How text became a font

Before saving a note, the server checked its content with `http.DetectContentType` from Go's standard library. The function looks at the first 512 bytes and compares them with a list of known signatures. PNG starts with `\x89PNG`, GIF with `GIF89a`, MP3 with `ID3`. If no signature matches and there are no binary bytes, the answer is "text". The server let through only text and HTML.

The EOT font signature is unusual. The first 34 bytes can be anything, and bytes 35 and 36 must be a capital `LP`. Count it out: `---`, a newline and `title: "` make 12 bytes. So bytes 35 and 36 of the file are the 23rd and 24th letters of the title. In "BASOPHILS AND THE T HELPER 2" those are the `L` and `P` of HELPER. To the function, that is a font.

The function is not wrong. It does exactly what it was written for: it implements the algorithm a browser uses to guess a response's type when the server sent no `Content-Type` header. The browser has to decide whether to show the response as an image, play it as audio or print it as text. A best guess beats no guess.

Our question is different. We are not guessing what arrived: we know it is a note, and we want to confirm that it really is text. And arbitrary text will sooner or later match somebody's signature:

| Start of the note | What the function decided |
|---|---|
| `---` + `title: "BASOPHILS AND THE T HELPER 2"` | EOT font |
| `ID3 tags explained: how MP3 metadata works` | MP3 audio |
| `GIF89a is an image format from 1989` | GIF image |
| `<?xml version="1.0"?> notes about XML` | XML document |
| `# A note` | text |

A note about MP3 tags that opens with the word ID3 is an entirely ordinary case. And the EOT collision is less rare than it looks: an acronym like NLP or the word HELP, in capitals, at the right spot in a title is enough.

## The message was the worst part

A user who hit this saw "Unsupported content type: application/vnd.ms-fontobject". Not a word about the title, byte 35 or what to fix. The note simply does not sync while its neighbours do. We ran into it by accident, across five thousand texts someone else wrote. A real user with one such note would most likely conclude that sync was broken.

## What changed

Every format the sync accepts is text: Markdown, HTML, the JSON of canvases and bases. So there is no need to guess from signatures. Checking the encoding is enough: the content must be valid UTF-8 with no NUL bytes.

```go
if !utf8.ValidString(update.Content) || strings.ContainsRune(update.Content, 0) {
    return &model.ErrorPayload{Message: "File content must be UTF-8 text"}
}
```

An image, a binary or a file in another encoding posing as `.md` is still rejected: a PNG contains both NUL bytes and sequences that are invalid UTF-8. Text with the word HELPER gets through, because text is what it was all along.

Checking by content is right when you do not know what arrived. When you do know, check that it is what it claims to be.
