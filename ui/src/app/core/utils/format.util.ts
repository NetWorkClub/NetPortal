import { ImportDetectionResult } from '@core/models';

export function formatDuration(sec: number): string {
  const h = String(Math.floor(sec / 3600)).padStart(2, '0');
  const m = String(Math.floor((sec % 3600) / 60)).padStart(2, '0');
  const s = String(sec % 60).padStart(2, '0');
  return `${h}:${m}:${s}`;
}

export function formatSpeed(mbps: number, lang: 'ru' | 'en'): string {
  const unit = lang === 'en' ? 'Mbps' : 'Мбит/с';
  return `${mbps.toFixed(1)} ${unit}`;
}

export function getLatencyClass(ms: number | null | undefined): 'ok' | 'warn' | 'danger' | 'muted' {
  if (ms === null || ms === undefined) return 'muted';
  if (ms < 80) return 'ok';
  if (ms <= 180) return 'warn';
  return 'danger';
}

export function splitTokens(val: string): string[] {
  return String(val || '')
    .split(/[\n,]+/)
    .map((s) => s.trim())
    .filter(Boolean);
}

export function nowTime(): string {
  const d = new Date();
  return [d.getHours(), d.getMinutes(), d.getSeconds()]
    .map((n) => String(n).padStart(2, '0'))
    .join(':');
}

export function detectImportFormat(raw: string, fileName = ''): ImportDetectionResult {
  const text = raw.trim();
  const ext = fileName.toLowerCase();

  if (/^vless:\/\//im.test(text)) return { proto: 'VLESS', protoGroup: 'proxy', label: 'VLESS URI / Subscription' };
  if (/^(hy2|hysteria2):\/\//im.test(text)) return { proto: 'HY2', protoGroup: 'proxy', label: 'Hysteria2 URI' };
  if (/^masque:\/\//im.test(text)) return { proto: 'MASQUE', protoGroup: 'masque', label: 'MASQUE RFC 9484' };
  if (/^trojan:\/\//im.test(text) || /^ss:\/\//im.test(text)) {
    return { proto: 'VLESS', protoGroup: 'proxy', label: 'Proxy URI List' };
  }
  if ((text.startsWith('{') || ext.endsWith('.json')) && /"outbounds"/i.test(text)) {
    return { proto: 'VLESS', protoGroup: 'proxy', label: 'sing-box JSON config' };
  }
  if (/^proxies\s*:/im.test(text) || ext.endsWith('.yaml') || ext.endsWith('.yml')) {
    return { proto: 'VLESS', protoGroup: 'proxy', label: 'Clash / Mihomo YAML' };
  }
  if (/RandomTrailers|DisableCookies|S3\s*=|S4\s*=/i.test(text)) {
    return { proto: 'AWG 3.1', protoGroup: 'awg', label: 'AmneziaWG 3.1 (.conf)' };
  }
  if (/HeaderProtectionKey|ContentPaddingAddition/i.test(text)) {
    return { proto: 'AWG 3.0', protoGroup: 'awg', label: 'AmneziaWG 3.0 (.conf)' };
  }
  if (/\[Interface\]/i.test(text) && /Jc\s*=/i.test(text)) {
    return { proto: 'AWG 2.0', protoGroup: 'awg', label: 'AmneziaWG 2.0 (.conf)' };
  }
  if (/\[Interface\]/i.test(text)) {
    return { proto: 'AWG 2.0', protoGroup: 'awg', label: 'WireGuard (.conf)' };
  }
  if (/^[A-Za-z0-9+/=_-]{32,}$/.test(text.replace(/\s+/g, ''))) {
    return { proto: 'VLESS', protoGroup: 'proxy', label: 'Base64 Subscription Bundle' };
  }
  return { proto: 'VLESS', protoGroup: 'proxy', label: 'URI / Subscription / Config' };
}
