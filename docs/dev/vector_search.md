# Vector Search

Semantic search using OpenAI embeddings for finding similar notes by meaning.

## Overview

Vector search enables semantic similarity matching - finding notes by meaning rather than exact keywords. The `similarNotes` GraphQL query returns notes that are semantically similar to a given note.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Similar Notes Flow                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Note: "/my-article"                                            │
│       │                                                          │
│       ▼                                                          │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │ In-Memory Cache (NoteViews)                                  ││
│  │  - Note with Embedding []float32 (1536 dimensions)          ││
│  │  - Loaded on startup via SQL JOIN                           ││
│  └─────────────────────────────────────────────────────────────┘│
│       │                                                          │
│       ▼                                                          │
│  Cosine Similarity Calculation (in-memory)                       │
│       │                                                          │
│       ▼                                                          │
│  Top N Similar Notes (filtered by CanReadNote)                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Database Schema

Embeddings are cached in SQLite:

```sql
create table note_version_embeddings (
    version_id integer primary key references note_versions(id) on delete cascade,
    embedding blob not null,
    model_id integer not null,
    content_hash blob not null,
    tokens integer not null,
    created_at datetime not null default (datetime('now'))
);

create index idx_note_version_embeddings_model_id on note_version_embeddings(model_id);
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `version_id` | integer | FK to note_versions.id |
| `embedding` | blob | float32 array as bytes (1536 dims = 6KB) |
| `model_id` | integer | Model constant (1=small, 2=large, 3=ada) |
| `content_hash` | blob | SHA256 of title+content to detect changes |
| `tokens` | integer | Tokens consumed to generate this embedding |
| `created_at` | datetime | When embedding was generated |

## Embedding Generation

### Background Job

Embeddings are generated asynchronously via goqite queue:

```
Note Created/Updated (HandleLatestNotesAfterSave)
    │
    ▼
Enqueue GenerateNoteVersionEmbedding job
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│ Background Worker                                            │
│  1. Get note from LatestNoteViews cache                     │
│  2. Calculate content hash (SHA256 of title+content)        │
│  3. Check if embedding exists with same hash → skip         │
│  4. Call OpenAI embeddings API                              │
│  5. Store in note_version_embeddings                        │
└─────────────────────────────────────────────────────────────┘
```

### Cronjob for Bulk Regeneration

The `regenerate_note_embeddings` cronjob runs:
- Daily at 3:00 AM
- On server startup (ExecuteAfterStart: true)

It compares content hashes and enqueues jobs for notes with stale/missing embeddings.

## In-Memory Cache

Embeddings are loaded into `NoteView.Embedding` field via SQL JOIN when notes are loaded:

```sql
-- AllLatestNotes query includes embedding
select value as path, p.id as path_id, v.id as version_id, content, v.created_at, e.embedding
  from note_paths p
  join note_versions v on p.id = v.path_id and p.version_count = v.version
  left join note_version_embeddings e on v.id = e.version_id
 where p.hidden_by is null;
```

This means:
- No database queries during `similarNotes` requests
- Memory usage: ~6MB for 1000 notes (1536 floats × 4 bytes × 1000)
- Embeddings are refreshed when notes are reloaded

## GraphQL API

### similarNotes Query

```graphql
input SimilarNotesInput {
  noteId: String!    # Note permalink
  limit: Int         # Max results (default: 5, max: 20)
}

type SimilarNote {
  score: Float!      # 0-1, higher is more similar
  note: PublicNote!
}

type Query {
  similarNotes(input: SimilarNotesInput!): [SimilarNote!]!
}
```

### Example

```graphql
query {
  similarNotes(input: { noteId: "/my-article", limit: 5 }) {
    score
    note {
      id
      title
      path
    }
  }
}
```

## Configuration

### Feature Flag

```bash
FEATURES='{"vector_search": {"enabled": true, "model": "text-embedding-3-small"}}'
```

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `FEATURES` | No | JSON with feature configuration |
| `OPENAI_API_KEY` | When enabled | OpenAI API key for embeddings |

### Models

| Model | ID | Dimensions | Cost |
|-------|-----|------------|------|
| `text-embedding-3-small` | 1 | 1536 | $0.02/1M tokens |
| `text-embedding-3-large` | 2 | 3072 | $0.13/1M tokens |
| `text-embedding-ada-002` | 3 | 1536 | $0.10/1M tokens |

## Implementation Details

### Package Structure

```
internal/
├── features/
│   ├── features.go        # Features struct, Parse()
│   └── vector_search.go   # VectorSearchConfig, EmbeddingModel
├── openai/
│   └── client.go          # OpenAI client wrapper
├── case/
│   ├── similarnotes/
│   │   └── resolve.go     # similarNotes query resolver
│   └── backjob/
│       └── generatenoteversionembedding/
│           ├── job.go     # Job registration
│           └── resolve.go # Embedding generation logic
└── noteloader/
    └── loader.go          # Loads embeddings via SQL JOIN
```

### Cosine Similarity

```go
func cosineSimilarity(a, b []float32) float64 {
    var dotProduct, normA, normB float64
    for i := range a {
        dotProduct += float64(a[i]) * float64(b[i])
        normA += float64(a[i]) * float64(a[i])
        normB += float64(b[i]) * float64(b[i])
    }
    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

## Cost Estimation

OpenAI embedding costs for `text-embedding-3-small`:

| Notes | Avg Tokens/Note | Total Tokens | Cost |
|-------|-----------------|--------------|------|
| 100 | 500 | 50,000 | $0.001 |
| 1,000 | 500 | 500,000 | $0.01 |
| 10,000 | 500 | 5,000,000 | $0.10 |

## Graceful Degradation

- **Vector search disabled**: `similarNotes` returns empty array
- **Note has no embedding**: Note is excluded from results
- **OpenAI API error**: Job is retried by goqite

## Future Improvements

1. **Hybrid search** - Combine vector similarity with bleve text search
2. **Batch embedding generation** - Process multiple notes in one API call
3. **Local embeddings** - Use local model to avoid API costs
4. **Semantic query search** - Generate query embedding for search
