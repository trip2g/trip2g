# Screenshots needed for user docs

_Created 2026-06-21. The pages below contain `> 🖼️ Screenshot — …` placeholder blocks instead of real images (we don't have the admin captures yet)._

## How to fill one

1. Capture the admin screen described below.
2. Save the PNG under `docs/assets/admin/` with the given filename.
3. Replace the placeholder block in the page with `![alt text](/assets/admin/<name>.png)`.
4. Bilingual pages share the same image file — one capture serves both EN and RU.

> Open question: confirm the served asset path. Existing docs reference images as `/assets/...`; `docs/assets/` currently holds `screenshot.webp`. If `docs/assets/admin/` doesn't resolve on the rendered site, sync the image into the page's own folder instead.

If/when the e2e screenshot harness is built (deferred — see `docs_diagrams_plan.md`), these same filenames can be produced automatically.

## List

### `assets/admin/editor-open.png`
- **In:** [[en/user/editor]], [[ru/user/editor]]
- **Capture:** Editor modal open. Left: folder-tree file browser with one file highlighted. Center: note markdown in the text area. Top-right: **Save** and **Versions** buttons. Admin header visible.

### `assets/admin/editor-versions.png`
- **In:** [[en/user/editor]], [[ru/user/editor]]
- **Capture:** Versions panel open on the right of the editor — list of past saves with dates/times, one version selected with its content shown, **Restore** button visible.

### `assets/admin/editor-icon-location.png`
- **In:** [[en/user/editor]], [[ru/user/editor]]
- **Capture:** Top of a published page with the admin bar — the editor (pencil) icon highlighted next to the search box, plus an inset showing the same icon in the admin panel header (top-right).

### `assets/admin/live-editing-side-by-side.png`
- **In:** [[en/user/live-editing]], [[ru/user/live-editing]]
- **Capture:** Obsidian and a browser side by side. Left: a note being edited in Obsidian (a visible change, e.g. a heading retyped). Right: the published page already showing the update — no reload spinner/flicker, to convey the surgical update.

### `assets/admin/hub-index.png`
- **In:** [[en/user/hub]], [[ru/user/hub]]
- **Capture:** The public hub page at `/hub` — the list of federated knowledge bases with titles and descriptions; highlight the entries and the link to each base.

### `assets/admin/canvas-rendered.png`
- **In:** [[en/user/canvas]], [[ru/user/canvas]]
- **Capture:** A published `.canvas` file in a browser — the board with at least two cards and one connection arrow; highlight that cards are clickable links.

### `assets/admin/oauth-settings-form.png`
- **In:** [[en/user/oauth]] (mirror into [[ru/user/oauth]])
- **Capture:** Admin panel → Google OAuth (or GitHub OAuth) settings form — Client ID field, Client Secret field, Save/Activate button. Show one provider with a note that the other looks identical.

### `assets/admin/telegram-import.png`
- **In:** [[en/user/telegram-import]] (mirror into [[ru/user/Импорт из Telegram]])
- **Capture:** Admin panel → Telegram import screen — the account selector, the channel ID field, the target-folder field, and the **Start import** button, all four visible.
