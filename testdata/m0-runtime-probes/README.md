# M0 runtime probes

Run probes only in a temporary Git fixture created by `./create-fixture.sh`.
Set `M0_PROBE_LOG` to an absolute JSONL path before launching either runtime.
Each record has `timestamp`, `runtime`, `event`, and `cwd`; logs may contain
prompt text or paths, so keep them outside the repository and delete them when
the compatibility record has captured the result.

A passing script assertion proves that an event reached the probe. It does not
replace the manual observation noted by the matching command below.

## Launch commands and manual observations

### Pi

```bash
fixture="$(testdata/m0-runtime-probes/create-fixture.sh)"
log="$(mktemp "${TMPDIR:-/tmp}/agentic-pbt-m0-pi.XXXXXX")"
testdata/m0-runtime-probes/pi/run.sh "$fixture" "$log"
```

Not yet implemented. Tasks 3-5 of this plan will add:

- Task 3: the manual observation notes for the Pi probe run.
- Task 4: the Claude runtime launch command, analogous to Task 2.
- Task 5: the manual observation notes for the Claude probe run.

This README will be updated with the exact commands and observations once
those tasks land.
