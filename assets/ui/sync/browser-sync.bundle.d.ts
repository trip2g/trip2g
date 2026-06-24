// Ambient type definitions for the browser-sync IIFE bundle (browser-sync.js).
//
// IMPORTANT: this file is globbed and transpiled by mam ($node.typescript.transpileModule)
// and concatenated into the page bundle. It MUST stay fully ambient — no top-level
// `export`/`import` — otherwise mam emits a stray `export ...` into the script and the
// whole bundle fails to parse ("Unexpected token 'export'"). Keep everything inside the
// `declare namespace`, which produces zero runtime output.

declare namespace $trip2g_sync_types {
	export interface StorageConfig {
		dbName: string
	}

	export function configureStorage(config: Partial<StorageConfig>): void
	export function saveDirectoryHandle(handle: FileSystemDirectoryHandle): Promise<void>
	export function loadDirectoryHandle(): Promise<FileSystemDirectoryHandle | null>
	export function clearDirectoryHandle(): Promise<void>
	export function requestPermission(handle: FileSystemDirectoryHandle): Promise<boolean>
	export function checkPermission(handle: FileSystemDirectoryHandle): Promise<boolean>

	export interface Progress {
		step: 'classify' | 'pull' | 'push' | 'upload_asset' | 'download_asset' | 'conflict' | 'commit'
		current: number
		total: number
		path?: string
	}

	export interface ConflictInfo {
		path: string
		localContent: string
		remoteContent: string
		localHash: string
		remoteHash: string
	}

	export type ConflictResolution = 'keep_local' | 'keep_remote' | 'keep_both' | 'skip'

	export interface AssetConflictInfo {
		path: string
		absolutePath: string
		noteId: string
		localHash: string
		remoteHash: string
		remoteUrl: string
	}

	export type AssetConflictResolution = 'keep_local' | 'keep_remote' | 'skip'

	export interface UICallbacks {
		onProgress?: (progress: Progress) => void
		onConflict?: (conflicts: ConflictInfo[]) => Promise<ConflictResolution[]>
		onAssetConflict?: (conflicts: AssetConflictInfo[]) => Promise<AssetConflictResolution[]>
		onServerDeleted?: (paths: string[]) => Promise<boolean>
		confirmPush?: (paths: string[]) => Promise<boolean>
		onLog?: (message: string, level: 'info' | 'warn' | 'error') => void
	}

	export interface BrowserEnvOptions {
		apiUrl: string
		apiKey?: string
		useCookieAuth?: boolean
		twoWaySync?: boolean
		publishField?: string
	}

	export interface FileClassification {
		path: string
		action: string
		localHash: string | null
		remoteHash: string | null
		lastSyncedHash: string | null
	}

	export interface SyncPlan {
		classifications: FileClassification[]
		pushes: FileClassification[]
		pulls: FileClassification[]
		conflicts: FileClassification[]
		localOnly: FileClassification[]
		remoteOnly: FileClassification[]
		localDeleted: FileClassification[]
		serverDeleted: FileClassification[]
		unchanged: number
	}

	export interface SyncResult {
		pulled: number
		pushed: number
		conflictsResolved: number
		assetsUploaded: number
		assetsDownloaded: number
		errors: string[]
	}

	export class BrowserEnv {
		constructor(options: BrowserEnvOptions, callbacks?: UICallbacks)

		init(): Promise<void>
		hasStoredDirectory(): Promise<boolean>
		requestStoredPermission(): Promise<boolean>
		selectDirectory(): Promise<boolean>
		clearDirectory(): Promise<void>
		getDirectoryName(): string | null
		isReady(): boolean

		getLocalFiles(): Promise<{ path: string }[]>
		getSyncPlan(): Promise<SyncPlan>
		sync(): Promise<SyncResult>
	}

	// Shape of the global `$trip2g_sync_bundle` created by the IIFE bundle.
	export interface Bundle {
		BrowserEnv: typeof BrowserEnv
		configureStorage: typeof configureStorage
		saveDirectoryHandle: typeof saveDirectoryHandle
		loadDirectoryHandle: typeof loadDirectoryHandle
		clearDirectoryHandle: typeof clearDirectoryHandle
		requestPermission: typeof requestPermission
		checkPermission: typeof checkPermission
	}
}
