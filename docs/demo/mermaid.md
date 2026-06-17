---
free: true
title: Mermaid — diagrams from code blocks
---

# Mermaid diagrams

Obsidian renders ` ```mermaid ` fenced blocks as diagrams out of the box, and so
does trip2g. Write the diagram in a `mermaid` code block — trip2g loads the
mermaid renderer only on pages that actually contain one, so diagram-free pages
pay nothing.

## Flowchart

```mermaid
graph TD;
  A[Start] --> B{Is it working?};
  B -- Yes --> C[Ship it];
  B -- No --> D[Debug];
  D --> B;
```

## Sequence diagram

```mermaid
sequenceDiagram
  participant U as User
  participant T as trip2g
  participant TG as Telegram
  U->>T: Publish note
  T->>TG: Send post
  TG-->>T: message_id
  T-->>U: Published link
```

## Pie chart

```mermaid
pie title Traffic sources
  "Search" : 45
  "Telegram" : 30
  "Direct" : 25
```
