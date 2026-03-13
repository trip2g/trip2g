---
free: true
---
Test: [[dup]] - goes to ROOT, not local! ⚠️
Local: [[./dup]] - this one stays local ✅
Explicit: [[folder/dup]] - also local ✅

Should resolve to ROOT:

![[_banner]]

Should resolve to local folder:

![[./_banner]]
