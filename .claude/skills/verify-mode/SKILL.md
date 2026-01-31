---
name: verify-mode
description: Verify which operational mode a task relates to
invocation: /verify-mode <task-description>
model: claude-opus-4-5
disable-model-invocation: false
---

# Verify Operational Mode

Analyze the task: **{{ARGUMENTS}}**

## Carbon Relay's Two Modes

### 🟢 Mode 1: Free Article Collection
- **Purpose**: Collect free carbon news articles directly
- **Command**: `-queriesPerHeadline=0`
- **Features**:
  - 16 free sources (no Carbon Pulse/QCI)
  - No OpenAI API required
  - Fast execution (5-15 seconds)
  - Output: Email distribution
- **Use case**: Daily free article review

### 🔵 Mode 2: Paid Article Matching
- **Purpose**: Find free articles related to paid headlines
- **Command**: `-queriesPerHeadline=3`
- **Features**:
  - Carbon Pulse + QCI headlines
  - OpenAI search + IDF matching
  - Notion integration
- **Use case**: Weekly paid article deep dive

## Analysis

Based on the task "{{ARGUMENTS}}", this relates to:

**Mode: [Determine from context]**

**Reasoning**: [Explain why]

**Recommended flags**:
```bash
./pipeline -sources=[appropriate sources] -perSource=5 -queriesPerHeadline=[0 or 3]
```

## Verification Checklist

- [ ] Correct `-sources` flag (all-free vs carbonpulse,qci)
- [ ] Correct `-queriesPerHeadline` (0 vs 3)
- [ ] OpenAI API key set if Mode 2
- [ ] Output destination matches use case (email vs Notion)
