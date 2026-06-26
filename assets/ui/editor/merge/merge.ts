namespace $ {
	// Line-level 2-way merge: lines common to both stay once; only diverging runs are
	// wrapped in git-style conflict markers. Alignment via a longest-common-subsequence
	// of lines, so unchanged regions don't get marked.
	export function $trip2g_editor_merge(mine: string, theirs: string): string {
		const a = mine.split('\n')
		const b = theirs.split('\n')
		const n = a.length, m = b.length
		// LCS table is O(n*m); fall back to whole-block markers for pathological sizes.
		if (n * m > 4_000_000) {
			return `<<<<<<< mine\n${mine}\n=======\n${theirs}\n>>>>>>> updated elsewhere\n`
		}
		const lcs: number[][] = Array.from({ length: n + 1 }, () => new Array(m + 1).fill(0))
		for (let i = n - 1; i >= 0; i--) {
			for (let j = m - 1; j >= 0; j--) {
				lcs[i][j] = a[i] === b[j] ? lcs[i + 1][j + 1] + 1 : Math.max(lcs[i + 1][j], lcs[i][j + 1])
			}
		}
		const out: string[] = []
		let mineBuf: string[] = []
		let theirsBuf: string[] = []
		const flush = () => {
			if (mineBuf.length || theirsBuf.length) {
				out.push('<<<<<<< mine', ...mineBuf, '=======', ...theirsBuf, '>>>>>>> updated elsewhere')
				mineBuf = []
				theirsBuf = []
			}
		}
		let i = 0, j = 0
		while (i < n && j < m) {
			if (a[i] === b[j]) {
				flush()
				out.push(a[i])
				i++; j++
			} else if (lcs[i + 1][j] >= lcs[i][j + 1]) {
				mineBuf.push(a[i]); i++
			} else {
				theirsBuf.push(b[j]); j++
			}
		}
		while (i < n) { mineBuf.push(a[i]); i++ }
		while (j < m) { theirsBuf.push(b[j]); j++ }
		flush()
		return out.join('\n')
	}
}
