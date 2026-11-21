export type RuleTier = 'app' | 'network' | 'combined';

export type AppMatchMode = 'off' | 'include' | 'exclude';

export type TransportNetwork = 'both' | 'tcp' | 'udp';

export type IpVersionFilter = 'any' | 'ipv4' | 'ipv6';

export type InboundFilter = 'any' | 'tun' | 'proxy' | 'lan';

export interface RoutingRule {
  id: string;
  presetId?: string;
  name: string;
  tier: RuleTier;
  enabled: boolean;
  invert: boolean;
  continueToNetwork: boolean;
  appMode: AppMatchMode;
  apps: string[];
  androidApps: string[];
  appPaths: string[];
  domains: string[];
  suffixes: string[];
  keywords: string[];
  regex: string;
  lists: string[];
  cidrs: string[];
  ipPools: string[];
  geoip: string[];
  network: TransportNetwork;
  ports: string[];
  l7: string[];
  ipVer: IpVersionFilter;
  inbound: InboundFilter;
  target: string;
}

export interface RoutingProfile {
  id: string;
  nameRu: string;
  nameEn: string;
  defaultTarget: string;
  rules: RoutingRule[];
}

export interface PresetDef {
  id: string;
  nameKey: string;
  descKey: string;
  tier: RuleTier;
  target: string;
  appMode?: AppMatchMode;
  apps?: string[];
  androidApps?: string[];
  domains?: string[];
  suffixes?: string[];
  lists?: string[];
  cidrs?: string[];
  ipPools?: string[];
  network?: TransportNetwork;
  ports?: string[];
  l7?: string[];
}

export interface StickyHop {
  nodeId: string;
  label: string;
  proto: string;
  failed: boolean;
}

export interface RuleEvalResult {
  matchedRule: RoutingRule | null;
  reason: string | null;
}
