// Load IIFE bundle (sets global $trip2g_sync_bundle)
require('./browser-sync.js')

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
	export const $trip2g_sync = $trip2g_sync_bundle
}
