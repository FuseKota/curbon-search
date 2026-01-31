---
name: commit-pattern
description: Create a properly formatted commit message
invocation: /commit-pattern <commit-description>
model: claude-opus-4-5
disable-model-invocation: false
---

# Create Commit Message

Generate a commit message for: **{{ARGUMENTS}}**

## Carbon Relay Commit Convention

### Format

```
<type>: <brief description>

<detailed explanation>
- Bullet point 1
- Bullet point 2

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

### Types

- `feat:` - New feature (new source, new mode)
- `fix:` - Bug fix (scraping error, parsing issue)
- `docs:` - Documentation only
- `refactor:` - Code restructuring
- `test:` - Test additions

### Examples

**Good**:
```
feat: Add PwC Japan source with JSON parsing

- Implemented 3-level unescaping for embedded JSON
- Added date parsing for YYYY-MM-DD format
- Tested with 5 articles successfully

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

**Bad**:
```
update code
```

## Generated Commit Message

Based on "{{ARGUMENTS}}":

```
[Generate appropriate commit message here]
```

## Ready to Commit

Run:
```bash
git add [files]
git commit -m "$(cat <<'EOF'
[Generated message here]
EOF
)"
```
