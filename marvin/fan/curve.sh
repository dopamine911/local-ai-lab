#!/usr/bin/env bash
# marvin fan curve (plans/strix-strata.md "Later: fan control", step 3; timeline plans/marvin-fan-timeline.md).
# Needs the ec_su_axb35 module (ec-module.sh install). Runs ON marvin; refuses on any other board.
#
#   ./curve.sh check            no sudo. Mode, level, rpm and curve of each fan; EC temperature; power mode.
#   sudo ./curve.sh maxrpm      all fans at 100% for 30 s, print rpm, then back to how they were. Measures headroom.
#   sudo ./curve.sh apply       the curve below on all 3 fans (module "curve" mode). Not kept across a reboot.
#   sudo ./curve.sh auto        back to the fan chip's own control (factory), and the module's default curve values.
#   sudo ./curve.sh boot on|off apply at every boot (systemd unit marvin-fan-curve), or stop doing that (and go auto).
#
# How "curve" mode works (module source, f62c2c2): every second the module reads the EC temperature and moves each fan
# one level (0-5 = 0-100% in 20% steps) up when temp >= UP[next level], down when temp <= DOWN[current level].
# Removing the module puts the fans back to auto. In "auto" the fan chip runs its own curve, which Linux cannot read;
# the module's default curve (60,70,83,95,97) is believed to copy it.
#
# The curve under test (Daniel 2026-11-01: lower temperature wanted, noise doesn't matter). EC temperature, C:
UP=50,58,65,72,78      # reach level 1..5 at these (factory-like default: 60,70,83,95,97)
DOWN=45,53,60,67,74    # drop from level 1..5 at these (5 C below each UP, so it doesn't flap)
# Kept 2026-11-02 (GPU 82 -> 73 C median vs factory). To TRY another curve without changing this file:
#   sudo env CURVE_UP=40,45,50,55,60 CURVE_DOWN=35,40,45,50,55 ./curve.sh apply      (until reboot or `apply`)
UP=${CURVE_UP:-$UP}; DOWN=${CURVE_DOWN:-$DOWN}
#
# Good result of a test: same load, lower GPU temperature, clock the same or higher (tuning/speed + sample.sh).
set -euo pipefail
E=${EC_SYSFS:-/sys/class/ec_su_axb35}   # EC_SYSFS: a fake folder, for the dry run only (no board/root check then)
HERE=$(cd "$(dirname "$0")" && pwd)
UNIT=/etc/systemd/system/marvin-fan-curve.service

board_check() {
  [ -n "${EC_SYSFS:-}" ] && return 0
  local b; b=$(cat /sys/class/dmi/id/board_name 2>/dev/null || echo unknown)
  [ "$b" = AXB35-02 ] || { echo "STOP: board is '$b', not AXB35-02 (Bosgame M5)."; exit 1; }
  [ -d $E ] || { echo "STOP: module ec_su_axb35 not loaded (marvin/fan/ec-module.sh)"; exit 1; }
}
root() { [ -n "${EC_SYSFS:-}" ] || [ "$(id -u)" = 0 ] || { echo "run with sudo"; exit 1; }; }
gpu_temp() { echo $(( $(cat /sys/class/drm/card1/device/hwmon/hwmon*/temp1_input) / 1000 )); }

check() {
  board_check
  echo "EC temp $(cat $E/temp1/temp) C (max since module load $(cat $E/temp1/max) C), GPU $(gpu_temp) C, power mode $(cat $E/apu/power_mode)"
  for f in 1 2 3; do
    printf 'fan%s  mode %-5s  level %s  %5s rpm   up %-14s down %s\n' $f "$(cat $E/fan$f/mode)" "$(cat $E/fan$f/level)" \
      "$(cat $E/fan$f/rpm)" "$(cat $E/fan$f/rampup_curve)" "$(cat $E/fan$f/rampdown_curve)"
  done
  local u; u=$(systemctl is-enabled marvin-fan-curve 2>/dev/null || true)
  echo "boot unit: ${u/not-found/not installed}"
  echo "Note: in auto mode 'level' is not live (the fan chip decides); in curve/fixed mode it is."
}

maxrpm() {
  root; board_check
  # global, not local: the EXIT trap runs after this function has returned (a local was gone: dry run 2026-11-01)
  BEFORE=(); for f in 1 2 3; do BEFORE+=("$(cat $E/fan$f/mode)"); done
  trap 'for f in 1 2 3; do echo "${BEFORE[$((f-1))]}" > $E/fan$f/mode; done; echo "fans back to: ${BEFORE[*]}"' EXIT
  for f in 1 2 3; do echo fixed > $E/fan$f/mode; echo 5 > $E/fan$f/level; done
  echo "all fans at level 5 (100%) for 30 s ..."
  for i in 1 2 3 4 5 6; do sleep 5; echo "  ${i}x5 s: $(cat $E/fan1/rpm) / $(cat $E/fan2/rpm) / $(cat $E/fan3/rpm) rpm"; done
}

apply() {
  root; board_check
  for f in 1 2 3; do
    echo "$UP" > $E/fan$f/rampup_curve
    echo "$DOWN" > $E/fan$f/rampdown_curve
    echo curve > $E/fan$f/mode
    [ "$(cat $E/fan$f/rampup_curve)" = "$UP" ] && [ "$(cat $E/fan$f/mode)" = curve ] \
      || { echo "STOP: fan$f did not take the curve; going back to auto"; auto; exit 1; }
  done
  echo "curve applied: up $UP / down $DOWN (EC C)"; check
}

auto() {
  root; board_check
  for f in 1 2 3; do echo auto > $E/fan$f/mode; done
  echo 60,70,83,95,97 > $E/fan1/rampup_curve; echo 40,50,80,94,96 > $E/fan1/rampdown_curve
  echo 60,70,83,95,97 > $E/fan2/rampup_curve; echo 40,50,80,94,96 > $E/fan2/rampdown_curve
  echo 20,60,83,95,97 > $E/fan3/rampup_curve; echo 0,50,80,94,96 > $E/fan3/rampdown_curve
  echo "fans back to the fan chip's own control (auto)"
}

boot() {
  root; board_check
  case "${1:-}" in
    on)
      cat > $UNIT <<EOF
# Written by $HERE/curve.sh boot on. Re-applies marvin's fan curve after each boot (the fan chip forgets it).
[Unit]
Description=marvin fan curve (ec_su_axb35 curve mode)
After=systemd-modules-load.service
ConditionPathExists=$E

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=$HERE/curve.sh apply
ExecStop=$HERE/curve.sh auto

[Install]
WantedBy=multi-user.target
EOF
      systemctl daemon-reload; systemctl enable --now marvin-fan-curve; systemctl is-active marvin-fan-curve ;;
    off)
      systemctl disable --now marvin-fan-curve 2>/dev/null || true; rm -f $UNIT; systemctl daemon-reload; auto ;;
    *) echo "boot on|off"; exit 1 ;;
  esac
}

case "${1:-}" in
  check) check ;;
  maxrpm) maxrpm ;;
  apply) apply ;;
  auto) auto ;;
  boot) shift; boot "${1:-}" ;;
  *) sed -n 2,23p "$0"; exit 1 ;;
esac
