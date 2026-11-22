import { NodeItem, SourceItem, WarpEndpoint } from '@core/models';
import {
  ConnectPayload,
  ImportResult,
  ProbeOutcome,
  StatusSnapshot,
  TransportContract,
} from './transport.interface';

const TOKEN_STORAGE_KEY = 'netportal_web_token';

export class HttpWsTransport implements TransportContract {
  readonly transportKind = 'http-ws' as const;
  private readonly token: string;

  constructor(token?: string) {
    const urlToken = new URLSearchParams(window.location.search).get('token');
    if (urlToken) {
      sessionStorage.setItem(TOKEN_STORAGE_KEY, urlToken);
    }
    this.token = token ?? urlToken ?? sessionStorage.getItem(TOKEN_STORAGE_KEY) ?? '';
  }

  private async getJson<T>(path: string): Promise<T> {
    const res = await fetch(path, {
      headers: {
        Authorization: `Bearer ${this.token}`,
      },
    });
    if (!res.ok) {
      const msg = await res.text();
      throw new Error(msg || `HTTP ${res.status}`);
    }
    return (await res.json()) as T;
  }

  private async postJson<T>(path: string, body: unknown): Promise<T> {
    const res = await fetch(path, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${this.token}`,
      },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      const msg = await res.text();
      throw new Error(msg || `HTTP ${res.status}`);
    }
    return (await res.json()) as T;
  }

  async connect(payload: ConnectPayload): Promise<void> {
    await this.postJson('/api/connect', payload);
  }

  async disconnect(): Promise<void> {
    await this.postJson('/api/disconnect', {});
  }

  async probeNode(node: NodeItem, kind: 'tcp' | 'url'): Promise<ProbeOutcome> {
    return this.postJson<ProbeOutcome>('/api/probe', { node, kind });
  }

  async scanWarpEndpoints(base: WarpEndpoint[]): Promise<WarpEndpoint[]> {
    return base;
  }

  async importConfig(name: string, content: string): Promise<ImportResult> {
    return this.postJson<ImportResult>('/api/import', { name, content });
  }

  async exportDiagnostics(): Promise<string> {
    const res = await fetch('/api/diagnostics', {
      headers: { Authorization: `Bearer ${this.token}` },
    });
    if (!res.ok) {
      throw new Error(`HTTP ${res.status}`);
    }
    return res.text();
  }

  async listSources(): Promise<SourceItem[]> {
    return this.getJson<SourceItem[]>('/api/sources');
  }

  async listNodes(): Promise<NodeItem[]> {
    return this.getJson<NodeItem[]>('/api/nodes');
  }

  async getStatus(): Promise<StatusSnapshot> {
    return this.getJson<StatusSnapshot>('/api/status');
  }
}
