/* ------------------------------------------------------------------ */
/*  WebSocket connection manager & API types                           */
/*  WS для check / accept / reject / applyAll (не REST — FRONTEND.md)  */
/* ------------------------------------------------------------------ */

export type ConnectionStatus = "online" | "reconnecting" | "offline";

/* ── Client → Server message types ── */

interface CheckMessage {
  type: "check";
  text: string;
  textHash: string;
}

interface AcceptMessage {
  type: "accept";
  flagId: string;
}

interface RejectMessage {
  type: "reject";
  flagId: string;
}

interface ApplyAllMessage {
  type: "applyAll";
  flagIds: string[];
}

export type ClientMessage =
  | CheckMessage
  | AcceptMessage
  | RejectMessage
  | ApplyAllMessage;

/* ── Server → Client message types ── */

export interface CheckResultMessage {
  type: "check_result";
  textHash: string;
  flags: import("./store").ServerFlag[];
  scores: import("./store").Scores;
  session_id?: string;
}

export interface AckMessage {
  type: "ack";
  action: "accept" | "reject" | "applyAll";
  flagId?: string;
  flagIds?: string[];
}

export type ServerMessage = CheckResultMessage | AckMessage;

/* ── Callbacks ── */

export interface WSCallbacks {
  onMessage: (msg: ServerMessage) => void;
  onStatusChange: (status: ConnectionStatus) => void;
}

/* ── Connection manager ── */

const RECONNECT_DELAYS = [1000, 2000, 5000]; // exponential backoff cap

export class WSConnection {
  private ws: WebSocket | null = null;
  private closed = false;
  private reconnectAttempt = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private callbacks: WSCallbacks | null = null;

  connect(callbacks: WSCallbacks): void {
    this.callbacks = callbacks;
    this.closed = false;
    this.doConnect();
  }

  disconnect(): void {
    this.closed = true;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
    this.ws = null;
    this.callbacks?.onStatusChange("offline");
  }

  send(msg: ClientMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
    }
  }

  get connected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  private doConnect(): void {
    if (this.closed) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host || "localhost:8080";
    const ws = new WebSocket(`${protocol}//${host}/ws/live`);

    ws.onopen = () => {
      this.reconnectAttempt = 0;
      this.callbacks?.onStatusChange("online");
    };

    ws.onclose = () => {
      this.callbacks?.onStatusChange("reconnecting");
      this.scheduleReconnect();
    };

    ws.onerror = () => {
      ws.close();
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as ServerMessage;
        this.callbacks?.onMessage(data);
      } catch {
        // ignore malformed
      }
    };

    this.ws = ws;
  }

  private scheduleReconnect(): void {
    if (this.closed) return;
    const delay =
      RECONNECT_DELAYS[Math.min(this.reconnectAttempt, RECONNECT_DELAYS.length - 1)];
    this.reconnectAttempt++;
    this.reconnectTimer = setTimeout(() => this.doConnect(), delay);
  }
}

/* ------------------------------------------------------------------ */
/*  REST helpers                                                       */
/* ------------------------------------------------------------------ */

export interface VersionInfo {
  version: string;
  module?: string;
  component?: string;
  commit?: string;
  build_time?: string;
  license?: string;
}

export async function fetchVersion(): Promise<VersionInfo> {
  const res = await fetch("/api/version");
  if (!res.ok) {
    throw new Error(`Failed to fetch version: ${res.status}`);
  }
  const contentType = res.headers.get("content-type") ?? "";
  if (!contentType.includes("application/json")) {
    throw new Error("Version endpoint returned non-JSON response");
  }
  return res.json() as Promise<VersionInfo>;
}
