import { NodeItem, SourceItem, WarpEndpoint } from '@core/models';

export interface ConnectPayload {
  mode: string;
  targetName: string;
  nodeId?: string;
  standalone: boolean;
}

export interface ProbeOutcome {
  nodeId: string;
  tcpMs: number;
  urlMs: number | null;
  antiBot: 'Clean' | 'Challenge' | 'Error';
  error?: string;
}

export interface ImportResult {
  source: SourceItem;
  nodes: NodeItem[];
}

export interface TransportContract {
  readonly transportKind: 'mock' | 'wails' | 'android' | 'http-ws';
  connect(payload: ConnectPayload): Promise<void>;
  disconnect(): Promise<void>;
  probeNode(node: NodeItem, kind: 'tcp' | 'url'): Promise<ProbeOutcome>;
  scanWarpEndpoints(base: WarpEndpoint[]): Promise<WarpEndpoint[]>;
  importConfig?(name: string, content: string): Promise<ImportResult>;
  exportDiagnostics?(): Promise<string>;
}
