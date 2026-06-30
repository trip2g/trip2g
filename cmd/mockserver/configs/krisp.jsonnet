// krisp.jsonnet — deterministic Krisp API stub for fleet e2e tests.
// Replicates cmd/krispmock/main.go field-for-field.
//
// Routes:
//   GET  /health              → 200 "ok"
//   POST /v2/meetings/list   → paginated synthetic meetings
//   GET  /v2/block/<id>/tree → block tree with utterances

local req = std.parseJson(std.extVar("request"));

local mockReqID = "00000000-0000-0000-0000-000000000001";
local meetingID1 = "aabbccddeeff00112233445566778800";
local meetingID2 = "1122334455667788aabbccddeeff0011";
local meetingID3 = "99aabbccddeeff001122334455667788";

local permEdit = "edit";
local blockTypeMeeting = "meeting";
local blockTypeUtt = "utterance";
local statusDone = "done";

local speakerID(n) = std.format("%032x", n);

// blockChildID replicates Go's blockChildID: meetingID[:24] + %08x(pos) when
// len(meetingID) >= 24, else %032x(pos).
local blockChildID(meetingID, pos) =
  if std.length(meetingID) < 24 then
    std.format("%032x", pos)
  else
    meetingID[:24] + std.format("%08x", pos);

// makeSpeakers replicates Go's makeSpeakers(names).
local makeSpeakers(names) = [
  {
    id: speakerID(i + 1),
    first_name: names[i],
    last_name: "Mock",
    email: names[i] + "@mock.example",
    photo: "",
  }
  for i in std.range(0, std.length(names) - 1)
];

// selfAccess is the shared self_access block used in meetings and trees.
local selfAccess = {
  is_owner: true,
  type: "personal",
  resources: {
    transcript: permEdit,
    meeting_note: permEdit,
    recording: permEdit,
    agenda: permEdit,
  },
};

// makeMeeting replicates Go's makeMeeting.
local makeMeeting(id, name, startedAt, duration, names) = {
  id: id,
  name: name,
  started_at: startedAt,
  duration: duration,
  speakers: makeSpeakers(names),
  is_demo: false,
  created_at: startedAt,
  includes_external_attendees: false,
  resources: {
    transcript: {
      status: "complete",
      processor: "krisp",
    },
    recording: false,
    recordings: [],
    meeting_notes: {},
  },
  thumbnails: [],
  user_interactions: {
    read: false,
    starred: false,
    hidden: false,
    listen_later: false,
    progress: null,
    is_new: true,
  },
  app_name: "Meet",
  parent_id: null,
  accesses: [],
  status: statusDone,
  self_access: selfAccess,
  is_private: false,
  highlight: null,
  tags: [],
  folders: [],
  storage_size: 0,
};

// syntheticMeetings replicates Go's syntheticMeetings().
local syntheticMeetings = [
  makeMeeting(meetingID1, "Team Sync Q1 Planning", "2026-01-15T10:00:00Z", 3600, ["Alice", "Bob"]),
  makeMeeting(meetingID2, "Product Demo and Feedback Session", "2026-02-20T14:30:00Z", 2700, ["Carol", "Dave"]),
  makeMeeting(meetingID3, "Engineering Architecture Review", "2026-03-10T09:00:00Z", 5400, ["Eve", "Frank", "Grace"]),
];

// treeUtterances replicates Go's treeUtterances(id).
local treeUtterances(id) =
  if id == meetingID1 then [
    { speakerIdx: 1, start: 0.0,  text: "Good morning, let us get started with the Q1 planning." },
    { speakerIdx: 2, start: 15.5, text: "I have prepared the roadmap items for this quarter." },
    { speakerIdx: 1, start: 40.0, text: "Great, let us go through each milestone in order." },
    { speakerIdx: 2, start: 65.0, text: "The first milestone is the launch of the new dashboard." },
  ]
  else if id == meetingID2 then [
    { speakerIdx: 1, start: 0.0,  text: "Welcome everyone to the product demo session." },
    { speakerIdx: 2, start: 12.0, text: "Thank you. Let me walk you through the new features." },
    { speakerIdx: 1, start: 35.0, text: "The search functionality looks very intuitive." },
    { speakerIdx: 2, start: 52.0, text: "We redesigned the search to reduce the number of clicks." },
  ]
  else if id == meetingID3 then [
    { speakerIdx: 1, start: 0.0,  text: "Today we are reviewing the architecture proposal." },
    { speakerIdx: 2, start: 18.0, text: "I will cover the data layer decisions first." },
    { speakerIdx: 3, start: 42.0, text: "The caching strategy needs to address cache invalidation." },
    { speakerIdx: 1, start: 68.0, text: "Agreed. Let us document the eviction policy." },
    { speakerIdx: 2, start: 90.0, text: "I will write up the ADR after this call." },
  ]
  else [
    { speakerIdx: 1, start: 0.0,  text: "Synthetic transcript for meeting " + id + "." },
    { speakerIdx: 2, start: 20.0, text: "Content is generated deterministically for testing." },
  ];

// buildBlockChildren replicates Go's buildBlockChildren.
local buildBlockChildren(meetingID, utterances) = [
  {
    id: blockChildID(meetingID, i),
    permission: permEdit,
    label: "label",
    block_type: blockTypeUtt,
    speakerIndex: utterances[i].speakerIdx,
    speech: {
      text: utterances[i].text,
      start: utterances[i].start,
    },
    resources: [],
    children: [],
    associated_to: [],
    "$version": 1,
    content: {
      is_edited: false,
      status: "ready",
      language: "en-US",
    },
  }
  for i in std.range(0, std.length(utterances) - 1)
];

// syntheticTree replicates Go's syntheticTree(id).
local syntheticTree(id) =
  local utterances = treeUtterances(id);
  {
    id: id,
    permission: permEdit,
    label: "label",
    block_type: blockTypeMeeting,
    resources: [],
    children: buildBlockChildren(id, utterances),
    associated_to: [],
    "$version": 1,
    content: {
      is_demo: false,
      status: statusDone,
    },
    self_access: selfAccess,
    accesses: {},
    access_requests: {},
    folders: [],
  };

if req.method == "GET" && req.path == "/health" then
  { bodyText: "ok", headers: { "Content-Type": "text/plain; charset=utf-8" } }

else if req.method == "POST" && req.path == "/v2/meetings/list" then
  local body = if req.body != "" then std.parseJson(req.body) else {};
  local page = if std.objectHas(body, "page") then body.page else 0;
  local rows = if page > 1 then [] else syntheticMeetings;
  {
    body: {
      code: 0,
      message: "success",
      data: {
        rows: rows,
        count: std.length(syntheticMeetings),
      },
      req_id: mockReqID,
    },
  }

else if req.method == "GET" && std.startsWith(req.path, "/v2/block/") && std.endsWith(req.path, "/tree") then
  // Extract <id> from /v2/block/<id>/tree by splitting on "/" and taking index 3.
  local parts = std.split(req.path, "/");
  local id = parts[3];
  { body: syntheticTree(id) }

else
  { status: 404, bodyText: "not found\n", headers: { "Content-Type": "text/plain; charset=utf-8" } }
