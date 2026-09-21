import fs from "node:fs";
import os from "node:os";
import path from "node:path";

const ENDPOINT_FILE = "ide-ipc.json";

/**
 * Local Agent가 state dir에 쓰는 IDE IPC 디스커버리 정보.
 * 워크스페이스 파일이 아니며, 이 모듈은 이 파일만 읽는다.
 */
export interface IpcEndpoint {
  bind: string;
  port: number;
  path: string;
  token: string;
}

/**
 * ide-ipc.json 본문을 검증한다. bind/path/token이 비었거나 port가 1 미만이면 거부한다.
 *
 * @param raw UTF-8 JSON 문자열. 빈 값은 안 된다.
 * @returns 파싱된 endpoint
 */
export function parseEndpoint(raw: string): IpcEndpoint {
  const parsed = JSON.parse(raw) as Partial<IpcEndpoint>;
  if (
    typeof parsed.bind !== "string" ||
    parsed.bind.trim() === "" ||
    typeof parsed.port !== "number" ||
    parsed.port < 1 ||
    typeof parsed.path !== "string" ||
    parsed.path.trim() === "" ||
    typeof parsed.token !== "string" ||
    parsed.token.trim() === ""
  ) {
    throw new Error("incomplete ide ipc endpoint");
  }
  return {
    bind: parsed.bind,
    port: parsed.port,
    path: parsed.path,
    token: parsed.token,
  };
}

/**
 * Local Agent state dir을 고른다. 워크스페이스 루트가 아니라 에이전트 설정 디렉터리다.
 *
 * @param override 확장 설정값. 빈 문자열이면 환경변수/플랫폼 기본값
 */
export function resolveStateDir(override?: string): string {
  if (override && override.trim() !== "") {
    return path.resolve(override);
  }
  if (process.env.ONCODE_AGENT_STATE_DIR) {
    return path.resolve(process.env.ONCODE_AGENT_STATE_DIR);
  }
  return path.join(userConfigDir(), "oncode-agent");
}

/**
 * state dir의 ide-ipc.json을 동기 읽기한다. 없거나 깨졌으면 null이다.
 * 워크스페이스 소스는 읽지 않는다.
 *
 * @param stateDir Local Agent state dir 절대 경로
 */
export function readEndpointFile(stateDir: string): IpcEndpoint | null {
  const file = path.join(stateDir, ENDPOINT_FILE);
  try {
    const raw = fs.readFileSync(file, "utf8");
    return parseEndpoint(raw);
  } catch {
    return null;
  }
}

/** Go os.UserConfigDir()과 같은 플랫폼 설정 루트. */
function userConfigDir(): string {
  if (process.platform === "win32") {
    return process.env.APPDATA || path.join(os.homedir(), "AppData", "Roaming");
  }
  if (process.platform === "darwin") {
    return path.join(os.homedir(), "Library", "Application Support");
  }
  return process.env.XDG_CONFIG_HOME || path.join(os.homedir(), ".config");
}
