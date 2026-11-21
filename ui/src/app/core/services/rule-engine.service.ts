import { Injectable } from '@angular/core';
import { AppLang, PlatformTarget, RoutingProfile, RoutingRule, RuleEvalResult } from '@core/models';

@Injectable({ providedIn: 'root' })
export class RuleEngineService {
  isTier1(rule: RoutingRule): boolean {
    return rule.tier === 'app' || rule.tier === 'combined' || rule.appMode !== 'off';
  }

  ipv4ToInt(ip: string): number | null {
    const m = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/.exec(ip.trim());
    if (!m) return null;
    const nums = m.slice(1).map(Number);
    if (nums.some((x) => x > 255)) return null;
    return (((nums[0] << 24) >>> 0) + (nums[1] << 16) + (nums[2] << 8) + nums[3]) >>> 0;
  }

  matchCidr(ipStr: string, cidr: string): boolean {
    const ipInt = this.ipv4ToInt(ipStr);
    if (ipInt === null) return false;
    const parts = cidr.trim().split('/');
    const netInt = this.ipv4ToInt(parts[0]);
    if (netInt === null) return false;
    const prefix = parts.length > 1 ? Number(parts[1]) : 32;
    if (Number.isNaN(prefix) || prefix < 0 || prefix > 32) return false;
    const mask = prefix === 0 ? 0 : (0xffffffff << (32 - prefix)) >>> 0;
    return (ipInt & mask) === (netInt & mask);
  }

  matchIpPool(ipStr: string, pool: string): boolean {
    const ipInt = this.ipv4ToInt(ipStr);
    if (ipInt === null) return false;
    const parts = pool.split('-').map((s) => s.trim());
    if (parts.length !== 2) return false;
    const a = this.ipv4ToInt(parts[0]);
    const b = this.ipv4ToInt(parts[1]);
    if (a === null || b === null) return false;
    return ipInt >= Math.min(a, b) && ipInt <= Math.max(a, b);
  }

  isAppQuery(q: string): boolean {
    if (/^(com|org|ru|io|net)\.[a-z0-9_.-]{2,}$/i.test(q) && !/\.(com|org|ru|io|net|ai)$/i.test(q)) {
      return true;
    }
    if (/\.exe$/i.test(q) || q.includes('/') || q.includes('\\')) return true;
    return ['claude', 'cursor', 'chrome', 'discord', 'telegram', 'steam'].includes(q);
  }

