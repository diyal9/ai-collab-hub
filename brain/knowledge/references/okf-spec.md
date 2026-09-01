---
type: Reference
title: Open Knowledge Format v0.1
description: Vendor-neutral markdown + YAML frontmatter knowledge bundle spec.
tags: [reference, okf, google]
timestamp: 2026-06-24T00:00:00Z
resource: https://github.com/GoogleCloudPlatform/knowledge-catalog/tree/main/okf
---

# Required

- Every concept `.md` has YAML frontmatter with non-empty `type`
- Reserved: `index.md`, `log.md`

# Recommended Frontmatter

`title`, `description`, `resource`, `tags`, `timestamp`

# Links

Bundle-relative: `[concept](/path/to/concept.md)`

# Citations

[1] [OKF SPEC.md](https://github.com/GoogleCloudPlatform/knowledge-catalog/blob/main/okf/SPEC.md)
