import * as esbuild from 'esbuild'
import { mkdirSync } from 'fs'

mkdirSync('dist', { recursive: true })

const watch = process.argv.includes('--watch')

/** @type {import('esbuild').BuildOptions} */
const opts = {
  entryPoints: ['src/index.tsx'],
  bundle: true,
  format: 'iife',
  minify: true,
  define: { 'process.env.NODE_ENV': '"production"' },
  loader: { '.css': 'text' },
  target: 'es2019',
  jsx: 'automatic',
  outfile: 'dist/kanban.js',
}

if (watch) {
  const ctx = await esbuild.context(opts)
  await ctx.watch()
  console.log('watching...')
} else {
  const result = await esbuild.build({ ...opts, metafile: true })
  const size = Object.values(result.metafile.outputs)[0].bytes
  console.log(`dist/kanban.js  ${(size / 1024).toFixed(1)} kB`)
}
