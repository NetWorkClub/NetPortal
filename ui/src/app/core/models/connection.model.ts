export type PlatformTarget = 'desktop' | 'android';

export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected';

export type HomeMethod = 'all' | 'awg' | 'proxy' | 'masque' | 'psiphon' | 'novpn';

export type NovpnMode = 'byedpi' | 'zapret' | 'smartdns' | 'byedpi_smartdns';

export type ServiceMode = 'vpn' | 'system-proxy' | 'local-proxy';

export type SmartDnsMode = 'proxy' | 'tun' | 'hosts';

export type ChainCascade = 'none' | 'psiphon' | 'warp';

export interface SignalHop {
  title: string;
  meta?: string;
  tone?: 'default' | 'active' | 'failover' | 'failed';
}
