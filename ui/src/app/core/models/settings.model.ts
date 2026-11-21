export type ThemeMode = 'dark' | 'light' | 'system';

export type AppLang = 'ru' | 'en';

export type BalancerStrategy = 'sticky' | 'lowest' | 'round-robin';

export type RoutingIpv6Mode = 'ipv4-only' | 'prefer-ipv6' | 'dual';

export interface ZapretStrategy {
  id: string;
  name: string;
  yt: string;
  quic: string;
  discord: string;
}

export interface ConflictAction {
  label: string;
  action: string;
  primary: boolean;
}

export interface ConflictReport {
  id: string;
  title: string;
  detail: string;
  actions: ConflictAction[];
}

export type LogLevel = 'info' | 'warn' | 'error';

export type LogCategory = 'core' | 'router' | 'probe' | 'dpi';

export interface LogEntry {
  time: string;
  lvl: LogLevel;
  cat: LogCategory;
  msg: string;
}
