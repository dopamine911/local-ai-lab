#!/usr/bin/env bash
# Host setup for zaphod or marvin, after a fresh Ubuntu 26.04 server install (see setup/REBUILD.md).
# Run on the machine itself:   sudo ~/homelab/setup/host.sh
# Safe to run again: every step checks first and skips what is already done.
set -euo pipefail
[ "$(id -u)" = 0 ] || { echo "run with sudo"; exit 1; }
U=${SUDO_USER:?run with sudo from your normal account}
H=$(hostname)
step() { echo; echo "== $*"; }

case "$H" in
  zaphod|marvin) ;;
  *) echo "hostname is '$H'; expected zaphod or marvin"; exit 1 ;;
esac

step "1. packages"
apt-get update -q
apt-get install -y -q docker.io docker-compose-v2 docker-buildx mesa-vulkan-drivers vulkan-tools radeontop btop \
  python3-venv curl git ssh-import-id

step "2. groups for $U (docker, GPU)"
usermod -aG docker,render,video "$U"

step "3. SSH keys from GitHub (zaphod's key is one of them)"
sudo -u "$U" ssh-import-id gh:dopamine911

step "4. Tailscale"
command -v tailscale >/dev/null || curl -fsSL https://tailscale.com/install.sh | sh
if ! tailscale status >/dev/null 2>&1; then
  echo "Log in: open the link below and approve $H (plain login, no --ssh)."
  tailscale up
fi
# llama-swap listens on 127.0.0.1:8080 only; this shares it with the tailnet (not the internet).
tailscale serve status 2>/dev/null | grep -q 'localhost:8080' || tailscale serve --bg --tcp=8080 tcp://localhost:8080

step "5. model folder"
install -d -o "$U" /srv/models

step "6. ~/llama-swap link"
sudo -u "$U" ln -sfn "/home/$U/homelab/$H/llama-swap" "/home/$U/llama-swap"

step "6b. time zone America/Chicago (Daniel's local time; was Ubuntu's default UTC until 2026-10-30)"
# Logs, file names and cron use it (the nightly backup's 04:17 is Chicago time). Containers stay UTC unless they set TZ.
[ "$(timedatectl show -p Timezone --value)" = America/Chicago ] || timedatectl set-timezone America/Chicago
timedatectl | grep 'Time zone'

if [ "$H" = marvin ]; then
  step "7. marvin: let the GPU use 120 GB of the shared memory (default is about half)"
  f=/etc/default/grub.d/90-gpu-memory.cfg
  want='GRUB_CMDLINE_LINUX_DEFAULT="$GRUB_CMDLINE_LINUX_DEFAULT amdgpu.gttsize=122880 ttm.pages_limit=31457280"'
  if [ "$(cat $f 2>/dev/null)" != "$want" ]; then
    echo "$want" > $f; update-grub; echo ">>> reboot needed for this setting"
  else echo "already set"; fi

  # Measured 2026-10-29 (Strata, 32K prompt): 758 → 757 tok/s reading, no change; kept as harmless.
  step "8. marvin: tuned profile accelerator-performance (CPU cores stay out of deep sleep; strix-halo-toolboxes.com)"
  dpkg -s tuned >/dev/null 2>&1 || { apt-get update -q; apt-get install -y tuned; }
  systemctl enable --now tuned
  if [ "$(tuned-adm active 2>/dev/null | sed 's/.*: //')" != accelerator-performance ]; then
    tuned-adm profile accelerator-performance; fi
  tuned-adm active
fi

step "done. Log out and back in (new groups), then continue with setup/REBUILD.md"
