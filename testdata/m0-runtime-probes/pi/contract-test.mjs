// Deterministic contract checks for probe.ts's follow-up command and
// cancellation tool. Mocks the pi ExtensionAPI (no live/interactive pi
// process) but exercises the real appendFile-based record() through the
// module's own M0_PROBE_LOG contract.
import { readFile } from "node:fs/promises";

const probePath = process.env.M0_PROBE_TS_PATH;
if (!probePath) throw new Error("M0_PROBE_TS_PATH must point at probe.ts");
const logPath = process.env.M0_PROBE_LOG;
if (!logPath) throw new Error("M0_PROBE_LOG must be set");

async function readLog() {
  const text = await readFile(logPath, "utf8");
  return text
    .trim()
    .split("\n")
    .filter(Boolean)
    .map((line) => JSON.parse(line));
}

let failed = false;
function fail(msg) {
  failed = true;
  console.error(`FAIL: ${msg}`);
}
function ok(msg) {
  console.log(`ok - ${msg}`);
}

const mod = await import(probePath);
const commands = new Map();
const tools = new Map();
const events = new Map();
const sent = [];

const pi = {
  on: (name, handler) => events.set(name, handler),
  registerCommand: (name, opts) => commands.set(name, opts),
  registerTool: (tool) => tools.set(tool.name, tool),
  sendMessage: (message, options) => sent.push({ message, options }),
  exec: async () => ({ code: 0, killed: false, stdout: "" }),
};

mod.default(pi);

if (!commands.has("m0-probe-follow-up")) fail('registerCommand("m0-probe-follow-up") was not called');
else ok("registers m0-probe-follow-up command");
if (!tools.has("m0_probe_wait")) fail("registerTool for m0_probe_wait was not called");
else ok("registers m0_probe_wait tool");
if (!events.has("session_before_compact")) fail("registers session_before_compact handler");
else ok("registers session_before_compact handler");

const policy = await events.get("before_agent_start")({}, { cwd: "/tmp/m0-probe-contract-cwd" });
if (policy?.message?.content !== 'M0 validation instruction: if the user says "M0 policy delivery check", reply exactly "M0_PI_POLICY_DELIVERED".')
  fail("before_agent_start does not inject the M0 policy delivery instruction");
else ok("before_agent_start injects the M0 policy delivery instruction");

if (failed) {
  console.error("FAIL: probe.ts contract checks");
  process.exit(1);
}

const cwd = "/tmp/m0-probe-contract-cwd";
const beforeCompact = events.get("session_before_compact");
await beforeCompact({ reason: "manual" }, { cwd });
let compactLog = (await readLog()).filter((e) => e.event === "session_before_compact");
if (compactLog.length !== 1 || compactLog[0].reason !== "manual")
  fail("session_before_compact did not record the compaction reason");
else ok("session_before_compact records the compaction reason");

const followUp = commands.get("m0-probe-follow-up");

await followUp.handler("", { cwd });
let queued = (await readLog()).filter((e) => e.event === "follow_up_queued");
if (queued.length !== 1) fail(`expected exactly one follow_up_queued record, got ${queued.length}`);
else if (queued[0].cwd !== cwd) fail("follow_up_queued did not record ctx.cwd");
else ok("first /m0-probe-follow-up call records follow_up_queued once");

if (sent.length !== 1) {
  fail(`expected exactly one sendMessage call, got ${sent.length}`);
} else {
  const { message, options } = sent[0];
  if (message.customType !== "agentic-pbt-m0-follow-up") fail("sendMessage customType mismatch");
  else if (options?.deliverAs !== "followUp" || options?.triggerTurn !== true)
    fail("sendMessage did not use deliverAs: followUp with triggerTurn: true");
  else ok("follow-up is delivered as a labelled followUp turn, not sendUserMessage");
}

await followUp.handler("", { cwd });
queued = (await readLog()).filter((e) => e.event === "follow_up_queued");
if (queued.length !== 1) fail("second /m0-probe-follow-up call queued another follow-up (should be one-time only)");
else if (sent.length !== 1) fail("second /m0-probe-follow-up call sent another message (should be one-time only)");
else ok("second /m0-probe-follow-up call is a no-op (one-time only)");

const waitTool = tools.get("m0_probe_wait");

pi.exec = async () => ({ code: null, killed: true, stdout: "" });
const cancelled = await waitTool.execute("call-1", {}, { aborted: true }, undefined, { cwd });
let cancelLog = (await readLog()).filter((e) => e.event === "cancellation_probe");
if (cancelLog.length !== 1) fail(`expected one cancellation_probe record, got ${cancelLog.length}`);
else if (cancelLog[0].killed !== true || cancelLog[0].aborted !== true)
  fail("cancellation_probe did not record killed/aborted from the exec result and signal");
else ok("m0_probe_wait records cancellation_probe with killed:true, aborted:true");
if (cancelled.content?.[0]?.text !== "M0_WAIT_CANCELLED") fail("m0_probe_wait did not report M0_WAIT_CANCELLED when killed");
else ok("m0_probe_wait reports M0_WAIT_CANCELLED when killed");

pi.exec = async () => ({ code: 0, killed: false, stdout: "" });
const completed = await waitTool.execute("call-2", {}, undefined, undefined, { cwd });
cancelLog = (await readLog()).filter((e) => e.event === "cancellation_probe");
if (cancelLog.length !== 2) fail(`expected two cancellation_probe records, got ${cancelLog.length}`);
else if (cancelLog[1].killed !== false || cancelLog[1].aborted !== false)
  fail("cancellation_probe recorded wrong killed/aborted for the completed run");
else ok("m0_probe_wait records cancellation_probe with killed:false, aborted:false when no signal");
if (completed.content?.[0]?.text !== "M0_WAIT_COMPLETED") fail("m0_probe_wait did not report M0_WAIT_COMPLETED when not killed");
else ok("m0_probe_wait reports M0_WAIT_COMPLETED when not killed");

if (failed) {
  console.error("FAIL: probe.ts contract checks");
  process.exit(1);
}
console.log("PASS: probe.ts follow-up and cancellation contract checks");
