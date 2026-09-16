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
  let followUpSent = false;

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
  pi.on("session_before_compact", async (event, ctx) => {
    await record("session_before_compact", ctx.cwd, { reason: event.reason });
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

  pi.registerCommand("m0-probe-follow-up", {
    description: "Queue one labelled M0 follow-up after the current agent run.",
    handler: async (_args, ctx) => {
      if (followUpSent) return;
      followUpSent = true;
      await record("follow_up_queued", ctx.cwd);
      pi.sendMessage(
        {
          customType: "agentic-pbt-m0-follow-up",
          content: "M0 probe follow-up: reply exactly M0_PI_FOLLOW_UP_OK.",
          display: true,
        },
        { deliverAs: "followUp", triggerTurn: true },
      );
    },
  });

  pi.registerTool({
    name: "m0_probe_wait",
    label: "M0 Probe Wait",
    description: "Waits for ten seconds so the M0 probe can observe cancellation.",
    parameters: Type.Object({}),
    async execute(_id, _params, signal, _update, ctx) {
      const result = await pi.exec(process.execPath, ["-e", "setTimeout(() => {}, 10000)"], { signal });
      await record("cancellation_probe", ctx.cwd, { code: result.code, killed: result.killed, aborted: Boolean(signal?.aborted) });
      return { content: [{ type: "text", text: result.killed ? "M0_WAIT_CANCELLED" : "M0_WAIT_COMPLETED" }], details: {} };
    },
  });
}
