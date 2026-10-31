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
| next | Read the sampler after load: does the clock fall as temperature passes ~95 C, and do the fans reach full speed? Then decide on a curve (plan step 3) | — | — |
| (done) | Daniel: tells the port agent; `prepare` (password), reboot at the screen, MOK enroll, `install`, `check`; restart llama-swap | — | — |
