import { NodeItem, WarpEndpoint } from '@core/models';
import {
  ConnectPayload,
  ImportResult,
  ProbeOutcome,
  StatusSnapshot,
  TransportContract,
} from './transport.interface';

interface AndroidNativeBridge {
  connect(payloadJson: string): void;
  disconnect(): void;
  probeNode(nodeJson: string, kind: string): string;
  importConfig(name: string, content: string): string;
  listNodes(): string;
  status(): string;
}

interface AndroidWindow extends Window {
  NetPortalBridge?: AndroidNativeBridge;
}

export class AndroidBridgeTransport implements TransportContract {
  readonly transportKind = 'android' as const;

  private get bridge(): AndroidNativeBridge {
    const native = (window as AndroidWindow).NetPortalBridge;
    if (!native) {
      throw new Error('Android NetPortalBridge is not available');
    }
    return native;
  }

  async connect(payload: ConnectPayload): Promise<void> {
    this.bridge.connect(JSON.stringify(payload));
  }

  async disconnect(): Promise<void> {
    this.bridge.disconnect();
  }

  async probeNode(node: NodeItem, kind: 'tcp' | 'url'): Promise<ProbeOutcome> {
    const raw = this.bridge.probeNode(JSON.stringify(node), kind);
    return JSON.parse(raw) as ProbeOutcome;
  }

  async scanWarpEndpoints(base: WarpEndpoint[]): Promise<WarpEndpoint[]> {
    return base;
  }

  async importConfig(name: string, content: string): Promise<ImportResult> {
    return JSON.parse(this.bridge.importConfig(name, content)) as ImportResult;
  }

  async listNodes(): Promise<NodeItem[]> {
    return JSON.parse(this.bridge.listNodes()) as NodeItem[];
  }

  async getStatus(): Promise<StatusSnapshot> {
    return JSON.parse(this.bridge.status()) as StatusSnapshot;
  }
}
