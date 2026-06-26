// Load the IIFE bundle (sets global $trip2g_sync_bundle). Browser-only: the $mol
// node test build (e.g. admin/-/node.test.js) executes this require but cannot
// resolve the browser-only bundle. Guard it on `window` — mam still bundles
// browser-sync.js for the web build, but node skips the require at runtime.
if (typeof window !== 'undefined') {
	require('./browser-sync.js')
}

// Declare the global created by IIFE bundle (types are ambient: $trip2g_sync_types)
declare const $trip2g_sync_bundle: $trip2g_sync_types.Bundle

namespace $ {
	/**
	 * Browser sync module for synchronizing local markdown files with trip2g server.
	 * Uses File System Access API via browser-fs-access.
	 *
	 * Usage:
	 *   const env = new $.$trip2g_sync.BrowserEnv({ apiUrl, apiKey }, callbacks)
	 *   await env.init()
	 *   await env.selectDirectory()
	 *   const result = await env.sync()
	 *
	 * See docs/browser_sync.md for full documentation.
	 */
	// `typeof` guards against a ReferenceError under the node test build, where the
	// browser bundle (and its global) is never loaded; in the browser it is set.
	export const $trip2g_sync = (typeof $trip2g_sync_bundle === 'undefined'
		? undefined
		: $trip2g_sync_bundle) as $trip2g_sync_types.Bundle
}
