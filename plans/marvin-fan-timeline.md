# marvin fan control (ec_su_axb35): timeline

What was done and when. Times are Chicago time (CDT). Plan: `plans/strix-strata.md` "Later: fan control on marvin";
script: `marvin/fan/ec-module.sh`. Sources: commits, command output, marvin logs.

## 2026-10-30

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 20:20 | Daniel: apply option 3 (ec_su_axb35 module), reboot needed; he tells the port agent | — | — |
| 20:25 | Module source read (cmetz/ec-su_axb35-linux at f62c2c2, latest, 2026-08-20) | init only reads the EC; the update job writes only in CURVE mode; unload writes AUTO to 3 fan registers; no board check (our script adds one); upstream script 02's enrollment check always passes (`if !mokutil` bug) | — |
| 20:27 | marvin read-only check | board AXB35-02, BIOS 1.07, kernel 7.0.0-42, Secure Boot on, lockdown integrity, headers present, mokutil present, dkms not installed, no MOK key yet | — |
| 20:29 | Compile test against kernel 7.0.0-42 (zaphod has the same kernel) in a throwaway Ubuntu 26.04 container | builds, no warnings from the module's code (only "compiler differs from the kernel's", the container's gcc) | — |
| 20:30 | `marvin/fan/ec-module.sh` (check / prepare / install / remove; board check; pinned SHA; dkms signs; refuses unsigned or unenrolled) and `marvin/fan/sample.sh` (read-only CSV every 5 s) written; copied to marvin `~/homelab-fan/` | `check` reads the board/Secure Boot state; `prepare` refuses without sudo; sampler writes rows (fan columns empty until the module loads) | this commit |
| 20:36 | Dry-run of `install`'s dkms path in a container: without `shim-signed`, dkms neither makes the key nor signs ("update-secureboot-policy not found"); with it (marvin has shim-signed 1.59), `dkms generate_mok` makes `/var/lib/shim-signed/mok/MOK.*` and dkms signs with kmodsign: signer = the MOK cert | `prepare` now installs shim-signed too; script re-copied to marvin `~/homelab-fan/` | this commit |
| evening | Daniel: `prepare` (password), reboot, enrolled the key at the blue screen | worked: marvin's enrolled keys include `CN=marvin Secure Boot Module Signature key` | `mokutil --list-enrolled` |
| evening | Daniel: `install` → "STOP: the MOK is not enrolled"; `prepare` again (mokutil: "SKIP: already enrolled"), reboot: no blue screen (nothing to enroll) | **script bug**: `mokutil --test-key` prints "is already enrolled" but exits non-zero; with pipefail the check read that as "not enrolled". The container dry-run never had an enrolled key, so it couldn't catch this | — |

## 2026-10-31

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 15:20 | marvin up after Daniel's second reboot | — | `uptime -s` |
| 15:2x | Fix: one `enrolled()` check reads mokutil's text, not its exit code (3 places); copied to marvin | `check`: MOK enrolled yes, module not loaded | this commit |
| 15:23 | Daniel: `install` | built, **signed by `marvin Secure Boot Module Signature key`**, loaded; power mode **performance**; EC 42 C vs GPU 39 C; fans auto 1,613 / 1,893 / 822 rpm | marvin/SETUP.md |
| 15:24 | llama-swap back (`marvin/big`), Beszel agent up; baseline sampler started, 10 h, every 5 s | `~/fan-samples/2026-10-31-baseline.csv` on marvin | — |
| (done) | Read the sampler after load (below, 2026-11-01) | — | — |
| (done) | Daniel: tells the port agent; `prepare` (password), reboot at the screen, MOK enroll, `install`, `check`; restart llama-swap | — | — |

## 2026-11-01 (CST from today)

