import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const scope = (...files) =>
  spawnSync("bash", ["scripts/ci-platform-scope.sh"], { cwd: root, input: files.join("\n"), encoding: "utf8" }).stdout.trim();

test("neutral paths do not need the macOS and Windows legs", () => {
  assert.equal(scope("docs/a.md", "README.md", "release-notes/studio/1.md"), "false");
  assert.equal(scope("desktop/frontend-next/src/a.tsx", "workers/crash-report/src/a.ts"), "false");
  assert.equal(scope("internal/model/openai/client.go", "internal/contract/tool/tool.go"), "false");
});

test("platform paths and unknown paths need them", () => {
  for (const f of [
    "internal/safety/sandbox/a.go",
    "internal/platform/update/a.go",
    "cmd/reasonix/main.go",
    "desktop/electron/main.ts",
    "go.mod",
    ".github/workflows/ci.yml",
    "scripts/install-smoke.ps1",
    "internal/session/control/a.go",
    "unknown-file.txt",
  ]) assert.equal(scope("docs/a.md", f), "true", f);
});

test("a package carrying an OS-constrained file is platform-relevant whichever file changed", () => {
  assert.equal(scope("internal/frontend/cli/root.go"), "true");
  assert.equal(scope("internal/model/openai/other_windows.go"), "true");
});

test("an empty diff fails toward running everything", () => {
  assert.equal(scope(), "true");
});
