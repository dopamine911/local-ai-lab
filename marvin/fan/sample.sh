#!/usr/bin/env bash
# Read-only sampler for marvin's cooling (plans/strix-strata.md "Later: fan control", step 2: measure before any fan
# change). Every 5 s, one CSV line: time, GPU busy %, GPU watts, GPU clock MHz, GPU temp C, and, when the
# ec_su_axb35 module is loaded, EC temp C, fan1-3 rpm and mode, BIOS power mode. No sudo; writes nothing to hardware.
#
#   ./sample.sh [minutes] [out.csv]      default: 60 minutes, ~/fan-samples/<date-time>.csv
#   ./sample.sh 600 &                    e.g. through a night of benchmark runs
#
# Good result: busy rows at 130-140 W and ~2,850 MHz. A clock that falls while busy as the temperature climbs past
# ~95 C means heat is costing speed (then a fan curve is worth testing); a flat clock means it isn't.
set -u
MIN=${1:-60}; OUT=${2:-$HOME/fan-samples/$(date +%Y%m%d-%H%M).csv}
mkdir -p "$(dirname "$OUT")"
D=/sys/class/drm/card1/device; H=$(echo $D/hwmon/hwmon*); E=/sys/class/ec_su_axb35
r() { cat "$1" 2>/dev/null || echo ""; }
echo "time,busy_pct,gpu_w,sclk_mhz,gpu_temp_c,ec_temp_c,fan1_rpm,fan2_rpm,fan3_rpm,fan1_mode,fan2_mode,fan3_mode,power_mode" > "$OUT"
end=$(( $(date +%s) + MIN * 60 ))
while [ "$(date +%s)" -lt "$end" ]; do
  sclk=$(grep '\*' $D/pp_dpm_sclk | grep -oE '[0-9]+Mhz' | tr -d 'Mhz')
  echo "$(date +%H:%M:%S),$(r $D/gpu_busy_percent),$(( $(r $H/power1_average) / 1000000 )),$sclk,$(( $(r $H/temp1_input) / 1000 )),$(r $E/temp1/temp),$(r $E/fan1/rpm),$(r $E/fan2/rpm),$(r $E/fan3/rpm),$(r $E/fan1/mode),$(r $E/fan2/mode),$(r $E/fan3/mode),$(r $E/apu/power_mode)" >> "$OUT"
  sleep 5
done
echo "wrote $OUT ($(( $(wc -l < "$OUT") - 1 )) samples)"
