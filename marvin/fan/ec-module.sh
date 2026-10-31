#!/usr/bin/env bash
# marvin fan control, step 1: the ec_su_axb35 kernel module, READ-ONLY (plans/strix-strata.md "Later: fan control").
# It lets Linux read marvin's fan chip (EC): fan speeds and modes, the EC temperature, and the BIOS power mode
# (performance / balanced / quiet: the "55 W" check becomes a direct read). This script never writes a fan setting.
# Runs ON marvin (Bosgame M5, board AXB35-02). Refuses on any other board: unloading the module writes 3 fan registers.
#
#   ./ec-module.sh check            no sudo. What is installed and what the module reads. Run any time.
#   sudo ./ec-module.sh prepare     INTERACTIVE, in Daniel's own ssh session: installs dkms, makes the module-signing
#                                   key (MOK) and asks to enroll it. mokutil asks for a one-time password (max 5
#                                   characters, keep it in your head only). Then reboot with marvin's screen and
#                                   keyboard attached: see "At the blue MOK Manager screen" below.
#   sudo ./ec-module.sh install     after the reboot: build, sign (dkms, with the enrolled key), load, load at boot.
#   sudo ./ec-module.sh remove      unload (fans back to AUTO), uninstall, stop loading at boot.
#
# At the blue MOK Manager screen (after `prepare`, on the next boot only):
#   1. press a key  2. Enroll MOK  3. Continue  4. Yes  5. type the password  6. Reboot
#
# Pinned: module source cmetz/ec-su_axb35-linux at f62c2c228959a08683273a26ef3afd8991e69f6d (latest, 2026-08-20),
# compile-tested against kernel 7.0.0-42 on zaphod before the first reboot (plans/marvin-fan-timeline.md).
set -euo pipefail
REPO=https://github.com/cmetz/ec-su_axb35-linux.git
SHA=f62c2c228959a08683273a26ef3afd8991e69f6d
NAME=ec_su_axb35; VER=1.0
MOK_KEY=/var/lib/shim-signed/mok/MOK.priv; MOK_CERT=/var/lib/shim-signed/mok/MOK.der
SYS=/sys/class/ec_su_axb35
U=${SUDO_USER:-$USER}; SRC=/home/$U/src/ec-su_axb35-linux

board_check() {
  local b; b=$(cat /sys/class/dmi/id/board_name 2>/dev/null || echo unknown)
  [ "$b" = AXB35-02 ] || { echo "STOP: board is '$b', not AXB35-02 (Bosgame M5). This module writes EC registers on unload."; exit 1; }
}
root() { [ "$(id -u)" = 0 ] || { echo "run with sudo"; exit 1; }; }
say() { printf '%-34s %s\n' "$1" "$2"; }

check() {
  board_check
  say "board" "$(cat /sys/class/dmi/id/board_name), BIOS $(cat /sys/class/dmi/id/bios_version), kernel $(uname -r)"
  say "secure boot / lockdown" "$(mokutil --sb-state 2>/dev/null | head -1) / $(grep -o '\[[a-z]*\]' /sys/kernel/security/lockdown)"
  say "dkms" "$(dpkg -s dkms 2>/dev/null | awk '/^Version/{print $2}' || true)$(dpkg -s dkms >/dev/null 2>&1 || echo 'not installed')"
  say "signing key (MOK)" "$([ -e $MOK_CERT ] && echo present || echo 'not made yet (prepare)')"
  if [ -e $MOK_CERT ]; then
    say "MOK enrolled" "$(mokutil --test-key $MOK_CERT 2>&1 | grep -q 'is already enrolled' && echo yes || echo 'NO (reboot and enroll at the blue screen)')"
  fi
  if [ ! -d $SYS ]; then say "module" "not loaded"; return 0; fi
  say "module" "loaded, signer: $(modinfo -F signer $NAME 2>/dev/null || echo unknown)"
  local pm; pm=$(cat $SYS/apu/power_mode)
  say "BIOS power mode" "$pm$([ "$pm" = performance ] || echo '   <-- NOT performance: GPU will be held at ~55 W')"
  say "EC temperature" "$(cat $SYS/temp1/temp 2>/dev/null || cat $SYS/temp1/temp_cur 2>/dev/null) C (GPU hwmon: $(( $(cat /sys/class/drm/card1/device/hwmon/hwmon*/temp1_input) / 1000 )) C)"
  for f in $SYS/fan*; do
    say "$(basename $f)" "$(cat $f/rpm 2>/dev/null) rpm, mode $(cat $f/mode), level $(cat $f/level 2>/dev/null)"
  done
  echo "Good result: power mode performance; every fan mode auto (nothing written); rpm non-zero under load."
}

