---
name: test-all
description: Test all sources quickly
---

Quick test of all sources (1 article each):

```bash
./pipeline -sources=all -perSource=1 -queriesPerHeadline=0 -out=/tmp/test_all.json

echo "Results:"
cat /tmp/test_all.json | jq 'length'
```
