# Pi PBT lifecycle adapter

Load the adapter with:

```sh
pi -e /absolute/path/to/agentic-pbt/integrations/pi/index.ts
```

It is inert unless `.pbt/state/active.json` exists in Pi's working directory.
For a Pi `PATH` that cannot find `pbt`, set `PBT_BIN` to an absolute executable
path. The adapter injects the active task's PBT policy and maps `pbt gate --json`
follow-ups from `pbt gate --task <id> --json` to Pi's settlement boundary; it adds no Pi tools or slash commands.
