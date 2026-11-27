// Offline YAML check for .github/workflows/c9-evidence.yml (Review 157
// round, review-resolution §208/§209). It uses prettier's YAML parser from
// web/node_modules; it runs no Go and needs no network.
//
// usage: node yaml-check.mjs <prettier yaml.mjs> <outdir> <label=file>...
// It converts each file to plain data, prints every field except the step
// scripts, compares each later file with the first, and writes every run
// block to <outdir>/<label>.step<N>.<sh|ps1>.
import { readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";
import { pathToFileURL } from "node:url";
import { isDeepStrictEqual } from "node:util";

const [pluginPath, outdir, ...pairs] = process.argv.slice(2);
const plugin = await import(pathToFileURL(pluginPath).href);

function conv(n) {
  if (n == null) return null;
  switch (n.type) {
    case "root": return conv(n.children[0]);
    case "document": return conv(n.children[1]);
    case "documentBody": return conv(n.children[0]);
    case "mapping": {
      const o = {};
      for (const it of n.children) {
        const [k, v] = it.children;
        o[conv(k.children[0])] = conv(v.children[0]);
      }
      return o;
    }
    case "sequence": return n.children.map((it) => conv(it.children[0]));
    case "flowSequence": return n.children.map((it) => conv(it.children[0]));
    case "plain": case "quoteDouble": case "quoteSingle":
    case "blockLiteral": case "blockFolded":
      return n.value;
    default: throw new Error("unexpected node " + n.type);
  }
}

const sha = (s) => createHash("sha256").update(s, "utf8").digest("hex");
const parsed = [];
for (const p of pairs) {
  const i = p.indexOf("=");
  const label = p.slice(0, i), file = p.slice(i + 1);
  const text = readFileSync(file, "utf8");
  const v = conv(plugin.parsers.yaml.parse(text, {}));
  const job = { ...v.jobs.evidence };
  const steps = job.steps.map((s) => ({ ...s }));
  delete job.steps;
  const settings = { name: v.name, on: v.on, permissions: v.permissions, concurrency: v.concurrency, jobs: Object.keys(v.jobs), job };
  console.log(`== ${label}: file sha256 ${sha(text)}`);
  console.log("settings " + JSON.stringify(settings));
  const runs = [];
  steps.forEach((s, n) => {
    const run = s.run;
    delete s.run;
    if (run !== undefined) {
      const ext = s.shell === "pwsh" ? "ps1" : "sh";
      writeFileSync(`${outdir}/${label}.step${n}.${ext}`, run);
      runs.push(run);
    } else runs.push(null);
    console.log(`step ${n} ${JSON.stringify(s)}` + (run !== undefined ? ` run(${run.length} chars, sha256 ${sha(run)})` : ""));
  });
  parsed.push({ label, settings, steps, runs });
}
const base = parsed[0];
for (const p of parsed.slice(1)) {
  console.log(`== ${p.label} vs ${base.label}`);
  console.log("settings equal: " + isDeepStrictEqual(p.settings, base.settings));
  const n = Math.max(p.steps.length, base.steps.length);
  // Match steps by name (or uses) so an inserted step is reported as such.
  const key = (s) => s.name ?? s.uses;
  const baseByKey = new Map(base.steps.map((s, i) => [key(s), i]));
  p.steps.forEach((s, i) => {
    const j = baseByKey.get(key(s));
    if (j === undefined) { console.log(`step ${i} "${key(s)}": new`); return; }
    const fields = isDeepStrictEqual(s, base.steps[j]);
    const run = p.runs[i] === base.runs[j];
    console.log(`step ${i} "${key(s)}": fields ${fields ? "equal" : "DIFFERENT"}, run ${run ? "equal" : "different"}`);
  });
  for (const [k, j] of baseByKey) if (!p.steps.some((s) => key(s) === k)) console.log(`step "${k}" (was ${j}): removed`);
}
