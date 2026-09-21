import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { test } from "node:test";

const root = path.join(__dirname, "..", "..");
const sourceDirs = ["extension", "ipc", "chat", "progress", "question", "approval", "diff", "result"];

const forbidden = [
  "workspace.fs",
  "workspace.applyEdit",
  "openTextDocument",
  "getText(",
  "child_process",
  "fs.writeFile",
  "fs.promises.writeFile",
  "createWriteStream",
  "workspace.fs.write",
  "workspace.apply_changes",
];

test("extension sources do not perform workspace file I/O, build, or test", () => {
  const files = listTs(sourceDirs.map((d) => path.join(root, d)));
  assert.ok(files.length > 0, "expected extension/ipc TypeScript sources");
  const hits: string[] = [];
  for (const file of files) {
    const text = fs.readFileSync(file, "utf8");
    for (const needle of forbidden) {
      if (text.includes(needle)) {
        hits.push(`${path.relative(root, file)}: ${needle}`);
      }
    }
  }
  assert.deepEqual(hits, []);
});

/** 테스트 디렉터리를 제외한 .ts 파일을 모은다. */
function listTs(dirs: string[]): string[] {
  const out: string[] = [];
  for (const dir of dirs) {
    for (const ent of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, ent.name);
      if (ent.isDirectory()) {
        out.push(...listTs([full]));
        continue;
      }
      if (ent.name.endsWith(".ts")) {
        out.push(full);
      }
    }
  }
  return out;
}
