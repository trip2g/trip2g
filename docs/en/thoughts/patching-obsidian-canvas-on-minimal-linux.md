---
title: "Patching Obsidian Canvas on minimal Linux"
order: 15
layout: trip2g/content-first
free: true
lang_redirect: "[[ru/thoughts/patching-obsidian-canvas-on-minimal-linux]]"
---

Ubuntu 24.04 LTS aarch64, VMware on a Mac, no desktop environment — dwm under bare X11, `startx` from TTY. Obsidian 1.12.7 (tested both Flatpak and AppImage).

Canvas opens. Nodes render. You can click them. But you can't drag existing nodes, and you can't draw edges between them. New nodes created via the toolbar drag fine. Console: silent.

**The fix: a two-file Obsidian plugin that rewrites all non-mouse pointer events to `pointerType: "mouse"` before Obsidian's handlers see them.** Install it, restart, done. The root cause and the dead ends are below if you want to understand why.

---

## What we tried that didn't work

- Reinstall AppImage, switch to Flatpak — same behavior.
- Force X11 backend: `GDK_BACKEND=x11 ELECTRON_OZONE_PLATFORM_HINT=x11` — no effect.
- Community plugin "Canvas Drag Fix" — didn't help. We'll explain why in a moment.
- udev rule rewriting `ID_INPUT_TABLET=0` `ID_INPUT_MOUSE=1` — didn't help either. Chromium reads XInput2 directly; udev properties don't reach it.

## Finding the root cause

Paste this into Obsidian DevTools (Ctrl+Shift+I → Console), then drag a node:

```js
['pointerdown','pointermove','pointerup'].forEach(name =>
  window.addEventListener(name, e =>
    console.log(name, 'type:', e.pointerType, 'id:', e.pointerId), true));
```

Output:

```
pointerdown  type: mouse  id: 1
pointermove  type: pen    id: 4
pointermove  type: pen    id: 4
pointerup    type: mouse  id: 1
```

Same physical mouse. `pointerdown` and `pointerup` are tagged `"mouse"`. Every `pointermove` in between is tagged `"pen"` with a different pointer ID.

Obsidian's Canvas drag handler starts a drag on `pointerdown`, captures the pointer, then expects subsequent `pointermove` events to match the same `pointerType` and `pointerId`. They don't. Drag silently rejected every time.

The cause is Chromium's XInput2 device classification. VMware's virtual mouse exposes stylus axes (absolute coordinates, pressure), so Chromium labels its motion events as pen input. The `pointerdown`/`pointerup` path apparently goes through a different code path and stays tagged as mouse.

**Why "Canvas Drag Fix" doesn't help:** that plugin intercepts `pointerdown` events where `pointerType === 'pen'` and re-dispatches them as mouse events. Our `pointerdown` is already `"mouse"` — the problem is `pointermove`, which the plugin doesn't touch.

## The fix

A small plugin that attaches a capture-phase listener at the `window` level and rewrites any non-`"mouse"` `pointerType` to `"mouse"` (and normalizes `pointerId` to `1`) before any other handler runs. It covers all ten pointer event types.

### Installation

1. Create the directory `<vault>/.obsidian/plugins/pointer-fix/`
2. Save the two files below into it
3. Open `<vault>/.obsidian/community-plugins.json` and add `"pointer-fix"` to the array (if the file doesn't exist yet, create it: `["pointer-fix"]`)
4. If you haven't enabled community plugins: Settings → Community plugins → "Turn on community plugins"
5. Restart Obsidian
6. Open DevTools → Console: you should see `[pointer-fix] installed`

Canvas drag now works. Edges form.

### `main.js`

```js
'use strict';

const obsidian = require('obsidian');

const EVENTS = [
	'pointerdown', 'pointermove', 'pointerup', 'pointercancel',
	'pointerover', 'pointerout', 'pointerenter', 'pointerleave',
	'gotpointercapture', 'lostpointercapture'
];

function fixPointer(ev) {
	if (ev.pointerType !== 'mouse') {
		try { Object.defineProperty(ev, 'pointerType', { value: 'mouse', configurable: true }); } catch (e) {}
		try { Object.defineProperty(ev, 'pointerId', { value: 1, configurable: true }); } catch (e) {}
	}
}

class PointerFixPlugin extends obsidian.Plugin {
	async onload() {
		for (const name of EVENTS) {
			window.addEventListener(name, fixPointer, { capture: true });
		}
		console.log('[pointer-fix] installed');
	}

	onunload() {
		for (const name of EVENTS) {
			window.removeEventListener(name, fixPointer, { capture: true });
		}
	}
}

module.exports = PointerFixPlugin;
```

### `manifest.json`

```json
{
	"id": "pointer-fix",
	"name": "Pointer Type Fix",
	"version": "0.1.0",
	"minAppVersion": "1.0.0",
	"description": "Rewrites non-mouse pointerType to 'mouse'. Workaround for Chromium misclassifying VMware/virtual mice as pen, which breaks Canvas drag.",
	"author": "alexes",
	"isDesktopOnly": true
}
```

## AppImage and `--no-sandbox`

On some minimal Linux setups — kernels or distros that restrict unprivileged user namespaces — Obsidian's AppImage won't start. The process exits immediately with a sandbox error. Workaround:

```bash
~/Obsidian.AppImage --no-sandbox
```

This disables Chromium's renderer-process sandbox. On a personal dev VM that's an acceptable trade. On a shared or internet-facing machine it is not.

## One caveat

If you have a real stylus or Wacom tablet on this machine, this plugin masks pen-specific data — pressure, tilt, the pen pointer ID. Disable it before drawing.

---

When a Chromium-based app misbehaves on Linux, log every pointer event at the capture phase — the truth is in the device classification, not in the layers above it.
