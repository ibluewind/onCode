import { randomUUID } from "node:crypto";
import WebSocket from "ws";
import { type IpcEndpoint } from "./endpoint";
import { IPC_TYPE_REGISTER, IPC_TYPE_REGISTERED, type IpcMessage } from "./messages";

export type IpcLogger = (line: string) => void;
export type EndpointLoader = () => IpcEndpoint | null;

const INITIAL_BACKOFF_MS = 200;
const MAX_BACKOFF_MS = 5000;

export interface AgentIpcClientOptions {
  loadEndpoint: EndpointLoader;
  log?: IpcLogger;
  extensionVersion: string;
  /** 테스트에서 백오프를 줄이기 위한 대기 함수. 단위는 ms. */
  sleep?: (ms: number) => Promise<void>;
}

/**
 * Local Agent WebSocket 세션을 유지한다.
 * 워크스페이스 파일 I/O·빌드·테스트는 하지 않는다. 토큰은 endpoint 로더가 준 값만 쓰고 디스크에 쓰지 않는다.
 */
export class AgentIpcClient {
  private readonly loadEndpoint: EndpointLoader;
  private readonly log: IpcLogger;
  private readonly extensionVersion: string;
  private readonly sleep: (ms: number) => Promise<void>;
  private disposed = false;
  private backoffMs = INITIAL_BACKOFF_MS;
  private socket: WebSocket | undefined;
  private loopPromise: Promise<void> | undefined;
  private readonly listeners: Array<(msg: IpcMessage) => void> = [];
  ideClientId: string | undefined;

  constructor(options: AgentIpcClientOptions) {
    this.loadEndpoint = options.loadEndpoint;
    this.log = options.log ?? (() => undefined);
    this.extensionVersion = options.extensionVersion;
    this.sleep = options.sleep ?? defaultSleep;
  }

  /**
   * 백그라운드에서 연결 루프를 시작한다. 이미 돌고 있으면 아무 것도 하지 않는다.
   * Local Agent가 없거나 끊기면 지수 백오프로 재연결한다.
   */
  start(): Promise<void> {
    if (this.loopPromise) {
      return this.loopPromise;
    }
    this.disposed = false;
    this.loopPromise = this.runLoop();
    return this.loopPromise;
  }

  /**
   * 현재 소켓을 닫아 루프가 곧바로 다시 붙게 한다. dispose 이후에는 연결하지 않는다.
   */
  reconnect(): void {
    if (this.disposed) {
      return;
    }
    this.backoffMs = INITIAL_BACKOFF_MS;
    this.closeSocket();
  }

  /**
   * 재연결 루프를 멈추고 소켓을 닫는다. 이후 start를 다시 호출하기 전에 새 인스턴스를 쓰는 편이 안전하다.
   */
  dispose(): void {
    this.disposed = true;
    this.closeSocket();
  }

  /**
   * 서버/에이전트가 푸시한 프레임을 받는다. 워크스페이스 파일은 읽지 않는다.
   */
  onMessage(listener: (msg: IpcMessage) => void): void {
    this.listeners.push(listener);
  }

  /**
   * 열려 있는 소켓으로 JSON 프레임을 보낸다. 연결 전이면 false.
   */
  send(msg: IpcMessage): boolean {
    if (!this.isOpen()) {
      return false;
    }
    this.socket!.send(JSON.stringify(msg));
    return true;
  }

  /**
   * Local Agent IPC 소켓이 OPEN이면 true. 연결 전·재연결 중이면 false.
   */
  isOpen(): boolean {
    return this.socket !== undefined && this.socket.readyState === WebSocket.OPEN;
  }

  private async runLoop(): Promise<void> {
    while (!this.disposed) {
      const endpoint = this.loadEndpoint();
      if (!endpoint) {
        this.log("onCode: Local Agent unavailable");
        await this.waitBackoff();
        continue;
      }
      try {
        await this.connectOnce(endpoint);
        this.backoffMs = INITIAL_BACKOFF_MS;
        if (!this.disposed) {
          this.log("onCode: Reconnecting");
        }
      } catch {
        if (!this.disposed) {
          this.log("onCode: Reconnecting");
        }
      }
      await this.waitBackoff();
    }
  }

  /**
   * 한 번 연결하고 ide.register를 보낸 뒤, 소켓이 닫힐 때까지 대기한다.
   * 토큰이 틀리면 WebSocket이 바로 실패하고 호출자에게 오류를 던진다.
   *
   * @param endpoint bind는 루프백이어야 한다. token은 URL 쿼리로만 전달한다.
   */
  private connectOnce(endpoint: IpcEndpoint): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = `ws://127.0.0.1:${endpoint.port}${endpoint.path}?token=${encodeURIComponent(endpoint.token)}`;
      const ws = new WebSocket(url);
      this.socket = ws;
      let settled = false;

      const finish = (err?: Error): void => {
        if (settled) {
          return;
        }
        settled = true;
        if (this.socket === ws) {
          this.socket = undefined;
        }
        if (err) {
          reject(err);
          return;
        }
        resolve();
      };

      ws.on("open", () => {
        const msg: IpcMessage = {
          type: IPC_TYPE_REGISTER,
          message_id: randomUUID(),
          data: {
            ide: "VSCODE",
            extension_version: this.extensionVersion,
          },
        };
        ws.send(JSON.stringify(msg));
      });
      ws.on("message", (raw) => {
        const text = raw.toString();
        let parsed: IpcMessage;
        try {
          parsed = JSON.parse(text) as IpcMessage;
        } catch {
          return;
        }
        if (parsed.type === IPC_TYPE_REGISTERED) {
          const id = parsed.data?.ide_client_id;
          if (typeof id === "string") {
            this.ideClientId = id;
          }
          this.log("onCode: Connected");
        }
        for (const listener of this.listeners) {
          listener(parsed);
        }
      });
      ws.on("error", (err) => {
        finish(err);
      });
      ws.on("unexpected-response", (_req, res) => {
        const code = res.statusCode;
        res.resume();
        finish(new Error(`ide ipc handshake rejected: ${code}`));
      });
      ws.on("close", () => {
        finish();
      });
    });
  }

  private closeSocket(): void {
    if (!this.socket) {
      return;
    }
    try {
      this.socket.close();
    } catch {
      // already closed
    }
    this.socket = undefined;
  }

  private async waitBackoff(): Promise<void> {
    if (this.disposed) {
      return;
    }
    await this.sleep(this.backoffMs);
    this.backoffMs = Math.min(this.backoffMs * 2, MAX_BACKOFF_MS);
  }
}

function defaultSleep(ms: number): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}
