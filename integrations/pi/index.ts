import { access, readFile } from "node:fs/promises";
import { isAbsolute, join } from "node:path";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

type Result = { task?: unknown; followUp?: unknown };

function setupMessage(error: string) {
  return {
    customType: "agentic-pbt-setup",
    content: `PBT setup error: ${error}`,
    display: true,
  };
}

function parseResult(stdout: string): Result | string {
  try {
    const result: unknown = JSON.parse(stdout);
    if (typeof result !== "object" || result === null || Array.isArray(result)) {
      return "pbt returned invalid JSON";
    }
    return result;
  } catch {
    return "pbt returned invalid JSON";
  }
}

function activeTaskPath(cwd: string) {
  return join(cwd, ".pbt/state/active.json");
}

async function activeTaskExists(cwd: string) {
  try {
    await access(activeTaskPath(cwd));
    return true;
  } catch {
    return false;
  }
}

async function activeTask(cwd: string) {
  if (!(await activeTaskExists(cwd))) return;
  try {
    const state: unknown = JSON.parse(await readFile(activeTaskPath(cwd), "utf8"));
    if (typeof state !== "object" || state === null || Array.isArray(state)) return;
    const task = (state as Record<PropertyKey, unknown>).task;
    if (typeof task === "string") return task;
  } catch {
    return;
  }
}

function resolvePbt() {
  const pbt = process.env.PBT_BIN ?? "pbt";
  if (process.env.PBT_BIN !== undefined && !isAbsolute(pbt)) {
    return "PBT_BIN must be an absolute path";
  }
  return pbt;
}

async function run(pi: ExtensionAPI, command: string, args: string[], signal: AbortSignal | undefined, parseNonzero = false): Promise<Result | string> {
  try {
    const result = await pi.exec(command, args, { signal });
    if (result.code !== 0 && (!parseNonzero || (result.code !== 1 && result.code !== 2))) {
      return result.stderr.trim() || `pbt ${args[0]} failed`;
    }
    if (result.stdout.trim() === "") return result.stderr.trim() || `pbt ${args[0]} failed`;
    return parseResult(result.stdout);
  } catch (error) {
    return error instanceof Error ? error.message : "pbt could not be started";
  }
}

function isError(result: Result | string): result is string {
  return typeof result === "string";
}

export default function (pi: ExtensionAPI) {
  pi.on("before_agent_start", async (_event, ctx) => {
    if (!(await activeTaskExists(ctx.cwd))) return;

    const pbt = resolvePbt();
    if (pbt.startsWith("PBT_BIN")) return { message: setupMessage(pbt) };

    const result = await run(pi, pbt, ["status", "--json"], ctx.signal);
    if (isError(result) || typeof result.task !== "string" || result.task === "") {
      return { message: setupMessage(isError(result) ? result : "pbt status returned no task") };
    }

    return {
      message: {
        customType: "agentic-pbt-policy",
        content: `PBT task ${result.task} is active. Read skills/property-based-testing/SKILL.md, update its contract, then run pbt inspect --task ${result.task} --json and pbt verify --task ${result.task} --json.`,
        display: true,
      },
    };
  });

  pi.on("agent_before_settle", async (event, ctx) => {
    if (event.outcome !== "completed") return;
    const task = await activeTask(ctx.cwd);
    if (!task) return;

    const pbt = resolvePbt();
    if (pbt.startsWith("PBT_BIN")) {
      return { entries: [{ type: "custom_message", ...setupMessage(pbt) }], continue: false };
    }

    const result = await run(pi, pbt, ["gate", "--task", task, "--json"], ctx.signal, true);
    if (isError(result)) {
      return { entries: [{ type: "custom_message", ...setupMessage(result) }], continue: false };
    }
    if (typeof result.followUp !== "string" || result.followUp === "") return;

    return {
      entries: [{ type: "custom_message", customType: "agentic-pbt-repair", content: result.followUp, display: true }],
      continue: true,
    };
  });
}
