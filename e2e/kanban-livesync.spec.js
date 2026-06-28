// @ts-check
/**
 * OUTLINE / TODO (not yet implemented) — kanban two-browser live-sync, no data loss.
 *
 * This is the test the unit mocks could not cover: the REAL transport (noteChanges
 * GraphQL-SSE + updateNotes) driving the kanban board in two browser contexts at once.
 * The unit reproductions live in templates/kanban/src/Board.dataloss.integration.test.tsx
 * (mocked ./api); they are the MUST and they pass. This e2e proves the same losslessness
 * end-to-end and is the stretch goal.
 *
 * It is `describe.skip` because the harness does not yet:
 *   1. Seed a kanban board note (e.g. boards/demo.md with the kanban-plugin frontmatter),
 *      and serve a page that mounts templates/kanban/dist/kanban.js against it (the same
 *      bundle `memcli --kanban` installs). The board reads window.__trip2g_kanban = {path,
 *      content, editable}. A small served HTML harness that sets that global + loads the
 *      bundle is enough; admin cookie auth makes it `editable`.
 *   2. Provide an admin browser session (cookie), not just the X-Api-Key request header —
 *      the board's subscribeNoteChanges/fetchNoteContent use the admin cookie
 *      (credentials:'include'), see templates/kanban/src/api.ts.
 *
 * WHAT TO ASSERT (the two failures the user actually hit):
 *
 *   Propagation (the live-sync that silently never happened):
 *     - Open the SAME board in context A and context B as the same admin.
 *     - In A: add a column "Remote" + a card. (drag/click the board UI, or POST updateNotes
 *       as A and let A's own board reflect it.)
 *     - Assert B's DOM shows the new column AND card within a few seconds (real SSE
 *       propagation — this is what root-cause (b) suppression broke: a genuine remote
 *       event at the latest version was dropped as a self-echo).
 *
 *   No-loss on concurrent same-column add (root-cause (a)):
 *     - With both boards open, B adds a card to a column while A's add to the SAME column
 *       propagates mid-edit.
 *     - Assert B shows NO `boardConflictReloading` toast (i18n key 'boardConflictReloading')
 *       and B's card persists — i.e. the column-keyed card-level 3-way merge kept both
 *       cards instead of conflict→reload→drop.
 *
 * Suggested skeleton (mirrors e2e/renderlayout-live.spec.js for the two-context + auth
 * pattern, and e2e/updatenotes.spec.js for the GraphQL/admin-key plumbing):
 *
 *   const ctxA = await browser.newContext({ ...adminAuth });
 *   const ctxB = await browser.newContext({ ...adminAuth });
 *   const a = await ctxA.newPage(); const b = await ctxB.newPage();
 *   await a.goto(BOARD_URL); await b.goto(BOARD_URL);
 *   // A adds a column + card …
 *   await expect(b.getByText('Remote')).toBeVisible({ timeout: 8000 });
 *   await expect(b.getByText('A-card')).toBeVisible({ timeout: 8000 });
 *   // B adds a card to the same column A is touching …
 *   await expect(b.getByText(/board.*conflict|reload/i)).toHaveCount(0);
 *   await expect(b.getByText('B-card')).toBeVisible();
 */
import { test } from '@playwright/test';

test.describe.skip('kanban two-browser live-sync (no data loss) — TODO: needs a served board harness', () => {
  test('A adds a column + card → B sees them; B adds a card with no conflict-reload', () => {
    // See the outline above. Implement once the harness seeds + serves a kanban board page.
  });
});
