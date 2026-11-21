export type NodeProtoGroup = 'awg' | 'proxy' | 'masque' | 'warp' | 'psiphon';

export type AntiBotState = 'Clean' | 'Challenge' | 'Error';

export interface SourceItem {
  id: string;
  name: string;
  meta: string;
  quota?: string;
  quotaKey?: string;
}

export interface NodeItem {
  id: string;
  sourceId: string;
  iso: string;
  name: string;
  endpoint: string;
  proto: string;
  protoGroup: NodeProtoGroup;
  tcpMs: number | null;
  urlMs: number | null;
  antiBot: AntiBotState;
  probing: boolean;
  mtu: number;
  params: Record<string, string>;
}

export interface WarpEndpoint {
  addr: string;
  region: string;
  rtt: number;
  loss: string;
}

export interface ImportDetectionResult {
  proto: string;
  protoGroup: NodeProtoGroup;
  label: string;
}
