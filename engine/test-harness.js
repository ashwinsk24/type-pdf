const crypto = require("crypto");
const fs = require("fs");
const path = require("path");
require("./wasm_exec.js");
const Go = globalThis.Go;

const FIXTURE_NAMES = [
  // Original Phase 1 fixtures (10)
  "empty",
  "headings-only",
  "code-heavy",
  "table-heavy",
  "nested-lists",
  "long-lines",
  "multi-page",
  "blockquotes",
  "mixed",
  "basic",
  // Additional coverage fixtures
  "comprehensive",
  "with-image",
  "ordered-lists",
  "only-codeblock",
  "extensions",
  "list-edge-cases",
  "bullet-alignment",
];

async function run() {
  const go = new Go();
  const wasmBuffer = fs.readFileSync("main.wasm");
  const { instance } = await WebAssembly.instantiate(wasmBuffer, go.importObject);
  go.run(instance);

  const fn = globalThis.compileMarkdownToPdf;
  if (!fn) {
    console.error("ERROR: compileMarkdownToPdf not found on globalThis");
    process.exit(1);
  }

  console.log(`WASM binary: ${(wasmBuffer.length / 1024 / 1024).toFixed(2)} MB`);
  console.log(`Fixtures: ${FIXTURE_NAMES.length} total\n`);

  const fixturesDir = path.join(__dirname, "fixtures");
  const outputDir = path.join(__dirname, "wasm-output");
  if (!fs.existsSync(outputDir)) fs.mkdirSync(outputDir);

  const snapshotsPath = path.join(__dirname, "snapshots.json");
  let snapshots = {};
  if (fs.existsSync(snapshotsPath)) {
    snapshots = JSON.parse(fs.readFileSync(snapshotsPath, "utf-8"));
  }
  let newSnapshots = {};

  // Config variants to test
  const configs = [
    { label: "A4/Normal/11pt/Light", pageSize: "a4",     marginMm: 25, baseFontPt: 11, theme: "light", baseDir: fixturesDir },
    { label: "Letter/Narrow/10pt",   pageSize: "letter", marginMm: 15, baseFontPt: 10, theme: "light", baseDir: fixturesDir },
    { label: "A4/Wide/12pt/Draft",   pageSize: "a4",     marginMm: 35, baseFontPt: 12, theme: "draft", baseDir: fixturesDir },
  ];

  let passed = 0;
  let failed = 0;

  // Test each fixture with default config
  for (const name of FIXTURE_NAMES) {
    const mdPath = path.join(fixturesDir, `${name}.md`);
    if (!fs.existsSync(mdPath)) {
      console.log(`  SKIP  ${name} (fixture not found)`);
      continue;
    }

    const mdText = fs.readFileSync(mdPath, "utf-8");

    try {
      const start = performance.now();
      const result = await globalThis.compileMarkdownToPdf(mdText, JSON.stringify({
        pageSize: "a4", marginMm: 25, baseFontPt: 11, theme: "light", baseDir: fixturesDir
      }));
      const elapsed = performance.now() - start;

      if (!result || !result.byteLength) {
        console.log(`  FAIL  ${name}  — no output`);
        failed++;
        continue;
      }

      const outputPath = path.join(outputDir, `${name}.pdf`);
      fs.writeFileSync(outputPath, Buffer.from(result));

      const header = Buffer.from(result.slice(0, 8)).toString("ascii");
      const isValid = header.startsWith("%PDF");
      const sizeKb = (result.byteLength / 1024).toFixed(1);

      const hash = crypto.createHash("sha256").update(Buffer.from(result)).digest("hex");
      newSnapshots[name] = hash;
      const expectedHash = snapshots[name];
      const hashMatch = expectedHash ? (hash === expectedHash) : true;

      if (!hashMatch) {
        console.log(`  FAIL  ${name}  — hash mismatch! Regression detected.`);
        failed++;
      } else {
        console.log(`  ${isValid ? "PASS" : "FAIL"} ${name}  ${sizeKb}KB  ${elapsed}ms  ${isValid ? "" : `(${header})`}`);
        if (isValid) passed++;
        else failed++;
      }
    } catch (err) {
      console.log(`  FAIL  ${name}  — ${err.message}`);
      failed++;
    }
  }

  // Test comprehensive.md with all config variants
  console.log("\n--- Config variants (comprehensive.md) ---");
  const compMdPath = path.join(fixturesDir, "comprehensive.md");
  const compMd = fs.readFileSync(compMdPath, "utf-8");

  for (const cfg of configs) {
    try {
      const start = Date.now();
      const options = JSON.stringify(cfg);
      const result = await globalThis.compileMarkdownToPdf(compMd, options);
      const elapsed = Date.now() - start;

      const isValid = result && result.byteLength > 0 && Buffer.from(result.slice(0, 8)).toString("ascii").startsWith("%PDF");
      const sizeKb = result ? (result.byteLength / 1024).toFixed(1) : "0";

      const hash = crypto.createHash("sha256").update(Buffer.from(result)).digest("hex");
      const compName = `comprehensive_${cfg.label.replace(/[^a-zA-Z0-9]/g, "_")}`;
      newSnapshots[compName] = hash;
      const expectedHash = snapshots[compName];
      const hashMatch = expectedHash ? (hash === expectedHash) : true;

      if (!hashMatch) {
        console.log(`  FAIL  ${cfg.label.padEnd(25)} — hash mismatch!`);
        failed++;
      } else {
        console.log(`  ${isValid ? "PASS" : "FAIL"} ${cfg.label.padEnd(25)} ${sizeKb}KB  ${elapsed}ms`);
        if (isValid) passed++;
        else failed++;
      }
    } catch (err) {
      console.log(`  FAIL  ${cfg.label}  — ${err.message}`);
      failed++;
    }
  }

  // Edge case: pass empty string
  try {
    const result = await globalThis.compileMarkdownToPdf("", JSON.stringify({ pageSize: "a4", marginMm: 25, baseFontPt: 11, theme: "light", baseDir: fixturesDir }));
    const isValid = result && result.byteLength > 0 && Buffer.from(result.slice(0, 8)).toString("ascii").startsWith("%PDF");
    console.log(`  ${isValid ? "PASS" : "FAIL"} ${"Empty string".padEnd(25)} ${result ? (result.byteLength/1024).toFixed(1) : "0"}KB`);
    if (isValid) passed++; else failed++;
  } catch (err) {
    console.log(`  FAIL  Empty string  — ${err.message}`);
    failed++;
  }

  fs.writeFileSync(snapshotsPath, JSON.stringify(newSnapshots, null, 2));

  console.log(`\n=== Results: ${passed} passed, ${failed} failed ===`);
  process.exit(failed > 0 ? 1 : 0);
}

run().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
