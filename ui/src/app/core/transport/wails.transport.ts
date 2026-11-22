import { NodeItem, WarpEndpoint } from '@core/models';
import { ConnectPayload, ImportResult, ProbeOutcome, TransportContract } from './transport.interface';

interface WailsAppBindings {
  Connect(payload: ConnectPayload): Promise<void>;
  Disconnect(): Promise<void>;
  ProbeNode(node: NodeItem, kind: 'tcp' | 'url'): Promise<ProbeOutcome>;
  ImportConfig(req: { name: string; content: string }): Promise<ImportResult>;
  ExportDiagnostics(): Promise<string>;
}

interface WailsWindow extends Window {
  go?: {
    main?: {
      App?: WailsAppBindings;
    };
  };
}

export class WailsTransport implements TransportContract {
  readonly transportKind = 'wails' as const;

  private get app(): WailsAppBindings {
    const bindings = (window as WailsWindow).go?.main?.App;
    if (!bindings) {
      throw new Error('Wails App bindings are not available');
    }
    return bindings;
  }

  async connect(payload: ConnectPayload): Promise<void> {
    await this.app.Connect(payload);
  }

  async disconnect(): Promise<void> {
    await this.app.Disconnect();
  }

  async probeNode(node: NodeItem, kind: 'tcp' | 'url'): Promise<ProbeOutcome> {
    return this.app.ProbeNode(node, kind);
  }

  async scanWarpEndpoints(base: WarpEndpoint[]): Promise<WarpEndpoint[]> {
    return base;
  }

  async importConfig(name: string, content: string): Promise<ImportResult> {
    return this.app.ImportConfig({ name, content });
  }

  async exportDiagnostics(): Promise<string> {
    return this.app.ExportDiagnostics();
  }
}
