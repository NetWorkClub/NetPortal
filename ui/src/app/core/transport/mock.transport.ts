import { Injectable } from '@angular/core';
import { NodeItem, WarpEndpoint } from '@core/models';
import { ConnectPayload, ProbeOutcome, TransportContract } from './transport.interface';

@Injectable({ providedIn: 'root' })
export class MockTransport implements TransportContract {
  readonly transportKind = 'mock' as const;

  connect(_payload: ConnectPayload): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, 340));
  }

  disconnect(): Promise<void> {
    return Promise.resolve();
  }

  probeNode(node: NodeItem, kind: 'tcp' | 'url'): Promise<ProbeOutcome> {
    const baseDelay = node.iso === 'FI' ? 29 : node.iso === 'DE' ? 37 : node.iso === 'NL' ? 43 : node.iso === 'US' ? 132 : 158;
    const tcpMs = Math.max(18, baseDelay + Math.floor(Math.random() * 14) - 5);
    if (kind === 'tcp') {
      return Promise.resolve({
        nodeId: node.id,
        tcpMs,
        urlMs: node.urlMs,
        antiBot: node.antiBot,
      });
    }
    if (node.id === 'awg20-legacy') {
      return Promise.resolve({
        nodeId: node.id,
        tcpMs,
        urlMs: null,
        antiBot: 'Error',
      });
    }
    const urlMs = tcpMs + 34 + Math.floor(Math.random() * 24);
    const antiBot = node.proto === 'WARP' || node.proto === 'MASQUE' ? 'Challenge' : 'Clean';
    return Promise.resolve({ nodeId: node.id, tcpMs, urlMs, antiBot });
  }

  scanWarpEndpoints(base: WarpEndpoint[]): Promise<WarpEndpoint[]> {
    const next = base
      .map((ep) => ({
        ...ep,
        rtt: Math.max(21, ep.rtt + Math.floor(Math.random() * 12) - 6),
      }))
      .sort((a, b) => a.rtt - b.rtt);
    return Promise.resolve(next);
  }
}
