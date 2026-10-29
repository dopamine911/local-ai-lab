# marvin setup log

Bosgame M5, Ryzen AI Max+ 395 (Strix Halo, 32 threads), 128 GB, board AXB35-02, BIOS 1.07.
One step at a time; each step records what was checked and the result.

| # | Date | Step | Result |
| --- | --- | --- | --- |
| 1 | 2026-10-29 | Ubuntu 26.04.1 server, UEFI install | UEFI confirmed; kernel 7.0.0-42 |
| 2 | 2026-10-29 | BIOS: UMA frame buffer 512 MB | GPU VRAM 512 MB; Linux sees 122 GB RAM |
| 3 | 2026-10-29 | Root volume 100 GB → full SSD (`lvextend -l +100%FREE -r`) | 1.9 TB, 1.8 TB free |
| 4 | 2026-10-29 | Tailscale (plain `tailscale up`, no `--ssh`); GitHub keys via `ssh-import-id gh:dopamine911` | zaphod logs in with `ssh marvin`, no password |
| 5 | 2026-10-29 | Checked GPU memory limit (GTT) before any change | **62,814 MB** (default: about half of RAM). Docker not installed yet. |
| 6 | 2026-10-29 | Boot settings `amdgpu.gttsize=122880 ttm.pages_limit=31457280` in `/etc/default/grub.d/90-gpu-memory.cfg` | GPU limit 62,814 → **122,880 MB**; both settings in `/proc/cmdline`. Not yet tested with a real model load. `amd_iommu=off` not set (separate test later). |
| 7 | 2026-10-29 | Docker 29.1.3 (Ubuntu `docker.io`, same as zaphod); user in docker, render, video | `docker` works without sudo |
| 8 | 2026-10-29 | Repo cloned to `~/homelab`, `~/llama-swap` → `~/homelab/marvin/llama-swap`; image `llama-swap-marvin:267` built | llama.cpp sees `Vulkan0: Radeon 8060S (RADV STRIX_HALO)`, **123,392 MiB** |
| 9 | 2026-10-29 | `/srv/models` created; downloading Qwen3.8-Flash-Next UD-IQ4_XS (93.7 GB, 3 files) + MTP Q8_0 (4.1 GB), unsloth rev 766911a6, with sha256 check (now `setup/models.sh` + `setup/models-marvin.txt`) | running, ~40 MB/s |
| 10 | 2026-10-29 | Beszel agent 0.21.1-alpine (disk health, systemd, containers), listening on the Tailscale IP only | zaphod reaches it on 100.101.162.9:45876; add in the Beszel page (setup/REBUILD.md 6.3) |
