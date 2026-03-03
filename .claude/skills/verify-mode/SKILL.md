---
name: verify-mode
description: Verify the operational setup for a task
invocation: /verify-mode <task-description>
model: claude-opus-4-5
disable-model-invocation: false
---

# Verify Operational Setup

Analyze the task: **{{ARGUMENTS}}**

## Carbon Relay - Free Article Collection

- **Purpose**: Collect free carbon news articles directly
- **Command**: `./pipeline -sources=all-free -perSource=5`
- **Features**:
  - 39 free sources
  - No OpenAI API required
  - Email distribution, Notion integration
- **Use case**: Daily free article review

## Analysis

Based on the task "{{ARGUMENTS}}":

**Recommended command**:
```bash
./pipeline -sources=[appropriate sources] -perSource=5
```

## Verification Checklist

- [ ] Correct `-sources` flag (specific source or `all-free`)
- [ ] Appropriate `-perSource` count
- [ ] Output destination matches use case (email / Notion / JSON)
