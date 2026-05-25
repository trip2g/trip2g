# How to count lines of code

Run these commands from the project root. Repeat every 5-10 months to track growth.

## Go — production (excluding codegen and tests)

```bash
# Hand-written production code
find . -type f -name "*.go" ! -path "*/vendor/*" ! -name "*_test.go" \
  | xargs grep -L "^// Code generated" \
  | xargs wc -l | tail -1

# Tests
find . -type f -name "*_test.go" ! -path "*/vendor/*" \
  | xargs wc -l | tail -1

# Codegen (gqlgen, sqlc, moq)
find . -type f -name "*.go" ! -path "*/vendor/*" \
  | xargs grep -l "^// Code generated" \
  | xargs wc -l | tail -1
```

## TypeScript and $mol — own code only

```bash
# assets/ui — exclude codegen dirs (/-/ and /-node/)
find assets/ui -type f -name "*.ts" \
  ! -path "*/-/*" ! -path "*/-node/*" ! -path "*/.omc/*" \
  | xargs wc -l | tail -1

# $mol view.tree templates
find assets/ui -type f -name "*.view.tree" \
  ! -path "*/-/*" ! -path "*/-node/*" ! -path "*/.omc/*" \
  | xargs wc -l | tail -1

# obsidian-sync plugin (exclude browser/ and node_modules)
find obsidian-sync -type f \( -name "*.ts" -o -name "*.js" \) \
  ! -name "*.spec.*" ! -name "*.test.*" \
  ! -path "*/node_modules/*" ! -path "*/browser/*" \
  | xargs wc -l | tail -1

# obsidian-sync tests
find obsidian-sync -type f \( -name "*.spec.ts" -o -name "*.spec.js" \) \
  ! -path "*/node_modules/*" ! -path "*/browser/*" \
  | xargs wc -l | tail -1

# Vendor bundles (NOT own code, for reference)
find assets/tiptap assets/milkdown -type f \( -name "*.ts" -o -name "*.js" \) \
  | xargs wc -l | tail -1
```

## SQL

```bash
# Migrations
find db/migrations -name "*.sql" | xargs wc -l | tail -1

# Hand-written queries
wc -l queries.read.sql queries.write.sql
```

## GraphQL schema

```bash
wc -l internal/graph/schema.graphqls
```

## E2E tests

```bash
find e2e -name "*.spec.js" | xargs wc -l | tail -1
```

## Admin pages count

```bash
for f in assets/ui/admin/menu/*/; do
  name=$(basename $f)
  count=$(grep -h "spreads \*" -A 200 $f*.view.tree 2>/dev/null | grep -c "^\t\t[a-z]")
  echo "$name: $count"
done
```

## Git stats

```bash
git log --oneline | wc -l          # commit count
git log --format="%ad" --date=format:"%Y-%m" | tail -1  # project start
```

## What NOT to count as own code

- `assets/tiptap/` — Tiptap vendor bundle
- `assets/milkdown/` — old Milkdown bundle (replaced)
- `assets/ui/-/` and `assets/ui/-node/` — $mol codegen
- `internal/graph/schema.graphqls` codegen output (gqlgen resolvers etc.)
- Go files with `// Code generated` header
