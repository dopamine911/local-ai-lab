# omp (oh-my-pi) with the lab's models, on any machine in the tailnet

omp talks to zaphod's llama-swap over Tailscale. zaphod forwards `marvin/*` to marvin, so `marvin/big` works
from anywhere zaphod is reachable. Tested 2026-11-03 with omp 18.15.0.

## Set up a machine

1. The machine is in the tailnet: `tailscale status` lists zaphod.
2. omp is installed: `omp --version`.
3. Copy [models.yml](models.yml) to `~/.omp/agent/models.yml` on that machine.
4. Set the key in the shell that starts omp (Proton Pass, or `zaphod/llama-swap/.env` on zaphod):
   `export ZAPHOD_KEY=...`. The file only names the variable; the key is never in it.
5. Test: `omp -p --no-session --model lab/marvin/big "Reply with exactly one word: ready"`.
   Good result: `ready`. `401 unauthorized` means ZAPHOD_KEY is not set in that shell.
6. Use: `omp --model lab/marvin/big` (or `lab/coder`).

## Notes

- `marvin/big`: 128K context, ~40 tok/s writing. Two people/agents can use it at once (2 slots).
- `coder` and `coder-tb` share card 1, so asking for one unloads the other.
- Benchmarks running on zaphod (`lab tbench`) are slowed down by other use of the same model.
