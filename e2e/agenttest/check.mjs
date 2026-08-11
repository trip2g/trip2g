// Assertions over a `claude -p --output-format stream-json` transcript.
//
// The point of the agent test is to assert on what the agent DID (tool calls,
// live site state), not on how it phrased things. Only the two scenarios that
// are genuinely about understanding fall back to an LLM judge.
//
// Usage:
//   node check.mjs tools <transcript.jsonl>
//   node check.mjs final <transcript.jsonl>
//   node check.mjs assert-tool <transcript.jsonl> <regex>      # tool name or its input
//   node check.mjs assert-no-tool <transcript.jsonl> <regex>
//   node check.mjs assert-final <transcript.jsonl> <regex>
//   node check.mjs judge <transcript.jsonl> <criterion> [model]
import fs from 'node:fs';
import { spawnSync } from 'node:child_process';

function readTranscript(file) {
  const toolUses = [];
  let finalText = '';
  let runError = '';
  const assistantTexts = [];

  for (const line of fs.readFileSync(file, 'utf8').split('\n')) {
    if (!line.trim()) continue;

    let event;
    try {
      event = JSON.parse(line);
    } catch {
      continue; // stream-json can be interleaved with stray output
    }

    if (event.type === 'result') {
      if (typeof event.result === 'string') finalText = event.result;
      // A run that never reached the model (auth, credits, rate limit) must not
      // be scored as a comprehension failure.
      if (event.is_error === true || (event.subtype && event.subtype !== 'success')) {
        runError = typeof event.result === 'string' ? event.result : event.subtype;
      }
    }

    for (const block of event.message?.content ?? []) {
      if (block.type === 'tool_use') {
        toolUses.push({ name: block.name, input: JSON.stringify(block.input ?? {}) });
      }
      if (block.type === 'text' && event.type === 'assistant') {
        assistantTexts.push(block.text);
      }
    }
  }

  // A run killed by --max-turns emits no result event; the last assistant
  // message is still the best available answer.
  if (!finalText) finalText = assistantTexts.at(-1) ?? '';

  return { toolUses, finalText, runError };
}

function matchesTool({ toolUses }, pattern) {
  const re = new RegExp(pattern, 'i');
  return toolUses.some((t) => re.test(t.name) || re.test(t.input));
}

function judge(transcript, criterion, model) {
  const prompt = [
    'You are grading one answer produced by another AI agent. Be strict but fair.',
    '',
    `CRITERION: ${criterion}`,
    '',
    'ANSWER:',
    '"""',
    transcript.finalText,
    '"""',
    '',
    'Reply with exactly PASS or FAIL on the first line, then one short sentence of justification.',
  ].join('\n');

  const res = spawnSync('claude', ['-p', prompt, '--model', model], { encoding: 'utf8' });
  const out = (res.stdout || '').trim();

  // Distinguish "the judge could not run" from "the judge said FAIL" — only the
  // latter is a verdict about the agent.
  if (res.status !== 0 || !/^\s*(PASS|FAIL)\b/i.test(out)) {
    return { broken: true, reason: `judge unusable: ${out || (res.stderr || '').trim()}` };
  }

  return { pass: /^\s*PASS\b/i.test(out), reason: out.split('\n').slice(0, 2).join(' ') };
}

const [command, file, arg, arg2] = process.argv.slice(2);

if (!command || !file) {
  console.error('usage: node check.mjs <command> <transcript.jsonl> [args]');
  process.exit(2);
}

const transcript = readTranscript(file);

// Exit code 3 means "this run cannot be scored", which the runner reports
// separately from a genuine assertion failure.
const EXIT_UNSCORABLE = 3;

if (transcript.runError && command !== 'final') {
  console.log(`RUN FAILED: ${transcript.runError.split('\n')[0]}`);
  process.exit(EXIT_UNSCORABLE);
}

switch (command) {
  case 'status': {
    console.log('run completed');
    break;
  }

  case 'tools': {
    const names = [...new Set(transcript.toolUses.map((t) => t.name))];
    console.log(names.join(', ') || '(none)');
    break;
  }

  case 'final': {
    console.log(transcript.finalText);
    break;
  }

  case 'assert-tool': {
    const ok = matchesTool(transcript, arg);
    console.log(ok ? `ok: used ${arg}` : `MISS: never used ${arg}`);
    process.exit(ok ? 0 : 1);
  }

  case 'assert-no-tool': {
    const ok = !matchesTool(transcript, arg);
    console.log(ok ? `ok: avoided ${arg}` : `UNEXPECTED: used ${arg}`);
    process.exit(ok ? 0 : 1);
  }

  case 'assert-final': {
    const ok = new RegExp(arg, 'i').test(transcript.finalText);
    console.log(ok ? `ok: answer matches /${arg}/` : `MISS: answer does not match /${arg}/`);
    process.exit(ok ? 0 : 1);
  }

  case 'judge': {
    const { pass, reason, broken } = judge(transcript, arg, arg2 || 'haiku');
    if (broken) {
      console.log(reason);
      process.exit(EXIT_UNSCORABLE);
    }
    console.log(`${pass ? 'ok' : 'MISS'}: ${reason}`);
    process.exit(pass ? 0 : 1);
  }

  default:
    console.error(`unknown command: ${command}`);
    process.exit(2);
}
