import { appendFile } from "node:fs/promises";
import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Type } from "typebox";

const logPath = process.env.M0_PROBE_LOG;
if (!logPath) throw new Error("M0_PROBE_LOG must be an absolute event-log path");

async function record(event: string, cwd: string, details: Record<string, unknown> = {}) {
  await appendFile(
    logPath!,
    `${JSON.stringify({
      timestamp: new Date().toISOString(),
      runtime: "pi",
      event,
      cwd,
      ...details,
    })}\n`,
  );
}

export default function (pi: ExtensionAPI) {
  pi.on("session_start", async (event, ctx) => {
    await record("session_start", ctx.cwd, { reason: event.reason });
  });
  pi.on("before_agent_start", async (_event, ctx) => {
    await record("before_agent_start", ctx.cwd);
    return {
      message: {
        customType: "agentic-pbt-m0-policy",
        content: "M0 probe: policy injection reached this Pi turn.",
        display: true,
      },
    };
  });
  pi.on("tool_execution_end", async (event, ctx) => {
    await record("tool_execution_end", ctx.cwd, { tool: event.toolName, isError: event.isError });
  });
  pi.on("agent_end", async (_event, ctx) => {
    await record("agent_end", ctx.cwd);
  });
  pi.on("agent_settled", async (_event, ctx) => {
    await record("agent_settled", ctx.cwd, { idle: ctx.isIdle() });
  });
  pi.on("session_compact", async (event, ctx) => {
    await record("session_compact", ctx.cwd, { reason: event.reason });
  });

  pi.registerTool({
    name: "m0_probe_exec",
    label: "M0 Probe Exec",
    description: "Runs a harmless child process for the M0 Pi integration probe.",
    parameters: Type.Object({}),
    async execute(_id, _params, signal, _update, ctx) {
      const result = await pi.exec(process.execPath, ["-e", "process.stdout.write('M0_PROCESS_OK')"], { signal });
      await record("process_probe", ctx.cwd, { code: result.code, killed: result.killed, stdout: result.stdout });
      return { content: [{ type: "text", text: result.stdout }], details: {} };
    },
  });
}
