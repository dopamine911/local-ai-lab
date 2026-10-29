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