prepare() {
  root; board_check
  [ -t 0 ] || { echo "prepare is interactive (mokutil asks for a password): run it in your own ssh session"; exit 1; }
  apt-get update -q && apt-get install -y -q dkms openssl build-essential "linux-headers-$(uname -r)"
  dpkg -s dkms build-essential | awk '/^(Package|Version)/{printf "%s ", $2} END{print ""}'
  if [ ! -e $MOK_CERT ]; then dkms generate_mok; fi
  [ -e $MOK_KEY ] && [ -e $MOK_CERT ] || { echo "MOK key missing after dkms generate_mok"; exit 1; }
  if mokutil --test-key $MOK_CERT 2>&1 | grep -q 'is already enrolled'; then
    echo "MOK already enrolled: no reboot needed, run: sudo $0 install"; exit 0
  fi
  echo; echo "mokutil now asks for a one-time password (5 characters or fewer). You type it again at the blue screen."
  mokutil --import $MOK_CERT
  echo; echo "Next: reboot marvin WITH its screen and keyboard attached:  sudo systemctl reboot"
  echo "At the blue MOK Manager screen: press a key, Enroll MOK, Continue, Yes, the password, Reboot."
  echo "Then: sudo $0 install"
}

install_mod() {
  root; board_check
  mokutil --test-key $MOK_CERT 2>&1 | grep -q 'is already enrolled' \
    || { echo "STOP: the MOK is not enrolled. Run prepare, reboot, and enroll at the blue screen first."; exit 1; }
  if [ ! -d "$SRC/.git" ]; then sudo -u "$U" git clone -q "$REPO" "$SRC"; fi
  sudo -u "$U" git -C "$SRC" fetch -q origin && sudo -u "$U" git -C "$SRC" checkout -q "$SHA"
  [ "$(git -C "$SRC" rev-parse HEAD)" = "$SHA" ] || { echo "STOP: source is not at $SHA"; exit 1; }
  dkms status "$NAME" 2>/dev/null | grep -q "$NAME" && dkms remove "$NAME/$VER" --all || true
  rm -rf "/usr/src/$NAME-$VER"; mkdir -p "/usr/src/$NAME-$VER"
  git -C "$SRC" archive "$SHA" | tar -x -C "/usr/src/$NAME-$VER"
  # dkms on Ubuntu signs built modules with the MOK key ($MOK_KEY) by itself; checked below with modinfo.
  printf 'PACKAGE_NAME="%s"\nPACKAGE_VERSION="%s"\nBUILT_MODULE_NAME[0]="%s"\nDEST_MODULE_LOCATION[0]="/updates"\nAUTOINSTALL="yes"\n' \
    "$NAME" "$VER" "$NAME" > "/usr/src/$NAME-$VER/dkms.conf"
  dkms add "$NAME/$VER" && dkms build "$NAME/$VER" && dkms install "$NAME/$VER"
  local signer; signer=$(modinfo -F signer "$NAME" 2>/dev/null || true)
  [ -n "$signer" ] || { echo "STOP: the built module is not signed (lockdown would refuse it). See /etc/dkms/framework.conf mok_signing_key."; exit 1; }
  echo "signed by: $signer"
  modprobe "$NAME" || { echo "STOP: modprobe refused (key rejected? check: dmesg | tail)"; exit 1; }
  echo "$NAME" > /etc/modules-load.d/$NAME.conf   # load at every boot
  dmesg | grep -q "Sixunited AXB35-02 EC driver loaded" && echo "module loaded" || echo "loaded, but no driver message in dmesg"
  echo; check
}

remove_mod() {
  root; board_check
  modprobe -r "$NAME" 2>/dev/null && echo "unloaded (fans set to AUTO by the module)" || true
  dkms remove "$NAME/$VER" --all 2>/dev/null || true
  rm -rf "/usr/src/$NAME-$VER" /etc/modules-load.d/$NAME.conf
  echo "removed (the MOK key stays enrolled; harmless)"
}

case "${1:-}" in
  check) check ;;
  prepare) prepare ;;
  install) install_mod ;;
  remove) remove_mod ;;
  *) sed -n 2,22p "$0"; exit 1 ;;
esac