| Time | What | Result | Record |
| --- | --- | --- | --- |
| 19:08 | Port agent starts a full-power llama.cpp benchmark; second sampler started, 4 h | `~/fan-samples/2026-11-01-full-power.csv` on marvin (until ~23:08) | — |
| 19:15 | Baseline read (10 h, 7,139 samples, 2.3 h busy at >=90%, the agent's tests) | **clock flat with temperature**: busy median 2,850-2,900 MHz in every 5 C band up to 90-94 C (min 2,659); power mode performance throughout; busy under 100 W 0.4%; max GPU 92 C, EC 87 C; fans max 3,129 / 3,281 / 2,481 rpm | table below |
| 19:15 | Fan level under load | ~~fans at full speed (`level 5`)~~ **wrong, corrected 19:25**: in auto mode `level` is not live (reads 5 with fans stopped at EC 45 C) | `ec-module.sh check` |
| 19:25 | Factory curve read (`rampup_curve`, EC C for levels 1-5): fan1/2 **60, 70, 83, 95, 97**; fan3 20, 60, 83, 95, 97 (rampdown 40, 50, 80, 94, 96). EC max last night 87 C | **fans never above level 3 (~60%)**: max rpm by EC temp ~2,650 below 70 C, ~3,100 above (level 2 step); 80% only at 95 C, 100% at 97 C. **Headroom exists**: a steeper curve can lower temperature | sysfs |
| — | Decision (corrected): no throttling seen, but Daniel wants lower temperature (92 C GPU, margin to 100 C Tjmax). Next: plan step 3, one curve change, measured with the same load. Not while the port agent benchmarks (it would change their conditions); Daniel decides when | — | — |

Baseline, busy samples (>=90%) by GPU temperature:

| GPU temp | samples | clock MHz median (min) | watts median (min) | fan1 / fan2 / fan3 rpm median |
| --- | --- | --- | --- | --- |
| 60-64 C | 13 | 2,859 (2,735) | 105 (98) | 1,765 / 2,032 / 885 |
| 70-74 C | 110 | 2,885 (2,659) | 114 (100) | 1,998 / 2,256 / 963 |
| 75-79 C | 256 | 2,898 (2,679) | 114 (95) | 2,171 / 2,390 / 998 |
| 80-84 C | 967 | 2,898 (2,670) | 118 (103) | 2,645 / 2,852 / 2,097 |
| 85-89 C | 264 | 2,882 (2,682) | 123 (109) | 2,678 / 2,867 / 2,406 |
| 90-94 C | 17 | 2,874 (2,829) | 134 (118) | 2,767 / 2,970 / 2,346 |
| 19:40 | Module source read (f62c2c2): the 60/70/83/95/97 curve is the **module's default for its own curve mode** (believed to copy the factory curve), not read from the chip; in auto the chip's curve is not readable. Curve mode = the module steps each fan one level per second from the EC temperature; unloading the module returns fans to auto | "fans never above ~60%" is an inference from rpm (jump at EC 70 C, ~3,100 rpm max), not a direct reading; `maxrpm` measures the real top speed | src/ec_su_axb35.c |
| 19:45 | `marvin/fan/curve.sh` (check / maxrpm / apply / auto / boot on\|off); test curve up 50,58,65,72,78 / down 45,53,60,67,74 (full speed from EC 78 C; Daniel: noise doesn't matter) | dry run on a fake folder: apply, auto, restore pass; **found and fixed**: maxrpm left fans at 100% (saved modes were a local, gone when the restore ran); Ctrl-C restores too; refuses on zaphod's board; `check` on marvin (read-only) OK | this commit |
| 2026-11-02 07:10 | Read the 4 h recording of the agent's Terminal-Bench re-run (19:08-23:08; run ended ~00:25, 18/19) | busy 2.4 h; max GPU **89 C** (0 samples at 90 C+), EC 87 C; busy clock median 2,876-2,898 MHz in every band, power mode performance, no busy sample under 100 W. Fans max **3,138 / 3,291 / 2,492 rpm**, the same ceiling as 2026-10-31 (3,129 / 3,281 / 2,481): either the auto curve's top step or the fans' real 100%; `maxrpm` decides | `~/fan-samples/2026-11-01-full-power.csv` on marvin |
| 2026-11-02 ~07:40 | Daniel: `sudo curve.sh maxrpm` (step 2) | fans 1/2 at 100%: **~4,750 / ~4,700 rpm**; fan3 (system fan) ~2,300. Under load in auto they peaked at 3,138 / 3,291 (**~66-70% of top speed**); fan3 reached ~2,490, so it is already at full under load. Headroom is on the two CPU fans. Restore to auto after 30 s checked | `curve.sh check` |
| 09:26-09:49 | **A, factory auto**: 4 x `tuning/speed marvin/big` back to back, from cold (EC 34 C) | see table | MLflow "fan test A factory auto run 1-4"; `~/fan-samples/2026-11-02-curve-test.csv` |
| 09:52 | Cooled to EC 40 / GPU 39 C (close to A's start) | — | — |
| 09:53 | Daniel: `sudo curve.sh apply` | all 3 fans mode curve, level 0 at EC 38 C; under load they stepped 0 -> 3 -> 4 within 40 s | — |
| 09:54-10:17 | **B, curve** (up 50,58,65,72,78): same 4 runs | see table | MLflow "fan test B curve 78C run 1-4" |
| — | Decision: **keep** | GPU **9-10 C cooler**, clock +2%, reading +1%, same watts | — |
| ~10:25 | Daniel: `sudo curve.sh boot on` | unit marvin-fan-curve enabled, `active`: the curve is re-applied at every boot | setup/REBUILD.md marvin step 5 |
| ~10:20 | Daniel: test curve **C** `sudo env CURVE_UP=40,45,50,55,60 CURVE_DOWN=35,40,45,50,55 curve.sh apply` (fans 100% whenever there is load: the limit any curve can reach) | fans run at idle too (level 2, ~2,600 rpm) | — |
| 10:21-10:44 | **C**: same 4 runs from EC 40 / GPU 38 C | GPU median **70** (B 73), max **74** (B 78); clock 2,871 (B 2,859, +0.4%); reading 319.9, 314.3, 314.5, 314.2 (B 319.7, 313.5, 313.4, 313.4: +0.3%, inside run-to-run spread); writing 44.6-44.8 (same); fans at 100%: 4,749 / 4,677 / 2,447 rpm | MLflow "fan test C curve 60C run 1-4"; `~/fan-samples/2026-11-02-curve-test-C.csv` |
| — | Result: the limit buys **3-4 C more, no measurable speed** | — | — |
| ~10:50 | Daniel: **keep B**; `sudo curve.sh apply` back to B | fans level 0, spinning down at EC 37 C; boot unit still applies B | — |

Result, busy samples (>=90%), 265 each:

| | GPU C median / 90th pct / max | EC max | clock MHz median (min) | watts | fan1 / fan2 / fan3 rpm median |
| --- | --- | --- | --- | --- | --- |
| A factory auto | 82 / 85 / 88 | 87 | 2,808 (2,708) | 120 | 3,075 / 3,247 / 1,561 |
| B curve | **73 / 76 / 78** | 76 | 2,859 (2,718) | 119 | 4,138 / 4,083 / 2,272 |

Speed (4 runs each): reading 32K A 315.5, 310.0, 310.3, 310.5 tok/s -> B 319.7, 313.5, 313.4, 313.4 (median +1%,
runs 2-4 consistently ~3 tok/s higher: small, possibly heat); writing A 45.1, 44.7, 45.6, 45.3 -> B 44.7, 46.2, 44.7,
44.7 (no change). Fans 1/2 stayed at level 4 (80%): EC never reached 78 C, so level 5 was not needed; a steeper curve
has a little more room, not tested.

### Curve test procedure (one change: factory auto -> curve)
1. Port agent done; nothing else running on marvin. Daniel pushes; on marvin `cd ~/homelab && git pull`.
2. On marvin: `sudo ~/homelab/marvin/fan/curve.sh maxrpm`. Good result: rpm well above ~3,100 (the headroom).
3. On marvin: `~/homelab/marvin/fan/sample.sh 60 ~/fan-samples/2026-11-0x-curve-test.csv &`
4. **A, factory:** on zaphod `cd ~/homelab/tuning; for i in 1 2 3 4; do ./speed marvin/big "fan test A factory auto run $i"; done`
5. Cool down until `curve.sh check` shows EC under 55 C.
6. On marvin: `sudo ~/homelab/marvin/fan/curve.sh apply`
7. **B, curve:** the same loop as 4 with "fan test B curve 78C run $i".
8. Compare from the CSV (same busy time in A and B): GPU temp, clock, watts, rpm; and the speed results.
   Keep if GPU temperature is lower with the clock the same or higher: `sudo curve.sh boot on` (setup/REBUILD.md).
   Otherwise `sudo curve.sh auto`.
