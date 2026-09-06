---
title: "Small MCP errors, and who pays for them"
free: true
lang: en
lang_redirect: "[[ru/thoughts/small-mcp-errors]]"
---

*One afternoon in September I asked a cheap AI assistant three questions about my own knowledge base, then read not its answers but everything it did on the way to them. I found six small defects in the service that the assistant talks to. Nobody had reported them, and I think I know why. A follow-up to [[en/thoughts/mcp-instructions-make-cheap-models-faster|the instructions A/B]].*

When an AI assistant answers a question from your company's documents, it does not read them the way you would. It sends short requests to a service that holds the documents: search for this, open that, show me the chapter list of this page. Each request comes back with a reply, and the assistant decides what to ask next based on that reply. A typical question takes ten such exchanges. You see the final answer. The exchanges happen out of sight.

That service can be wrong in ways nobody notices, because nobody looks. Not crashes. Small things. An error message that repeats your own typo back to you as the correction. A useful fact that the service knows but writes on a form the assistant never reads. A table of contents that, when you point at a chapter, says "go ask at the desk" instead of opening it. Each is a one-line fix. Each has a lopsided cost, and that lopsidedness is what this essay is about.

An expensive assistant absorbs the defect. It guesses around it, tries a different wording, spends two or three extra exchanges, and answers correctly. Your bill goes up by a few cents, and a few cents is not something anyone investigates. A cheap assistant cannot absorb it. It walks into the same wall three times, uses up its allowance of exchanges, and answers from general knowledge instead of from your documents. Same defect. Only the cheap assistant makes it visible.

## What I ran

trip2g has a public knowledge base that links several smaller ones together. A question about Marcus Aurelius and Confucius, for instance, has to reach two different collections: the Marcus Aurelius one sits directly under the main entrance, the Confucius one sits inside a separate hub of 21 philosophers, one level down. I took the search visualizer, a page on this site that shows an assistant's steps as an animated map, and turned it into a script that does the same thing without a browser, `scripts/mcp_search_logger.py`. It runs a cheap model, gives it about ten exchanges per question, and writes down every step: what the assistant thought, what it asked, what the service replied, what it finally answered. Three questions: what Marcus Aurelius and Confucius say about getting out of bed in the morning; where Epictetus and Wattles disagree about what is in our control; how to set up a persistent memory for Claude Code with trip2g.

## The error that repeats your own mistake

On 2 September the assistant answered none of the three from the documents. Here is the moment the first question died. The assistant had already found the card describing the Marcus Aurelius collection and was guessing the collection's address:

```
Assistant asked for:  the collection "trip2g/markavrelii"
Service answered:     no such collection; address it as "trip2g/markavrelii"
```

The assistant sent an address, and the service told it to use that same address. The message was a fill-in-the-blank template, and the blank had not been filled, so it echoed whatever came in. Imagine asking reception for room 214, being told "no, you want room 214", and going back to the lift. The assistant trusted the service, tried again, tried a third guess, and ran out of exchanges.

The correct address was in the service's reply all along. The service knew the shelf number. It just wrote it on the back of the form, in a structured section that most assistant programs, mine included, never show to the model. The model only sees the front, the plain text. A fact the assistant cannot see does not exist.

The other two questions ran into the same family of problems. On the Claude Code setup guide, the assistant asked to read three different sections and got the same paragraph back three times, because a leftover detail from the earlier search quietly took priority over the section it was now asking for. The final answer described the setup but had no actual command in it. And when the assistant asked the table of contents to open a chapter that had no subchapters, the service replied with a hint, "read it with the other tool", instead of just showing the chapter. One wasted exchange per chapter, out of ten. None of this produced an error. None of it failed a test.

## A more helpful error made things worse

I fixed all of it the same day, in [PR #339](https://github.com/trip2g/trip2g/pull/339). Search results now say in plain words which collection a card points to and how to get there. The section you name wins over leftovers. And the error message names the part of the address it does not recognise and lists the collections it does know. Within the hour the first question was answered from the source, Book 5 of the Meditations quoted with a link: rise for a human task, not to stay warm under the blanket.

Then the neighbouring collections received the same fix, and the first question broke again. The hub of philosophers now listed all 21 of its members in the error message, and the assistant stayed inside that list. It went to Epictetus as the nearest Stoic and never looked back at the main entrance, where Marcus Aurelius actually lives. The morning's vaguer message, "search the hub for it", had worked better by accident. A more detailed error had shrunk the assistant's world to one hub. [PR #341](https://github.com/trip2g/trip2g/pull/341) made the message name both levels, this hub's members and the main entrance's. On 6 September all three questions were answered from the documents on every run. Same cheap model, same allowance, about a cent per question before and after. The cost did not move. What moved was whether the allowance went into the answer or into the wall.

## Who pays

Here I have to be careful. I did not run an expensive assistant against the broken version. The claim that a stronger model would have absorbed these defects quietly comes from watching stronger models work in production, plus a reading of the records: at the echoing error, a stronger model would most likely have tried the short address on its own and paid two or three exchanges for it. That is a guess with a mechanism behind it, not a measurement. The measurement is cheap: replay the September records up to the first error with a stronger model and count how many exchanges it needs to recover. I have not done it.

The other half I did measure. On the cheap model these defects were the difference between answering from your documents and not. That is why they stayed invisible for so long. An expensive assistant turns a faulty service into a slightly higher bill, and a bill is not a bug report. A cheap one turns it into a wrong answer, and a wrong answer is a bug report, if someone reads the record.

## What to ask your team

Take the questions your assistant answers most often. Ask your team to run them once on a cheap model, with a limit on the number of exchanges, and to read what the assistant actually did, step by step. Not the answer. The steps. Every place where the service's reply cost the assistant one more exchange is a defect, whether it looked like an error or not. The script is `scripts/mcp_search_logger.py`; the browser version is [[search_visualizer|the visualizer]] on this site. Ten exchanges and a cent per run were enough to find six.
