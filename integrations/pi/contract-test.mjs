import assert from "node:assert/strict";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const extension = await import(pathToFileURL(join(process.cwd(), "integrations/pi/index.ts")).href);

async function fixture() {
  return mkdtemp(join(tmpdir(), "agentic-pbt-pi-"));
}

function load(responses) {
  const handlers = new Map();
  const calls = [];
  extension.default({
    on(name, handler) {
      handlers.set(name, handler);
    },
    async exec(command, args, options) {
      calls.push({ command, args, options });
      return responses.shift();
    },
  });
  return { handlers, calls };
}

function json(value, code = 0, stderr = "") {
  return { code, killed: false, stderr, stdout: JSON.stringify(value) };
}

function missing() {
  return { code: 127, killed: false, stderr: "pbt: command not found", stdout: "" };
}

const active = { schemaVersion: 1, task: "repair", workflowState: "active" };
const noActive = { code: 3, killed: false, stderr: "no active PBT task", stdout: "" };
const previousPbtBin = process.env.PBT_BIN;
try {
  delete process.env.PBT_BIN;

  {
    const cwd = await fixture();
    const { handlers, calls } = load([
      json(active),
      json(active),
      json({ schemaVersion: 1, followUp: "Run pbt verify" }, 1),
      json(active),
      json({ schemaVersion: 1, workflowState: "cancelled" }, 2),
      json(active),
      json({ schemaVersion: 1, workflowState: "cancelled" }, 2),
    ]);
    const before = handlers.get("before_agent_start");
    const boundary = handlers.get("agent_before_settle");

    const policy = await before({}, { cwd });
    assert.equal(policy.message.customType, "agentic-pbt-policy");
    assert.match(policy.message.content, /repair/);
    assert.match(policy.message.content, /skills\/property-based-testing\/SKILL\.md/);
    assert.match(policy.message.content, /pbt inspect/);
    assert.match(policy.message.content, /pbt verify/);
    assert.deepEqual(
      await boundary({ outcome: "completed" }, { cwd }),
      {
        entries: [{ type: "custom_message", customType: "agentic-pbt-repair", content: "Run pbt verify", display: true }],
        continue: true,
      },
    );
    const controller = new AbortController();
    controller.abort();
    assert.equal(await boundary({ outcome: "aborted" }, { cwd, signal: controller.signal }), undefined);
    assert.equal(await boundary({ outcome: "error" }, { cwd, signal: controller.signal }), undefined);
    assert.deepEqual(calls.map(({ args }) => args), [
      ["status", "--json"],
      ["status", "--json"], ["gate", "--task", "repair", "--json"],
      ["status", "--json"], ["cancel", "--task", "repair", "--json"],
      ["status", "--json"], ["cancel", "--task", "repair", "--json"],
    ]);
    for (const { options } of calls.filter(({ args }) => args[0] === "cancel" || args[0] === "status").slice(2)) {
      assert.equal(options.signal, undefined);
    }
    await rm(cwd, { recursive: true });
  }

  {
    const cwd = await fixture();
    const { handlers, calls } = load([noActive, noActive]);
    assert.equal(await handlers.get("before_agent_start")({}, { cwd }), undefined);
    assert.equal(await handlers.get("agent_before_settle")({ outcome: "completed" }, { cwd }), undefined);
    assert.deepEqual(calls.map(({ args }) => args), [["status", "--json"], ["status", "--json"]]);
    await rm(cwd, { recursive: true });
  }

  {
    const cwd = await fixture();
    const { handlers } = load([json({ schemaVersion: 1, task: "repair", workflowState: "cancelled" })]);
    assert.equal(await handlers.get("before_agent_start")({}, { cwd }), undefined);
    await rm(cwd, { recursive: true });
  }

  {
    const cwd = await fixture();
    const { handlers, calls } = load([
      json({ schemaVersion: 1, task: "repair", workflowState: "blocked" }),
      json({ schemaVersion: 1, task: "repair", workflowState: "blocked" }),
    ]);
    assert.equal(await handlers.get("before_agent_start")({}, { cwd }), undefined);
    assert.equal(await handlers.get("agent_before_settle")({ outcome: "completed" }, { cwd }), undefined);
    assert.deepEqual(calls.map(({ args }) => args), [["status", "--json"], ["status", "--json"]]);
    await rm(cwd, { recursive: true });
  }

  {
    const cwd = await fixture();
    const { handlers } = load([missing()]);
    const policy = await handlers.get("before_agent_start")({}, { cwd });
    assert.equal(policy.message.customType, "agentic-pbt-setup");
    await rm(cwd, { recursive: true });
  }

  {
    const cwd = await fixture();
    const { handlers } = load([json(active), json({ schemaVersion: 1, reasonCode: "setup_error" }, 3)]);
    const result = await handlers.get("agent_before_settle")({ outcome: "completed" }, { cwd });
    assert.equal(result.entries[0].customType, "agentic-pbt-setup");
    assert.match(result.entries[0].content, /pbt inspect --task repair --json/);
    await rm(cwd, { recursive: true });
  }

  {
    const cwd = await fixture();
    const { handlers } = load([{ code: 3, killed: false, stderr: "open pbt.config.json: no such file or directory", stdout: "" }]);
    const policy = await handlers.get("before_agent_start")({}, { cwd });
    assert.equal(policy.message.customType, "agentic-pbt-setup");
    await rm(cwd, { recursive: true });
  }

  {
    const cwd = await fixture();
    const { handlers } = load([{ code: 3, killed: false, stderr: "invalid active.json", stdout: "" }]);
    const policy = await handlers.get("before_agent_start")({}, { cwd });
    assert.equal(policy.message.customType, "agentic-pbt-setup");
    await rm(cwd, { recursive: true });
  }

  for (const [result, code] of [
    [{ schemaVersion: 2, task: "repair" }, 0],
    [active, 0],
  ]) {
    const cwd = await fixture();
    const responses = result.schemaVersion === 2
      ? [json(result, code)]
      : [json(result), json({ schemaVersion: 2, followUp: "Run pbt verify" }, 1)];
    const { handlers } = load(responses);
    if (result.schemaVersion === 2) {
      const policy = await handlers.get("before_agent_start")({}, { cwd });
      assert.equal(policy.message.customType, "agentic-pbt-setup");
    } else {
      assert.deepEqual(
        await handlers.get("agent_before_settle")({ outcome: "completed" }, { cwd }),
        {
          entries: [{ type: "custom_message", customType: "agentic-pbt-setup", content: "PBT setup error: pbt returned unsupported schemaVersion", display: true }],
          continue: false,
        },
      );
    }
    await rm(cwd, { recursive: true });
  }

  {
    process.env.PBT_BIN = "relative-pbt";
    const cwd = await fixture();
    const { handlers, calls } = load([]);
    const policy = await handlers.get("before_agent_start")({}, { cwd });
    assert.equal(policy.message.customType, "agentic-pbt-setup");
    assert.equal(calls.length, 0);
    await rm(cwd, { recursive: true });
  }
} finally {
  if (previousPbtBin === undefined) delete process.env.PBT_BIN;
  else process.env.PBT_BIN = previousPbtBin;
}

console.log("PASS: Pi lifecycle adapter contract");
