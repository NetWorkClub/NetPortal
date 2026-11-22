import { AndroidBridgeTransport } from './android-bridge.transport';
import { HttpWsTransport } from './http-ws.transport';
import { MockTransport } from './mock.transport';
import { TransportContract } from './transport.interface';
import { WailsTransport } from './wails.transport';

interface RuntimeWindow extends Window {
  NetPortalBridge?: unknown;
  go?: { main?: { App?: unknown } };
}

export function createRuntimeTransport(): TransportContract {
  const w = window as RuntimeWindow;
  if (w.NetPortalBridge) {
    return new AndroidBridgeTransport();
  }
  if (w.go?.main?.App) {
    return new WailsTransport();
  }
  const params = new URLSearchParams(window.location.search);
  if (params.has('token') || sessionStorage.getItem('netportal_web_token')) {
    return new HttpWsTransport();
  }
  return new MockTransport();
}