  matchRuleQuery(rule: RoutingRule, rawQuery: string, platform: PlatformTarget): string | null {
    const q = rawQuery.trim().toLowerCase();
    if (!q) return null;

    const appQuery = this.isAppQuery(q);
    const activeApps =
      platform === 'android' && rule.androidApps.length > 0 ? rule.androidApps : rule.apps;
    const allAppTokens = [...activeApps, ...rule.appPaths].map((s) => s.toLowerCase());

    let matchedReason: string | null = null;

    if (appQuery && rule.appMode !== 'off' && allAppTokens.length > 0) {
      const inList = allAppTokens.some((tok) => q === tok || q.includes(tok) || tok.includes(q));
      if (rule.appMode === 'include' && inList) {
        matchedReason = `${platform === 'android' ? 'pkg whitelist' : 'process whitelist'}: ${q}`;
      } else if (rule.appMode === 'exclude') {
        if (inList) return null;
        if (rule.tier === 'app') {
          matchedReason = `blacklist bypass: ${q}`;
        }
      }
    }

    if (!matchedReason && !appQuery) {
      for (const d of rule.domains) {
        if (q === d.toLowerCase()) {
          matchedReason = `domain: ${d}`;
          break;
        }
      }
      if (!matchedReason) {
        for (const s of rule.suffixes) {
          const clean = s.trim().toLowerCase().replace(/^\./, '');
          if (clean && (q === clean || q.endsWith('.' + clean))) {
            matchedReason = `domain_suffix: ${s}`;
            break;
          }
        }
      }
      if (!matchedReason) {
        for (const kw of rule.keywords) {
          if (kw && q.includes(kw.toLowerCase())) {
            matchedReason = `domain_keyword: ${kw}`;
            break;
          }
        }
      }
      if (!matchedReason && rule.regex) {
        try {
          if (new RegExp(rule.regex, 'i').test(q)) {
            matchedReason = `domain_regex: /${rule.regex}/`;
          }
        } catch {
          // Malformed regex handled by editor validation
        }
      }
      if (!matchedReason) {
        for (const lst of rule.lists) {
          if (q.includes('youtube') && lst.toLowerCase().includes('youtube')) {
            matchedReason = `rule_set: ${lst}`;
            break;
          }
          if ((q.includes('ads') || q.includes('doubleclick')) && lst.toLowerCase().includes('ads')) {
            matchedReason = `rule_set: ${lst}`;
            break;
          }
        }
      }
      if (!matchedReason && this.ipv4ToInt(q) !== null) {
        for (const c of rule.cidrs) {
          if (this.matchCidr(q, c)) {
            matchedReason = `ip_cidr: ${c}`;
            break;
          }
        }
        if (!matchedReason) {
          for (const pool of rule.ipPools) {
            if (this.matchIpPool(q, pool)) {
              matchedReason = `ip_pool: ${pool}`;
              break;
            }
          }
        }
      }
      if (!matchedReason) {
        const portMatch = /(?:^(?:tcp|udp):(\d+)$)|(?:^:(\d+)$)/.exec(q);
        if (portMatch) {
          const portNum = portMatch[1] || portMatch[2];
          if (rule.ports.includes(portNum)) matchedReason = `port: ${portNum}`;
        } else if (rule.l7.includes(q)) {
          matchedReason = `sniff: ${q}`;
        }
      }
    }

    if (rule.invert) {
      return matchedReason ? null : 'NOT matched (!invert)';
    }
    return matchedReason;
  }

  evaluateQuery(rawQuery: string, profile: RoutingProfile, platform: PlatformTarget): RuleEvalResult {
    const query = rawQuery.trim();
    if (!query) return { matchedRule: null, reason: null };

    const ordered = [
      ...profile.rules.filter((r) => this.isTier1(r)),
      ...profile.rules.filter((r) => !this.isTier1(r)),
    ];

    for (const r of ordered) {
      if (!r.enabled) continue;
      const reason = this.matchRuleQuery(r, query, platform);
      if (reason) return { matchedRule: r, reason };
    }
    return { matchedRule: null, reason: null };
  }

  detectRuleWarnings(
    draft: RoutingRule,
    existingRules: RoutingRule[],
    editingId: string | null,
    platform: PlatformTarget,
    lang: AppLang,
    regexError: string | null
  ): string[] {
    const warnings: string[] = [];
    if (regexError) warnings.push(regexError);

    const others = existingRules.filter((r) => r.enabled && r.id !== editingId);
    const activeApps = platform === 'android' && draft.androidApps.length > 0 ? draft.androidApps : draft.apps;

    if (draft.appMode !== 'off' && activeApps.length > 0) {
      for (const r of others) {
        if (!this.isTier1(r)) continue;
        const rApps = platform === 'android' && r.androidApps.length > 0 ? r.androidApps : r.apps;
        const overlap = activeApps.filter((a) => rApps.includes(a));
        if (overlap.length > 0) {
          warnings.push(
            lang === 'en'
              ? `Tier 1 app overlap with "${r.name}" (${overlap.join(', ')}). Higher rule executes first.`
              : `Пересечение Tier 1 по приложениям с «${r.name}» (${overlap.join(', ')}). Вышестоящее правило сработает первым.`
          );
          break;
        }
      }
    }

    if (draft.tier === 'network' && (draft.suffixes.length > 0 || draft.domains.length > 0)) {
      const shadow = others.find(
        (r) => this.isTier1(r) && !r.continueToNetwork && r.suffixes.some((s) => draft.suffixes.includes(s))
      );
      if (shadow) {
        warnings.push(
          lang === 'en'
            ? `Tier 1 rule "${shadow.name}" already captures matching suffixes before Tier 2.`
            : `Правило Tier 1 «${shadow.name}» уже перехватывает указанные суффиксы до перехода в Tier 2.`
        );
      }
    }

    return warnings;
  }
}
