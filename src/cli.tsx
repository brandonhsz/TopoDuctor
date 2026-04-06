#!/usr/bin/env node
import React from "react";
import { readFileSync } from "node:fs";
import { parseArgs } from "node:util";
import { render } from "ink";
import { App } from "./App.js";
import { printOnlyLine, runExitAction, type ExitPayload } from "./exit/runExit.js";

const pkg = JSON.parse(
  readFileSync(new URL("../package.json", import.meta.url), "utf8")
) as { version?: string };

const { values } = parseArgs({
  args: process.argv.slice(2),
  options: {
    "print-only": { type: "boolean", default: false },
    version: { type: "boolean", default: false },
  },
  strict: false,
  allowPositionals: false,
});

if (values.version) {
  console.log(pkg.version ?? "dev");
  process.exit(0);
}

const cwd = process.cwd();
const version = pkg.version ?? "dev";
const printOnly = values["print-only"] === true;

const exitOutcomeRef: { current: ExitPayload | null | undefined } = {
  current: undefined,
};

const inst = render(
  <App cwd={cwd} version={version} exitOutcomeRef={exitOutcomeRef} />
);

await inst.waitUntilExit();

const outcome = exitOutcomeRef.current;
if (!outcome) {
  process.exit(0);
}

if (printOnly) {
  console.log(printOnlyLine(outcome));
  process.exit(0);
}

try {
  await runExitAction(outcome);
} catch (e) {
  console.error(e instanceof Error ? e.message : String(e));
  process.exit(1);
}
