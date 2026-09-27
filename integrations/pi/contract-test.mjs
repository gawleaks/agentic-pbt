import assert from "node:assert/strict";
import { mkdtemp, mkdir, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const extension = await import(pathToFileURL(join(process.cwd(), "integrations/pi/index.ts")).href);

async function fixture(active = true) {
  const cwd = await mkdtemp(join(tmpdir(), "agentic-pbt-pi-"));
  if (active) {
    await mkdir(join(cwd, ".pbt/state"), { recursive: true });
    await writeFile(join(cwd, ".pbt/state/active.json"), '{"task":"repair"}\n');
  }
  return cwd;
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

const previousPbtBin = process.env.PBT_BIN;
try {
  delete process.env.PBT_BIN;

  {
    const cwd = await fixture();
    const { handlers, calls } = load([
      json({ schemaVersion: 1, task: "repair" }),
      json({ schemaVersion: 1, followUp: "Run pbt verify" }, 1),
      json({ schemaVersion: 1, followUp: "" }, 1),
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
    assert.equal(await boundary({ outcome: "completed" }, { cwd }), undefined);
    assert.deepEqual(calls.map(({ args }) => args), [
      ["status", "--json"],
      ["gate", "--task", "repair", "--json"],
      ["gate", "--task", "repair", "--json"],
    ]);
    assert.equal(await boundary({ outcome: "aborted" }, { cwd }), undefined);
    assert.equal(await boundary({ outcome: "error" }, { cwd }), undefined);
    await rm(cwd, { recursive: true });
  }

  {
    const cwd = await fixture(false);
    const { handlers, calls } = load([]);
    assert.equal(await handlers.get("before_agent_start")({}, { cwd }), undefined);
    assert.equal(await handlers.get("agent_before_settle")({ outcome: "completed" }, { cwd }), undefined);
    assert.equal(calls.length, 0);
    await rm(cwd, { recursive: true });
  }

  {
    const cwd = await fixture();
    const { handlers } = load([missing(), missing()]);
    const policy = await handlers.get("before_agent_start")({}, { cwd });
    assert.equal(policy.message.customType, "agentic-pbt-setup");
    assert.equal(policy.message.display, true);
    assert.deepEqual(
      await handlers.get("agent_before_settle")({ outcome: "completed" }, { cwd }),
      {
        entries: [{ type: "custom_message", customType: "agentic-pbt-setup", content: "PBT setup error: pbt: command not found", display: true }],
        continue: false,
      },
    );
    await rm(cwd, { recursive: true });
  }

  {
    const cwd = await fixture();
    const { handlers } = load([json({ schemaVersion: 1, reasonCode: "setup_error" }, 3, "invalid PBT config")]);
    assert.deepEqual(
      await handlers.get("agent_before_settle")({ outcome: "completed" }, { cwd }),
      {
        entries: [{ type: "custom_message", customType: "agentic-pbt-setup", content: "PBT setup error: invalid PBT config", display: true }],
        continue: false,
      },
    );
    await rm(cwd, { recursive: true });
  }

  for (const [result, code] of [
    [{ schemaVersion: 1, followUp: "" }, 1],
    [{ schemaVersion: 1, workflowState: "blocked" }, 2],
    [{ schemaVersion: 1, workflowState: "complete" }, 0],
  ]) {
    const cwd = await fixture();
    const { handlers } = load([json(result, code)]);
    assert.equal(await handlers.get("agent_before_settle")({ outcome: "completed" }, { cwd }), undefined);
    await rm(cwd, { recursive: true });
  }

  {
    process.env.PBT_BIN = "relative-pbt";
    const cwd = await fixture();
    const { handlers, calls } = load([]);
    const policy = await handlers.get("before_agent_start")({}, { cwd });
    assert.equal(policy.message.customType, "agentic-pbt-setup");
    assert.match(policy.message.content, /PBT_BIN must be an absolute path/);
    assert.deepEqual(
      await handlers.get("agent_before_settle")({ outcome: "completed" }, { cwd }),
      {
        entries: [{ type: "custom_message", customType: "agentic-pbt-setup", content: "PBT setup error: PBT_BIN must be an absolute path", display: true }],
        continue: false,
      },
    );
    assert.equal(calls.length, 0);
    await rm(cwd, { recursive: true });
  }
} finally {
  if (previousPbtBin === undefined) delete process.env.PBT_BIN;
  else process.env.PBT_BIN = previousPbtBin;
}

console.log("PASS: Pi lifecycle adapter contract");
