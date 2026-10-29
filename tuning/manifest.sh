#!/usr/bin/env bash
# Print the machine and software facts that belong with every test result, as key=value lines.
# Run on the machine that served the model:  ./manifest.sh   (zaphod)   or   ssh marvin bash -s < manifest.sh
# Used by ./log; every line becomes an "env.<key>" tag on the MLflow run. Missing values print as "unknown".
# Only stable facts: no temperatures, free memory or clock speeds. No secrets.
v() { local x; x=$("$@" 2>/dev/null) && [ -n "$x" ] && echo "$x" || echo unknown; }
echo "host=$(hostname)"
echo "kernel=$(uname -r)"
echo "mesa=$(v dpkg-query -W -f='${Version}' mesa-vulkan-drivers)"
echo "board=$(v cat /sys/class/dmi/id/board_name) BIOS $(v cat /sys/class/dmi/id/bios_version)"
echo "cpu=$(v sh -c "grep -m1 'model name' /proc/cpuinfo | cut -d: -f2- | sed 's/^ *//'")"
echo "ram_gb=$(v sh -c "awk '/MemTotal/ {printf \"%d\", \$2/1048576}' /proc/meminfo")"
# GPUs: PCI address and VBIOS of every AMD GPU. Names come from llama.cpp (vulkan_devices): lspci's are generic on Strix Halo
gpus=""
for d in /sys/class/drm/card[0-9]*/device; do
  [ -r "$d/vendor" ] && [ "$(cat "$d/vendor")" = 0x1002 ] || continue
  pci=$(basename "$(readlink -f "$d")"); pci=${pci#0000:}
  gpus+="${gpus:+; }$pci VBIOS $(v cat "$d/vbios_version")"
done
echo "gpus=${gpus:-unknown}"
# The llama-swap container: image, llama.cpp commit, llama-swap version
if docker inspect llama-swap >/dev/null 2>&1; then
  echo "image=$(docker inspect llama-swap --format '{{.Config.Image}}')"
  echo "image_id=$(docker inspect llama-swap --format '{{.Image}}' | cut -c8-19)"
  echo "llama_cpp_commit=$(v sh -c "docker exec llama-swap llama-server --version 2>&1 | sed -n 's/.*commit \([0-9a-f]*\).*/\1/p' | head -1")"
  # GPU names as llama.cpp sees them, with its Vulkan numbering (free memory left out: it changes)
  echo "vulkan_devices=$(v sh -c "docker exec llama-swap llama-server --list-devices 2>&1 | grep -E '^ *Vulkan[0-9]+:' | sed -E 's/^ *//; s/, [0-9]+ MiB free//' | paste -sd ';' | sed 's/;/; /g'")"
  echo "llama_swap_version=$(v sh -c "docker exec llama-swap llama-swap --version 2>&1 | awk '{print \$2}' | head -1")"
else
  echo "image=unknown (llama-swap not running)"
fi
