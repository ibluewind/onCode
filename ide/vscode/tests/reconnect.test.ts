import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { test } from "node:test";
import { WebSocketServer } from "ws";
import { AgentIpcClient } from "../ipc/client";
import { parseEndpoint, readEndpointFile, resolveStateDir } from "../ipc/endpoint";
import { IPC_TYPE_REGISTER, IPC_TYPE_REGISTERED, type IpcMessage } from "../ipc/messages";

test("parseEndpoint rejects incomplete json", () => {
  assert.throws(() => parseEndpoint("{}"));
  assert.throws(() =>
    parseEndpoint(JSON.stringify({ bind: "127.0.0.1", port: 0, path: "/ipc", token: "x" })),
  );
});

test("readEndpointFile returns null when missing", () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "oncode-ep-"));
  assert.equal(readEndpointFile(dir), null);
});

test("resolveStateDir prefers override then env", () => {
  const prev = process.env.ONCODE_AGENT_STATE_DIR;
  process.env.ONCODE_AGENT_STATE_DIR = path.join(os.tmpdir(), "from-env");
  try {
    assert.equal(resolveStateDir(""), path.resolve(path.join(os.tmpdir(), "from-env")));
    assert.equal(
      resolveStateDir(path.join(os.tmpdir(), "override")),
      path.resolve(path.join(os.tmpdir(), "override")),
    );
  } finally {
    if (prev === undefined) {
      delete process.env.ONCODE_AGENT_STATE_DIR;
    } else {
      process.env.ONCODE_AGENT_STATE_DIR = prev;
    }
  }
});

test("client registers then reconnects after server restart", async () => {
  const token = "test-token-slice1";
  const registrations: string[] = [];
  let port = 0;
  let wss = await listenMock(token, registrations, (p) => {
    port = p;
  });

  const logs: string[] = [];
  const client = new AgentIpcClient({
    extensionVersion: "0.1.0",
    log: (line) => logs.push(line),
    sleep: (ms) => new Promise((r) => setTimeout(r, Math.min(ms, 30))),
    loadEndpoint: () => ({
      bind: "127.0.0.1",
      port,
      path: "/ipc",
      token,
    }),
  });
  void client.start();
  await waitUntil(() => registrations.length >= 1, 3000);
  assert.ok((client.ideClientId ?? "").length > 0);
  assert.ok(logs.some((l) => l.includes("Connected")));

  await closeServer(wss);
  wss = await listenMock(token, registrations, (p) => {
    port = p;
  });
  await waitUntil(() => registrations.length >= 2, 4000);
  client.dispose();
  await closeServer(wss);
});

test("client does not connect without a token endpoint", async () => {
  const logs: string[] = [];
  const client = new AgentIpcClient({
    extensionVersion: "0.1.0",
    log: (line) => logs.push(line),
    sleep: (ms) => new Promise((r) => setTimeout(r, Math.min(ms, 20))),
    loadEndpoint: () => null,
  });
  void client.start();
  await waitUntil(() => logs.some((l) => l.includes("Local Agent unavailable")), 1000);
  client.dispose();
});

function listenMock(
  token: string,
  registrations: string[],
  onListen: (port: number) => void,
): Promise<WebSocketServer> {
  return new Promise((resolve, reject) => {
    const wss = new WebSocketServer({ host: "127.0.0.1", port: 0 });
    wss.on("error", reject);
    wss.on("listening", () => {
      const addr = wss.address();
      if (typeof addr === "object" && addr) {
        onListen(addr.port);
      }
      resolve(wss);
    });
    wss.on("connection", (socket, req) => {
      const url = new URL(req.url ?? "/", "http://127.0.0.1");
      if (url.searchParams.get("token") !== token) {
        socket.close();
        return;
      }
      socket.on("message", (raw) => {
        const msg = JSON.parse(raw.toString()) as IpcMessage;
        if (msg.type !== IPC_TYPE_REGISTER) {
          socket.close();
          return;
        }
        const id = `ide-mock-${registrations.length + 1}`;
        registrations.push(id);
        const reply: IpcMessage = {
          type: IPC_TYPE_REGISTERED,
          message_id: msg.message_id,
          data: { ide_client_id: id, agent_id: "agent-test" },
        };
        socket.send(JSON.stringify(reply));
      });
    });
  });
}

function closeServer(wss: WebSocketServer): Promise<void> {
  return new Promise((resolve) => {
    for (const client of wss.clients) {
      client.terminate();
    }
    wss.close(() => resolve());
  });
}

async function waitUntil(ok: () => boolean, timeoutMs: number): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (ok()) {
      return;
    }
    await new Promise((r) => setTimeout(r, 20));
  }
  throw new Error("timeout");
}
