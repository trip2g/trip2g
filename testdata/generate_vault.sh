#!/bin/bash

# Obsidian Link Resolution Test Vault Generator
# Creates minimal test structure for link resolution testing

VAULT="vault"

echo "Creating test vault: $VAULT"
rm -rf "$VAULT"
mkdir -p "$VAULT"/{folder,assets,projectA,projectB}

# ============================================================================
# Test 1: Unique filenames - simple case
# ============================================================================

cat > "$VAULT/unique.md" << 'EOF'
# Unique File
Link: [[deep]] - should find /folder/deep.md
EOF

cat > "$VAULT/folder/deep.md" << 'EOF'
# Deep File
Found me! Path: /folder/deep.md
EOF

# ============================================================================
# Test 2: Duplicate filenames - priority test (CRITICAL)
# ============================================================================

cat > "$VAULT/dup.md" << 'EOF'
# Duplicate in ROOT
I'm at /dup.md
EOF

cat > "$VAULT/folder/dup.md" << 'EOF'
# Duplicate in FOLDER
I'm at /folder/dup.md
EOF

cat > "$VAULT/folder/source.md" << 'EOF'
# Source File (in /folder/)
Test: [[dup]] - goes to ROOT, not local! ⚠️
Local: [[./dup]] - this one stays local ✅
Explicit: [[folder/dup]] - also local ✅
EOF

# ============================================================================
# Test 3: Multiple conflicts across subfolders
# ============================================================================

cat > "$VAULT/projectA/README.md" << 'EOF'
# Project A
Link: [[guide]] - ambiguous!
Explicit: [[projectA/guide]] - clear
EOF

cat > "$VAULT/projectA/guide.md" << 'EOF'
# Guide A
Path: /projectA/guide.md
EOF

cat > "$VAULT/projectA/_index.md" << 'EOF'
# Project A Index
This is the index page for Project A
EOF

cat > "$VAULT/projectB/README.md" << 'EOF'
# Project B
Link: [[README]] - ambiguous!
EOF

cat > "$VAULT/projectB/guide.md" << 'EOF'
# Guide B
Path: /projectB/guide.md
EOF

cat > "$VAULT/projectB/_index.md" << 'EOF'
# Project B Index
This is the index page for Project B
EOF

# ============================================================================
# Test 4: Assets (images) with duplicates
# ============================================================================

cat > "$VAULT/img-test.md" << 'EOF'
# Image Test
Global: ![[test.png]] - which one?
Explicit: ![[assets/test.png]] - clear
EOF

# Create test images (minimal 1x1 PNGs with different colors)
# Alternative with network: curl -s "https://placehold.co/600x200?text=/test.png" -o "$VAULT/test.png"
echo "Creating test images..."
# Red pixel (root image)
curl -s "https://placehold.co/600x200?text=/test.png" -o "$VAULT/test.png"
# Green pixel (assets image)
curl -s "https://placehold.co/600x200?text=/assets/test.png" -o "$VAULT/assets/test.png"
# Blue pixel (folder image)
curl -s "https://placehold.co/600x200?text=/folder/test.png" -o "$VAULT/folder/test.png"

# ============================================================================
# Test 5: Headers and blocks
# ============================================================================

cat > "$VAULT/headers.md" << 'EOF'
# Headers Test

## Section One
Content here.

## Section Two
More content. ^block-id

Link to header: [[headers#Section One]]
Link to block: [[headers#^block-id]]
EOF

# ============================================================================
# README with all tests
# ============================================================================

cat > "$VAULT/README.md" << 'EOF'
# Test Vault

## Tests
1. [[unique]] - unique filename
2. [[folder/source]] - duplicate filename priority
3. [[projectA/README]] - multiple conflicts
4. [[img-test]] - image resolution
5. [[headers]] - headers/blocks

## Key Test: Duplicate Priority
From [[folder/source]]:
- `[[dup]]` → /dup.md (root!) ⚠️
- `[[folder/dup]]` → /folder/dup.md ✅

## Expected Behavior
Global resolution, root priority on conflicts.
EOF
