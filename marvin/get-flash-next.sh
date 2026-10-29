#!/usr/bin/env bash
# Download Qwen3.8-Flash-Next UD-IQ4_XS + MTP (unsloth, pinned revision) to /srv/models, then check sha256.
# Safe to re-run: finished files are skipped, partial ones resume.
set -uo pipefail
REV=766911a6b7369840a91dbcd95f9f997acaab6cd6
BASE=https://huggingface.co/unsloth/Qwen3.8-Flash-Next-GGUF/resolve/$REV
cd /srv/models
while read -r path size sha; do
  f=$(basename "$path")
  if [ -f "$f" ] && [ "$(stat -c %s "$f")" = "$size" ]; then echo "have $f"
  else echo "$(date +%T) downloading $f ($((size/1000000000)) GB)"; wget -q -c -O "$f" "$BASE/$path" || { echo "FAILED $f"; exit 1; }; fi
  echo "$sha  $f" > "$f.sha256"
done <<'LIST'
UD-IQ4_XS/Qwen3.8-Flash-Next-UD-IQ4_XS-00001-of-00003.gguf 10946624 5ce89370720f8bf90890f439361282104c1aa1482d4013bb9a50923e758e71a4
UD-IQ4_XS/Qwen3.8-Flash-Next-UD-IQ4_XS-00002-of-00003.gguf 49835229856 577a38a2392b40ca2193cea502e1d92f60b8cd370675d308e0ec21885d9daaa7
UD-IQ4_XS/Qwen3.8-Flash-Next-UD-IQ4_XS-00003-of-00003.gguf 43836407744 d4634e6d84f0ebb0940be15c90d3790bf6464e3dea3a1cddc567dc0e83ad8833
mtp-Qwen3.8-Flash-Next-Q8_0.gguf 4137429120 58a47d6a216307e9b97172d5bfdc3436a8994b632f5e436b9bb6edf8951da8d1
LIST
echo "$(date +%T) checking sha256"; cat *.sha256 | sha256sum -c && echo DOWNLOAD-OK || echo CHECKSUM-FAILED
