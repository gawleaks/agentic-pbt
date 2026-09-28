import { isAbsolute } from "node:path";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

type Result = {
  schemaVersion: number;
  task?: unknown;
  workflowState?: unknown;
  reasonCode?: unknown;
  followUp?: unknown;
};

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
    const schemaVersion = (result as Record<PropertyKey, unknown>).schemaVersion;
    if (schemaVersion !== 1) return "pbt returned unsupported schemaVersion";
    return result as Result;
  } catch {
    return "pbt returned invalid JSON";
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
    if (result.stdout.trim() !== "") {
      const parsed = parseResult(result.stdout);
      if (isError(parsed)) return parsed;
      if (result.code !== 0 && (!parseNonzero || (result.code !== 1 && result.code !== 2))) {
        if (result.stderr.trim() !== "") return result.stderr.trim();
        if (parsed.reasonCode === "setup_error" && args[2]) {
          return `run pbt inspect --task ${args[2]} --json to diagnose PBT setup`;
        }
        return `pbt ${args[0]} failed`;
      }
      return parsed;
    }
    return result.stderr.trim() || `pbt ${args[0]} failed`;
  } catch (error) {
    return error instanceof Error ? error.message : "pbt could not be started";
  }
}

function isError(result: Result | string): result is string {
  return typeof result === "string";
}

function isNoActiveTask(error: string) {
  return error.includes("no active PBT task") || (error.includes("active.json") && error.includes("no such file or directory"));
}

async function activeTask(pi: ExtensionAPI, signal: AbortSignal | undefined, allowComplete = false): Promise<string | undefined> {
  const pbt = resolvePbt();
  if (pbt.startsWith("PBT_BIN")) throw new Error(pbt);
  const result = await run(pi, pbt, ["status", "--json"], signal);
  if (isError(result)) {
    if (isNoActiveTask(result)) return;
    throw new Error(result);
  }
  if (typeof result.task !== "string" || result.task === "") {
    throw new Error("pbt status returned no task");
  }
  if (result.workflowState === "complete" && allowComplete) return result.task;
  if (result.workflowState === "complete" || result.workflowState === "cancelled" || result.workflowState === "blocked") return;
  if (result.workflowState !== "active") {
    throw new Error("pbt status returned invalid workflowState");
  }
  return result.task;
}

export default function (pi: ExtensionAPI) {
  pi.on("before_agent_start", async (_event, ctx) => {
    let task: string | undefined;
    try {
      task = await activeTask(pi, ctx.signal);
    } catch (error) {
      return { message: setupMessage(error instanceof Error ? error.message : "pbt status failed") };
    }
    if (task === undefined) return;

    return {
      message: {
        customType: "agentic-pbt-policy",
        content: `PBT task ${task} is active. Read skills/property-based-testing/SKILL.md, update its contract, then run pbt inspect --task ${task} --json and pbt verify --task ${task} --json.`,
        display: true,
      },
    };
  });

  pi.on("agent_before_settle", async (event, ctx) => {
    const isCancellation = event.outcome === "aborted" || event.outcome === "error";
    let task: string | undefined;
    try {
      task = await activeTask(pi, isCancellation ? undefined : ctx.signal, event.outcome === "completed");
    } catch (error) {
      if (event.outcome === "completed") {
        return { entries: [{ type: "custom_message", ...setupMessage(error instanceof Error ? error.message : "pbt status failed") }], continue: false };
      }
      return;
    }
    if (task === undefined) return;

    const pbt = resolvePbt();
    if (isCancellation) {
      await run(pi, pbt, ["cancel", "--task", task, "--json"], undefined, true);
      return;
    }
    if (event.outcome !== "completed") return;

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
