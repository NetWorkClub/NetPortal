import { NodeItem, WarpEndpoint } from '@core/models';

export interface ConnectPayload {
  mode: string;
  targetName: string;
  standalone: boolean;
}

export interface ProbeOutcome {
  nodeId: string;
  tcpMs: number;
  urlMs: number | null;
  antiBot: 'Clean' | 'Challenge' | 'Error';
}

export interface TransportContract {
  readonly transportKind: 'mock' | 'wails' | 'android' | 'http-ws';
  connect(payload: ConnectPayload): Promise<void>;
  disconnect(): Promise<void>;
  probeNode(node: NodeItem, kind: 'tcp' | 'url'): Promise<ProbeOutcome>;
  scanWarpEndpoints(base: WarpEndpoint[]): Promise<WarpEndpoint[]>;
}
