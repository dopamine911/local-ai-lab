# Rebuild zaphod or marvin from scratch

Everything a machine needs is either **code in this repo** (run a script) or a **numbered step below** (for things
that can't be scripted: BIOS, the installer, web logins). If a step here is still manual, that's a to-do for code.

Rule: every change to a machine lands here or in a script **in the same commit**.

What you need besides this repo:
- **Secrets** (not in git): the `.env` files listed in section 5. Keep a copy in your password manager.
- **Accounts:** GitHub (dopamine911; your SSH keys come from there), Tailscale, Cloudflare (DNS for the domain).
- **Data** that is not in git: Open WebUI chats and settings, Beszel history, MLflow runs. See section 6.

## 1. BIOS (by hand, before installing)

| Machine | Setting | Value | Why |
| --- | --- | --- | --- |
| zaphod | CSM Support | **Enabled** | Ubuntu was installed in legacy mode; it won't boot without CSM |
| zaphod | Resizable BAR | Off (forced by CSM) | Why `GGML_VK_DISABLE_HOST_VISIBLE_VIDMEM=1` is needed |
| marvin | UMA frame buffer (GPU memory) | **512 MB** | Linux gives the GPU shared memory instead (set in `host.sh`) |

BIOS versions seen: zaphod board Gigabyte B550I AORUS PRO AX; cards 113-APM107573-101 (card 1), 113-R9700AT-F40 (card 2).
marvin board AXB35-02, BIOS 1.07.

If zaphod is ever reinstalled: install in **UEFI** mode instead. Then CSM can be turned off and Resizable BAR on,
which may make the `GGML_VK_DISABLE_HOST_VISIBLE_VIDMEM` switch unnecessary. Untested: measure before and after.

## 2. Ubuntu installer (by hand)

1. Ubuntu Server 26.04, booted from the USB stick's **"UEFI:"** entry.
2. Hostname `zaphod` or `marvin`; username `daniel`.
3. Storage: whole disk with LVM. On the summary screen, **edit `ubuntu-lv` and set it to the maximum size**
   (the default is only 100 GB). If you forget: `sudo lvextend -l +100%FREE -r /dev/ubuntu-vg/ubuntu-lv`.
4. Tick **Install OpenSSH server**. No extra snaps.
5. Network: Ethernet cable if possible (marvin is on Wi-Fi as of 2026-10-29).

Good: `ls /sys/firmware/efi` works; `df -h /` shows the full disk.

## 3. Host setup (script)

On the machine, logged in as daniel:
```
sudo apt update && sudo apt full-upgrade -y
git clone https://github.com/dopamine911/local-ai-lab.git ~/homelab
sudo ~/homelab/setup/host.sh
```
It installs Docker and tools, adds you to the docker/render/video groups, imports your SSH keys from GitHub,
logs in to Tailscale (open the link it prints), shares port 8080 on the tailnet, makes `/srv/models`, links
`~/llama-swap`, and on marvin sets the GPU memory boot setting. Then **reboot**.

Good: from zaphod, `ssh marvin` (or the other way) works without a password; `docker ps` works without sudo.

zaphod only, not yet in the script: the firewall (ufw) is on, with rules that were never written down.
To do: run `sudo ufw status numbered` on zaphod and add the rules to `host.sh`.

## 4. Models (script, takes hours)

```
tmux new -s models
~/homelab/setup/models.sh
```
Downloads everything in `setup/models-<machine>.txt` (pinned versions, sha256 checked). Good: ends with `MODELS-OK`.
It lists any file it can't download by itself ("get by hand").

## 5. Secrets and services

Restore the `.env` files from your password manager (or make new ones; each `.env.example` says how):

| File | Keys | If lost |
| --- | --- | --- |
| `zaphod/llama-swap/.env` | `ZAPHOD_KEY` | New one: `printf 'ZAPHOD_KEY=%s\n' "$(openssl rand -hex 24)"`; then update hub/.env and every app that uses it |
| `marvin/llama-swap/.env` | `MARVIN_KEY`, `BESZEL_KEY`, `TAILSCALE_IP` | New `MARVIN_KEY` as above; `BESZEL_KEY` = same as hub/.env; `TAILSCALE_IP` = `tailscale ip -4` |
| `hub/.env` | `DOMAIN`, `TS_IP`, `CF_API_TOKEN`, `ZAPHOD_KEY`, `MARVIN_KEY`, `BESZEL_KEY`, `OPEN_TERMINAL_API_KEY`, `SEARXNG_SECRET`, `WEBUI_SECRET_KEY` | `CF_API_TOKEN`: new token in Cloudflare (Zone → DNS → Edit). `BESZEL_KEY`: shown in Beszel's "Add system". Others: `openssl rand -hex 32` |

Then build and start (each `compose.yaml` has its `build:` lines):

**marvin**
```
cd ~/llama-swap && docker compose build && docker compose up -d
docker compose --profile agent up -d beszel-agent
```
**zaphod**
```
cd ~/llama-swap && docker compose build && docker compose up -d
cd ~/homelab/hub && docker compose --profile images build && docker compose up -d
docker compose --profile agent up -d beszel-agent
```
Good: `docker compose logs --tail 40 | grep 'Health check passed'` lists the preloaded models.

## 6. Web pages and outside services (by hand for now; each is a to-do for code)

1. **Cloudflare DNS:** `A` records `ai.dopamine911.com` and `*.ai.dopamine911.com` → zaphod's Tailscale IP, DNS only (grey cloud).
2. **Open WebUI** (`https://chat.<DOMAIN>`): first account becomes admin. Settings: see README.md, "Open WebUI tools".
3. **Beszel** (`https://monitor.<DOMAIN>`): create the account; **Add system** for each machine:
   zaphod = host `127.0.0.1`, port `45876`; marvin = host `100.101.162.9` (its Tailscale IP), port `45876`.
   Then `~/homelab/hub/beszel-monitors.sh` (asks for the Beszel login) adds the network monitors.
4. **Models panel:** copy `models-panel/index.html` from the claude.ai project into `hub/models-panel/`.

Data not in git (lost in a rebuild unless backed up): Open WebUI (`hub/data/open-webui`), Beszel (`hub/data/beszel`),
MLflow (`hub/data/mlflow`). Backups: not set up yet (to-do).
