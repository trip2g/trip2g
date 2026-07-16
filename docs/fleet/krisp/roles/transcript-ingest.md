---
description: Krisp meetings → transcript notes (cron ingest, deterministic)
fleet_id: codellm
mode: cron
cron_schedule: "* * * * *"
write_patterns:
  - transcripts/**
max_depth: 1
---
```python
import os
import json
import urllib.request

# Runs in codellm. codellm exposes KRISP_* to this child from ITS OWN env, per
# its operator expose-allowlist (CODELLM_EXPOSE_ENV), so they arrive as ordinary
# env variables. The role declares nothing about env; fleet holds no secrets.
base_url = os.environ['KRISP_BASE_URL'].rstrip('/')
token = os.environ['KRISP_TOKEN']


def api_post(path, payload):
    data = json.dumps(payload).encode()
    req = urllib.request.Request(
        base_url + path,
        data=data,
        headers={
            'Authorization': 'Bearer ' + token,
            'Content-Type': 'application/json',
        },
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read())


def api_get(path):
    req = urllib.request.Request(
        base_url + path,
        headers={'Authorization': 'Bearer ' + token},
    )
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read())


resp = api_post('/v2/meetings/list', {'page': 1, 'limit': 100, 'isOwner': True})
meetings = resp.get('data', {}).get('rows', [])

changes = []
for meeting in meetings:
    mid = meeting['id']
    name = meeting.get('name', mid)
    started_at = meeting.get('started_at', '')
    speakers = meeting.get('speakers', [])

    tree = api_get('/v2/block/' + mid + '/tree')

    lines = ['# ' + name, '', 'Date: ' + started_at, '']
    for child in tree.get('children', []):
        if child.get('block_type') != 'utterance':
            continue
        idx = child.get('speakerIndex', 0)
        if 0 < idx <= len(speakers):
            sp = speakers[idx - 1]
            speaker = sp.get('first_name', '') + ' ' + sp.get('last_name', '')
        else:
            speaker = 'Speaker ' + str(idx)
        speech = child.get('speech', {})
        start = speech.get('start', 0.0)
        text = speech.get('text', '')
        mins = int(start) // 60
        secs = int(start) % 60
        lines.append(speaker.strip() + ' | {:02d}:{:02d}'.format(mins, secs))
        lines.append(text)
        lines.append('')

    changes.append({'path': 'transcripts/' + mid + '.md', 'content': '\n'.join(lines)})

print(json.dumps({'changes': changes, 'answer': 'ingested ' + str(len(changes))}))
```
