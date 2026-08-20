// llm.jsonnet — deterministic OpenAI-compatible chat completions stub for fleet e2e tests.
// Replicates cmd/llmmock/main.go field-for-field.
//
// Routes:
//   GET  /health                  → 200 "ok"
//   POST /v1/chat/completions     → mock tool-call response

local req = std.parseJson(std.extVar("request"));

// Replicates Go's extractInstruction(systemContent):
//   find "\nInstruction:\n" in systemContent;
//   clip at "\n\nAccess scope" if present;
//   strip leading YAML frontmatter (---\n…\n---\n);
//   TrimSpace the result.
//   Falls back to the whole systemContent when the prefix is absent.

local prefix = "\nInstruction:\n";
local suffix = "\n\nAccess scope";

local extractInstruction(sc) =
  local starts = std.findSubstr(prefix, sc);
  if std.length(starts) == 0 then
    sc
  else
    local afterPrefix = sc[starts[0] + std.length(prefix):];
    local ends = std.findSubstr(suffix, afterPrefix);
    local clipped = if std.length(ends) > 0 then afterPrefix[:ends[0]] else afterPrefix;
    local stripped =
      if std.startsWith(clipped, "---\n") then
        local inner = clipped[4:];
        local innerEnds = std.findSubstr("\n---\n", inner);
        if std.length(innerEnds) > 0 then
          inner[innerEnds[0] + 5:]
        else
          clipped
      else
        clipped;
    std.stripChars(stripped, " \t\n\r");

// truncate replicates Go's truncate(s, n):
//   TrimSpace, then clip to n Unicode code points.
local truncate(s, n) =
  local trimmed = std.stripChars(s, " \t\n\r");
  if std.length(trimmed) <= n then trimmed else trimmed[:n];

// Pipeline routing: roles that name their source note in backticks (the krisp
// chain does) get an answer written to the next stage's folder, so a transcript
// becomes segments and segments become a wiki note. A role that names no
// backticked path keeps the historic fixed target — the fleet e2e relies on it.
local backtickedPaths(s) =
  local parts = std.split(s, "`");
  [parts[i] for i in std.range(0, std.length(parts) - 1)
   if i % 2 == 1 && std.endsWith(parts[i], ".md")];

local defaultTarget = "segments/sample.md";

local nextStagePath(instruction) =
  local transcripts = std.filter(
    function(p) std.length(std.findSubstr("transcripts/", p)) > 0,
    backtickedPaths(instruction),
  );
  local segments = std.filter(
    function(p) std.length(std.findSubstr("segments/", p)) > 0,
    backtickedPaths(instruction),
  );
  if std.length(transcripts) > 0 then
    std.strReplace(transcripts[0], "transcripts/", "segments/")
  else if std.length(segments) > 0 then
    std.strReplace(segments[0], "segments/", "wiki/")
  else
    defaultTarget;

if req.method == "GET" && req.path == "/health" then
  { bodyText: "ok", headers: { "Content-Type": "text/plain; charset=utf-8" } }

else if req.method == "POST" && req.path == "/v1/chat/completions" then
  local body = std.parseJson(req.body);
  local model = if std.objectHas(body, "model") && body.model != "" then body.model else "mock";
  local messages = if std.objectHas(body, "messages") then body.messages else [];

  // Scan messages for tool role and system content.
  local hasToolResult = std.length(std.filter(function(m) m.role == "tool", messages)) > 0;
  local sysMsgs = std.filter(function(m) m.role == "system", messages);
  local instructionContent =
    if std.length(sysMsgs) > 0 then
      extractInstruction(sysMsgs[0].content)
    else
      "Begin.";

  local bodyLen = std.length(req.body);

  // Detect kanban triage scenario: instruction rendered from roles/triage.md embeds
  // the board content which contains "@status:doing".  All other roles (e.g. the
  // transcript agent) do not mention "@status:doing".
  local isTriageCall = std.length(std.findSubstr("@status:doing", instructionContent)) > 0;

  local toolCall =
    if hasToolResult then
      {
        id: "t2",
        type: "function",
        "function": {
          name: "finish",
          // Preserve scenario-specific answer so tests can distinguish if needed.
          arguments: if isTriageCall then '{"answer":"triaged"}' else '{"answer":"done"}',
        },
      }
    else if isTriageCall then
      // Triage scenario: append @triaged to the @status:doing line via patch_note.
      // Go marshals map keys alphabetically: "find" < "path" < "replace".
      local triageArgs = { find: "@status:doing", path: "boards/sprint.md", replace: "@status:doing @triaged" };
      {
        id: "t1",
        type: "function",
        "function": {
          name: "patch_note",
          arguments: std.manifestJsonMinified(triageArgs),
        },
      }
    else
      local content = "processed: " + truncate(instructionContent, 200);
      // arguments must be a JSON string (not an object) — matches json.Marshal output.
      // Go marshals map keys alphabetically: "content" < "path".
      local argsObj = { content: content, path: nextStagePath(instructionContent) };
      {
        id: "t1",
        type: "function",
        "function": {
          name: "write_note",
          arguments: std.manifestJsonMinified(argsObj),
        },
      };

  {
    body: {
      id: "cmpl-mock",
      object: "chat.completion",
      model: model,
      choices: [{
        index: 0,
        message: {
          role: "assistant",
          content: null,
          tool_calls: [toolCall],
        },
        finish_reason: "tool_calls",
      }],
      usage: {
        prompt_tokens: std.floor(bodyLen / 4),
        completion_tokens: 5,
        total_tokens: std.floor(bodyLen / 4) + 5,
      },
    },
  }

else
  { status: 404, bodyText: "not found\n", headers: { "Content-Type": "text/plain; charset=utf-8" } }
