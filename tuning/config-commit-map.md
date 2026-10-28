# Old llama-swap config IDs

Before 2026-10-28 18:05, llama-swap lived in its own repo (~/llama-swap). When it moved into this repo, its history was rewritten to remove a private notes file, which changed the commit IDs. MLflow runs logged before the move carry the old IDs in their `config_commit` tag. This table maps them.

| Old ID (in MLflow) | New ID (this repo) | Change |
| --- | --- | --- |
| ab39924, 392ffbd, 98eb348 | f4b76e9 | 2x R9700, MTP depth 2, VIDMEM fix (the later two only changed notes) |
| ec7159e, d397f1d | a7f5b75 | coder-tb: 256K context, q8_0 KV, MTP (d397f1d only changed notes) |
| 9472959 | 9d6e940 | flint on card 2 |

The original repo is backed up in ~/backups/llama-swap-before-move-2026-10-28.tar.gz.
