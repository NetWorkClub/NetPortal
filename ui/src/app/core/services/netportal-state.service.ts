import { DestroyRef, Injectable, computed, effect, inject, signal } from '@angular/core';
import {
  BalancerStrategy,
  ChainCascade,
  ConflictReport,
  ConnectionStatus,
  HomeMethod,
  LogEntry,
  NodeItem,
  NovpnMode,
  PresetDef,
  RoutingIpv6Mode,
  RoutingProfile,
  RoutingRule,
  ServiceMode,
  SmartDnsMode,
  SourceItem,
  StickyHop,
  WarpEndpoint,
  ZapretStrategy,
} from '@core/models';
import { NETPORTAL_TRANSPORT } from '@core/transport/transport.token';
import { detectImportFormat, nowTime } from '@core/utils/format.util';
import {
  INITIAL_LOGS,
  INITIAL_NODES,
  INITIAL_PROFILES,
  INITIAL_SOURCES,
  INITIAL_STICKY_CHAIN,
  OPT_IN_PRESETS,
  WARP_ENDPOINTS,
  ZAPRET_STRATEGIES,
} from '@core/utils/initial-data';
import { I18nService } from './i18n.service';
import { PlatformService } from './platform.service';
import { RuleEngineService } from './rule-engine.service';
import { ToastService } from './toast.service';

@Injectable({ providedIn: 'root' })
export class NetPortalStateService {
  private readonly transport = inject(NETPORTAL_TRANSPORT);
  private readonly platformSvc = inject(PlatformService);
  private readonly i18n = inject(I18nService);
  private readonly toast = inject(ToastService);
  private readonly ruleEngine = inject(RuleEngineService);
  private readonly destroyRef = inject(DestroyRef);

  private uptimeTimer: ReturnType<typeof setInterval> | null = null;

  // Connection & telemetry
  readonly connStatus = signal<ConnectionStatus>('disconnected');
  readonly uptimeSec = signal(0);
  readonly rxMbps = signal(0);
  readonly txMbps = signal(0);
  readonly homeMethod = signal<HomeMethod>('all');
  readonly novpnMode = signal<NovpnMode>('byedpi');

  // Core VPN / DPI / DNS switches
  readonly serviceMode = signal<ServiceMode>('vpn');
  readonly mtu = signal(1280);
  readonly mux = signal(false);
  readonly strictRoute = signal(true);
  readonly balancerStrategy = signal<BalancerStrategy>('sticky');
  readonly sniffing = signal(true);
  readonly routingIpv6 = signal<RoutingIpv6Mode>('ipv4-only');
  readonly tlsFrag = signal(false);
  readonly tlsSniCase = signal(false);
  readonly tlsPadding = signal(false);
  readonly fakeIp = signal(false);
  readonly byedpiActive = signal(false);
  readonly zapretActive = signal(false);
  readonly smartDnsActive = signal(false);
  readonly smartDnsMode = signal<SmartDnsMode>('proxy');
  readonly chainCascade = signal<ChainCascade>('none');

  // General & additional settings toggles
  readonly ipCheckOnConnect = signal(true);
  readonly closeToTray = signal(true);
  readonly autostart = signal(false);
  readonly memLimit = signal(false);
  readonly perAppMode = signal<'all' | 'include' | 'exclude'>('all');
  readonly keepProxy = signal(false);
  readonly proxyAuth = signal(false);
  readonly lanShare = signal(false);
  readonly awg31Default = signal(true);
  readonly byedpiPreset = signal<'default' | 'mobile' | 'aggressive'>('default');
  readonly byedpiFlags = signal('--split 1 --disorder 3+s --tlsrec 1+s --port 10801');
  readonly byedpiTestStatus = signal<string | null>(null);

  readonly zapretSelected = signal('strat-1');
  readonly zapretStrategies = signal<ZapretStrategy[]>(ZAPRET_STRATEGIES.map((s) => ({ ...s })));

  // Nodes & Sources
  readonly sources = signal<SourceItem[]>(INITIAL_SOURCES.map((s) => ({ ...s })));
  readonly nodes = signal<NodeItem[]>(INITIAL_NODES.map((n) => ({ ...n, params: { ...n.params } })));
  readonly activeSource = signal<string>('all');
  readonly protoFilter = signal<string>('all');
  readonly searchQuery = signal('');
  readonly sortBy = signal<'url' | 'tcp' | 'name' | 'proto'>('url');
  readonly selectedNodeId = signal<string>('vless-de');
  readonly checkedNodes = signal<Set<string>>(new Set());

  // Routing & Sticky Chain
  readonly activeProfileId = signal<string>('hybrid');
  readonly profiles = signal<RoutingProfile[]>(
    INITIAL_PROFILES.map((p) => ({ ...p, rules: p.rules.map((r) => ({ ...r })) }))
  );
  readonly presets: PresetDef[] = OPT_IN_PRESETS;
  readonly lastHitRuleId = signal<string | null>(null);
  readonly stickyChain = signal<StickyHop[]>(INITIAL_STICKY_CHAIN.map((h) => ({ ...h })));
  readonly stickyStickBack = signal(true);

  // Generators
  readonly warpFormat = signal<'awg31' | 'awg30' | 'awg20' | 'masque'>('awg31');
  readonly warpEndpoints = signal<WarpEndpoint[]>(WARP_ENDPOINTS.map((e) => ({ ...e })));
  readonly warpSelectedEndpoint = signal<string>(WARP_ENDPOINTS[0].addr);
  readonly warpTrailers = signal(true);

  // Logs
  readonly logsPaused = signal(false);
  readonly logSearch = signal('');
  readonly logCat = signal('all');
  readonly logLevel = signal('all');
  readonly logs = signal<LogEntry[]>([...INITIAL_LOGS]);

  // Derived selectors
  readonly activeProfile = computed(
    () => this.profiles().find((p) => p.id === this.activeProfileId()) ?? this.profiles()[0]
  );

  readonly activeNode = computed(
    () => this.nodes().find((n) => n.id === this.selectedNodeId()) ?? this.nodes()[0]
  );

  readonly activeStickyHop = computed(
    () => this.stickyChain().find((h) => !h.failed) ?? this.stickyChain()[0]
  );

  readonly homeFilteredNodes = computed(() => {
    const m = this.homeMethod();
    const list = this.nodes();
    if (m === 'awg') return list.filter((n) => n.protoGroup === 'awg' || n.protoGroup === 'warp');
    if (m === 'proxy') return list.filter((n) => n.protoGroup === 'proxy');
    if (m === 'masque') return list.filter((n) => n.protoGroup === 'masque');
    if (m === 'psiphon') return list.filter((n) => n.protoGroup === 'psiphon');
    return list;
  });

  readonly filteredNodes = computed(() => {
    const src = this.activeSource();
    const proto = this.protoFilter();
    const q = this.searchQuery().trim().toLowerCase();
    const sort = this.sortBy();

    const list = this.nodes().filter((n) => {
      if (src !== 'all' && n.sourceId !== src) return false;
      if (proto !== 'all' && n.protoGroup !== proto) return false;
      if (q && !`${n.name} ${n.endpoint} ${n.proto} ${n.iso}`.toLowerCase().includes(q)) return false;
      return true;
    });

    return [...list].sort((a, b) => {
      if (sort === 'url') return (a.urlMs ?? 99999) - (b.urlMs ?? 99999);
      if (sort === 'tcp') return (a.tcpMs ?? 99999) - (b.tcpMs ?? 99999);
      if (sort === 'name') return a.name.localeCompare(b.name);
      return a.proto.localeCompare(b.proto);
    });
  });

  readonly filteredLogs = computed(() => {
    const q = this.logSearch().trim().toLowerCase();
    const cat = this.logCat();
    const lvl = this.logLevel();
    return this.logs().filter((l) => {
      if (cat !== 'all' && l.cat !== cat) return false;
      if (lvl !== 'all' && l.lvl !== lvl) return false;
      if (q && !`${l.msg} ${l.cat} ${l.lvl}`.toLowerCase().includes(q)) return false;
      return true;
    });
  });

  readonly conflicts = computed<ConflictReport[]>(() => {
    const platform = this.platformSvc.platform();
    const lang = this.i18n.lang();
    const status = this.connStatus();
    const tunActive =
      (status === 'connected' || status === 'connecting') &&
      this.serviceMode() === 'vpn' &&
      this.homeMethod() !== 'novpn';

    const list: ConflictReport[] = [];

    if (tunActive && this.zapretActive() && platform === 'desktop') {
      list.push({
        id: 'tun-zapret',
        title:
          lang === 'en'
            ? 'TUN and Zapret (WinDivert / NFQUEUE) conflict'
            : 'Конфликт TUN и Zapret (WinDivert / NFQUEUE)',
        detail:
          lang === 'en'
            ? 'Active TUN and kernel Zapret filter intercept traffic simultaneously, causing packet drops.'
            : 'Активный TUN и ядерный фильтр Zapret перехватывают трафик одновременно, что вызывает потери пакетов.',
        actions: [
          { label: lang === 'en' ? 'Disable Zapret' : 'Отключить Zapret', action: 'disable-zapret', primary: true },
          { label: lang === 'en' ? 'Switch to ByeDPI' : 'Заменить на ByeDPI', action: 'switch-to-byedpi', primary: false },
        ],
      });
    }

    if (tunActive && this.smartDnsActive() && this.smartDnsMode() === 'hosts' && platform === 'desktop') {
      list.push({
        id: 'tun-hosts',
        title:
          lang === 'en'
            ? 'TUN overrides system hosts file'
            : 'TUN перекрывает системный файл hosts',
        detail:
          lang === 'en'
            ? 'When VPN is active, DNS requests bypass local hosts records through the tunnel.'
            : 'При активном VPN DNS-запросы уходят в туннель в обход локальных записей hosts.',
        actions: [
          { label: lang === 'en' ? 'HTTP Proxy (:10809)' : 'Режим HTTP-прокси (:10809)', action: 'smartdns-proxy', primary: true },
          { label: lang === 'en' ? 'Disable Smart Hosts' : 'Отключить Smart Hosts', action: 'disable-smartdns', primary: false },
        ],
      });
    }

    if (this.fakeIp() && this.smartDnsActive()) {
      list.push({
        id: 'fakeip-smartdns',
        title: lang === 'en' ? 'FakeIP and Smart DNS conflict' : 'Конфликт FakeIP и Smart DNS',
        detail:
          lang === 'en'
            ? 'FakeIP returns synthetic addresses before resolution, shadowing Smart DNS rules.'
            : 'FakeIP возвращает синтетические адреса до резолва, перекрывая правила Smart DNS.',
        actions: [
          { label: lang === 'en' ? 'Disable FakeIP' : 'Отключить FakeIP', action: 'disable-fakeip', primary: true },
          { label: lang === 'en' ? 'Disable Smart DNS' : 'Отключить Smart DNS', action: 'disable-smartdns', primary: false },
        ],
      });
    }

    return list;
  });

  constructor() {
    effect(() => {
      if (this.platformSvc.platform() === 'android') {
        if (this.zapretActive()) this.zapretActive.set(false);
        if (this.smartDnsMode() === 'hosts') this.smartDnsMode.set('proxy');
        if (this.novpnMode() === 'zapret') this.novpnMode.set('byedpi');
      }
    });

    this.destroyRef.onDestroy(() => this.stopUptimeTimer());
  }

  getTargetLabel(targetKey: string): string {
    if (targetKey === 'direct') return this.i18n.t('common.direct');
    if (targetKey === 'block') return this.i18n.t('common.block');
    if (targetKey === 'byedpi') return this.i18n.t('common.byedpi');
    if (targetKey === 'group:sticky-ai') return this.i18n.t('common.stickyAi');
    if (targetKey === 'group:auto-fastest') {
      return this.i18n.lang() === 'en' ? 'Auto: Fastest URL' : 'Авто: Быстрый URL';
    }
    if (targetKey.startsWith('node:')) {
      const found = this.nodes().find((n) => n.id === targetKey.slice(5));
      if (found) return found.name;
    }
    return targetKey;
  }

  pushLog(lvl: LogEntry['lvl'], cat: LogEntry['cat'], msg: string): void {
    if (this.logsPaused()) return;
    this.logs.update((prev) => {
      const next = [...prev, { time: nowTime(), lvl, cat, msg }];
      return next.length > 1000 ? next.slice(next.length - 1000) : next;
    });
  }

  setHomeMethod(method: HomeMethod): void {
    this.homeMethod.set(method);
    if (method === 'novpn') {
      this.applyStandaloneMode(this.novpnMode());
    } else {
      const pool = this.homeFilteredNodes();
      if (pool.length > 0 && !pool.some((n) => n.id === this.selectedNodeId())) {
        this.selectedNodeId.set(pool[0].id);
      }
    }
  }

  applyStandaloneMode(mode: NovpnMode): void {
    const safeMode: NovpnMode =
      this.platformSvc.platform() === 'android' && mode === 'zapret' ? 'byedpi' : mode;
    this.novpnMode.set(safeMode);
    this.byedpiActive.set(safeMode === 'byedpi' || safeMode === 'byedpi_smartdns');
    this.zapretActive.set(safeMode === 'zapret' && this.platformSvc.platform() === 'desktop');
    this.smartDnsActive.set(safeMode === 'smartdns' || safeMode === 'byedpi_smartdns');
  }

  async toggleConnection(): Promise<void> {
    if (this.connStatus() === 'disconnected') {
      this.connStatus.set('connecting');
      const isNovpn = this.homeMethod() === 'novpn';
      if (isNovpn) {
        this.applyStandaloneMode(this.novpnMode());
        this.pushLog('info', 'dpi', `Starting standalone zero-VPN mode (${this.novpnMode()})...`);
      } else {
        this.pushLog('info', 'core', `Connecting via ${this.serviceMode()} (${this.activeNode().name})...`);
      }

      await this.transport.connect({
        mode: isNovpn ? 'standalone' : this.serviceMode(),
        targetName: isNovpn ? this.novpnMode() : this.activeNode().name,
        standalone: isNovpn,
      });

      if (this.connStatus() !== 'connecting') return;
      this.connStatus.set('connected');
      this.uptimeSec.set(1);
      if (isNovpn) {
        this.rxMbps.set(0);
        this.txMbps.set(0);
        this.pushLog('info', 'dpi', `Standalone bypass active: ${this.novpnMode()} (no VPN tunnel)`);
      } else {
        this.rxMbps.set(42.4);
        this.txMbps.set(8.7);
        this.pushLog('info', 'core', `Connected to ${this.activeNode().name} [${this.activeNode().endpoint}]`);
      }

      this.stopUptimeTimer();
      this.uptimeTimer = setInterval(() => {
        if (this.connStatus() !== 'connected') return;
        this.uptimeSec.update((s) => s + 1);
        if (this.homeMethod() !== 'novpn') {
          this.rxMbps.set(28 + Math.random() * 44);
          this.txMbps.set(5 + Math.random() * 11);
        }
      }, 1000);
    } else {
      await this.transport.disconnect();
      this.connStatus.set('disconnected');
      this.uptimeSec.set(0);
      this.rxMbps.set(0);
      this.txMbps.set(0);
      this.stopUptimeTimer();
      this.pushLog('info', 'core', 'Disconnected cleanly');
    }
  }

  selectNode(nodeId: string): void {
    this.selectedNodeId.set(nodeId);
    if (this.homeMethod() === 'novpn') this.homeMethod.set('all');
    const n = this.activeNode();
    this.pushLog('info', 'core', `Active node selected pre-connect: ${n.name} (${n.proto})`);
    this.toast.show(n.name);
  }

  toggleNodeCheck(nodeId: string, checked: boolean): void {
    this.checkedNodes.update((set) => {
      const next = new Set(set);
      if (checked) next.add(nodeId);
      else next.delete(nodeId);
      return next;
    });
  }

  toggleSelectAllVisible(checked: boolean): void {
    const visibleIds = this.filteredNodes().map((n) => n.id);
    this.checkedNodes.update((set) => {
      const next = new Set(set);
      for (const id of visibleIds) {
        if (checked) next.add(id);
        else next.delete(id);
      }
      return next;
    });
  }

  runProbe(kind: 'tcp' | 'url', forceAll = false): void {
    const pool = forceAll ? this.nodes() : this.filteredNodes();
    const checked = this.checkedNodes();
    const targets = !forceAll && checked.size > 0 ? pool.filter((n) => checked.has(n.id)) : pool;

    if (targets.length === 0) {
      this.toast.show(this.i18n.t('nodes.emptyText'));
      return;
    }

    const ids = new Set(targets.map((n) => n.id));
    this.nodes.update((list) => list.map((n) => (ids.has(n.id) ? { ...n, probing: true } : n)));
    this.pushLog('info', 'probe', `Starting ${kind.toUpperCase()} probe on ${targets.length} node(s) without TUN`);

    targets.forEach((node, idx) => {
      setTimeout(async () => {
        const outcome = await this.transport.probeNode(node, kind);
        this.nodes.update((list) =>
          list.map((n) =>
            n.id === outcome.nodeId
              ? { ...n, probing: false, tcpMs: outcome.tcpMs, urlMs: outcome.urlMs, antiBot: outcome.antiBot }
              : n
          )
        );
      }, 140 + idx * 90);
    });

    this.toast.show(`${kind.toUpperCase()} probe (${targets.length})`);
  }

  importRawNode(nameInput: string, raw: string, fileName = ''): void {
    const name = nameInput.trim() || 'Imported Node';
    const det = detectImportFormat(raw, fileName);
    const id = `custom-${Date.now()}`;
    const epMatch = raw.match(/Endpoint\s*=\s*([^\s\n]+)/i) || raw.match(/@([^\s/?#]+)/);
    const node: NodeItem = {
      id,
      sourceId: det.protoGroup === 'awg' ? 'awg-local' : 'sub-main',
      iso: 'NL',
      name,
      endpoint: epMatch ? epMatch[1] : '185.212.44.19:443',
      proto: det.proto,
      protoGroup: det.protoGroup,
      tcpMs: 41,
      urlMs: 83,
      antiBot: 'Clean',
      probing: false,
      mtu: this.mtu(),
      params: { Format: det.label, Source: 'Local DB' },
    };
    this.nodes.update((list) => [node, ...list]);
    this.selectedNodeId.set(id);
    this.pushLog('info', 'core', `Imported ${det.proto} node "${name}" into local inventory`);
    this.toast.show(name);
  }

  async scanWarpEndpoints(): Promise<void> {
    const sorted = await this.transport.scanWarpEndpoints(this.warpEndpoints());
    this.warpEndpoints.set(sorted);
    if (sorted.length > 0) {
      this.warpSelectedEndpoint.set(sorted[0].addr);
      this.pushLog('info', 'probe', `WARP endpoint scan best: ${sorted[0].addr} (${sorted[0].rtt} ms)`);
      this.toast.show(sorted[0].addr);
    }
  }

  generateWarpNode(mtuValue: number): void {
    const fmt = this.warpFormat();
    const fmtLabels: Record<string, string> = {
      awg31: 'AWG 3.1',
      awg30: 'AWG 3.0',
      awg20: 'AWG 2.0',
      masque: 'MASQUE',
    };
    const ep = this.warpSelectedEndpoint();
    const epShort = ep.split(':')[0];
    const bestRtt = this.warpEndpoints()[0]?.rtt ?? 34;
    const node: NodeItem = {
      id: `warp-${Date.now()}`,
      sourceId: 'warp-gen',
      iso: 'NL',
      name: `WARP ${fmtLabels[fmt]} [${epShort}]`,
      endpoint: ep,
      proto: fmt === 'masque' ? 'MASQUE' : 'WARP',
      protoGroup: fmt === 'masque' ? 'masque' : 'warp',
      tcpMs: bestRtt,
      urlMs: bestRtt + 42,
      antiBot: 'Challenge',
      probing: false,
      mtu: mtuValue || 1280,
      params: { Format: fmtLabels[fmt], RandomTrailers: this.warpTrailers() ? '1' : '0' },
    };
    this.nodes.update((list) => [node, ...list]);
    this.selectedNodeId.set(node.id);
    this.pushLog('info', 'core', `Generated WARP profile "${node.name}"`);
    this.toast.show(node.name);
  }

  addPsiphonNode(region: string, mode: 'node' | 'fallback'): void {
    const id = `psiphon-${region.toLowerCase()}-${Date.now()}`;
    const label = `Psiphon ${region}`;
    const node: NodeItem = {
      id,
      sourceId: 'psiphon-pool',
      iso: region,
      name: label,
      endpoint: `psiphon-tunnel-core (${region})`,
      proto: 'PSIPHON',
      protoGroup: 'psiphon',
      tcpMs: 54,
      urlMs: 118,
      antiBot: 'Clean',
      probing: false,
      mtu: 1280,
      params: { Region: region, LocalProxy: '127.0.0.1:10808' },
    };
    this.nodes.update((list) => [node, ...list]);
    if (mode === 'fallback') {
      this.stickyChain.update((c) => [...c, { nodeId: id, label, proto: 'PSIPHON', failed: false }]);
    }
    this.pushLog('info', 'core', `Added Psiphon region ${region} (${mode})`);
    this.toast.show(label);
  }

  updateNodeParams(nodeId: string, name: string, mtu: number, params: Record<string, string>): void {
    this.nodes.update((list) =>
      list.map((n) => (n.id === nodeId ? { ...n, name: name.trim() || n.name, mtu: mtu || n.mtu, params: { ...params } } : n))
    );
    this.pushLog('info', 'core', `Updated node parameters for "${name}" (MTU=${mtu})`);
    this.toast.show(name);
  }

  // Routing profile & rule mutations
  setDefaultTarget(target: string): void {
    const pid = this.activeProfileId();
    this.profiles.update((list) => list.map((p) => (p.id === pid ? { ...p, defaultTarget: target } : p)));
  }

  toggleRule(ruleId: string): void {
    const pid = this.activeProfileId();
    let changedName = '';
    let nextEnabled = false;
    this.profiles.update((list) =>
      list.map((p) =>
        p.id === pid
          ? {
              ...p,
              rules: p.rules.map((r) => {
                if (r.id !== ruleId) return r;
                changedName = r.name;
                nextEnabled = !r.enabled;
                return { ...r, enabled: nextEnabled };
              }),
            }
          : p
      )
    );
    if (changedName) {
      this.pushLog('info', 'router', `Rule "${changedName}" ${nextEnabled ? 'enabled' : 'disabled'}`);
    }
  }

  deleteRule(ruleId: string): void {
    const pid = this.activeProfileId();
    this.profiles.update((list) =>
      list.map((p) => (p.id === pid ? { ...p, rules: p.rules.filter((r) => r.id !== ruleId) } : p))
    );
    this.pushLog('info', 'router', 'Routing rule removed');
  }

  moveRuleInTier(ruleId: string, direction: -1 | 1): void {
    const pid = this.activeProfileId();
    this.profiles.update((list) =>
      list.map((p) => {
        if (p.id !== pid) return p;
        const rule = p.rules.find((r) => r.id === ruleId);
        if (!rule) return p;
        const isT1 = this.ruleEngine.isTier1(rule);
        const tierRules = p.rules.filter((r) => this.ruleEngine.isTier1(r) === isT1);
        const tIdx = tierRules.findIndex((r) => r.id === ruleId);
        const swapWith = tierRules[tIdx + direction];
        if (!swapWith) return p;

        const copy = [...p.rules];
        const a = copy.findIndex((r) => r.id === ruleId);
        const b = copy.findIndex((r) => r.id === swapWith.id);
        [copy[a], copy[b]] = [copy[b], copy[a]];
        return { ...p, rules: copy };
      })
    );
  }

  saveRule(rule: RoutingRule, editingId: string | null): void {
    const pid = this.activeProfileId();
    this.profiles.update((list) =>
      list.map((p) => {
        if (p.id !== pid) return p;
        if (editingId) {
          return {
            ...p,
            rules: p.rules.map((existing) => (existing.id === editingId ? rule : existing)),
          };
        }
        return { ...p, rules: [rule, ...p.rules] };
      })
    );
    this.pushLog('info', 'router', `Saved ${rule.tier} rule "${rule.name}" -> ${rule.target}`);
    this.toast.show(`${rule.name} (${this.getTargetLabel(rule.target)})`);
  }

  togglePreset(preset: PresetDef): void {
    const pid = this.activeProfileId();
    const existing = this.activeProfile().rules.some((r) => r.presetId === preset.id);
    this.profiles.update((list) =>
      list.map((p) => {
        if (p.id !== pid) return p;
        if (existing) {
          return { ...p, rules: p.rules.filter((r) => r.presetId !== preset.id) };
        }
        const created: RoutingRule = {
          id: `preset-${preset.id}-${Date.now()}`,
          presetId: preset.id,
          name: this.i18n.t(preset.nameKey),
          tier: preset.tier,
          enabled: true,
          invert: false,
          continueToNetwork: preset.tier !== 'network',
          appMode: preset.appMode ?? 'off',
          apps: [...(preset.apps ?? [])],
          androidApps: [...(preset.androidApps ?? [])],
          appPaths: [],
          domains: [...(preset.domains ?? [])],
          suffixes: [...(preset.suffixes ?? [])],
          keywords: [],
          regex: '',
          lists: [...(preset.lists ?? [])],
          cidrs: [...(preset.cidrs ?? [])],
          ipPools: [...(preset.ipPools ?? [])],
          geoip: [],
          network: preset.network ?? 'both',
          ports: [...(preset.ports ?? [])],
          l7: [...(preset.l7 ?? [])],
          ipVer: 'any',
          inbound: 'any',
          target: preset.target,
        };
        return { ...p, rules: [...p.rules, created] };
      })
    );
    this.pushLog('info', 'router', `Opt-in preset ${existing ? 'disabled' : 'enabled'}: ${preset.id}`);
  }

  moveStickyHop(index: number, direction: -1 | 1): void {
    this.stickyChain.update((chain) => {
      const target = index + direction;
      if (target < 0 || target >= chain.length) return chain;
      const copy = [...chain];
      [copy[index], copy[target]] = [copy[target], copy[index]];
      return copy;
    });
  }

  togglePrimaryFailover(): void {
    this.stickyChain.update((chain) => {
      if (chain.length === 0) return chain;
      const copy = [...chain];
      copy[0] = { ...copy[0], failed: !copy[0].failed };
      return copy;
    });
    const failed = this.stickyChain()[0]?.failed;
    const hop = this.activeStickyHop();
    this.pushLog(
      failed ? 'warn' : 'info',
      'router',
      failed
        ? `Healthcheck failed on ${this.stickyChain()[0].label}; switched to ${hop.label}`
        : `Primary hop ${this.stickyChain()[0].label} recovered`
    );
    this.toast.show(failed ? `Failover -> ${hop.label}` : hop.label);
  }

  setChainDpiLayer(v: 'none' | 'tls' | 'byedpi' | 'zapret'): void {
    this.tlsFrag.set(v === 'tls');
    this.byedpiActive.set(v === 'byedpi');
    this.zapretActive.set(v === 'zapret' && this.platformSvc.platform() === 'desktop');
    this.pushLog('info', 'dpi', `Chain DPI layer set to ${v}`);
  }

  resolveConflictAction(actionId: string): void {
    if (actionId === 'disable-zapret') {
      this.zapretActive.set(false);
      this.pushLog('info', 'dpi', 'Conflict resolved: Zapret disabled while TUN is active');
      this.toast.show(this.i18n.lang() === 'en' ? 'Zapret disabled.' : 'Zapret отключен.');
    } else if (actionId === 'switch-to-byedpi') {
      this.zapretActive.set(false);
      this.byedpiActive.set(true);
      this.pushLog('info', 'dpi', 'Conflict resolved: switched DPI layer to ByeDPI');
      this.toast.show(this.i18n.lang() === 'en' ? 'Switched to ByeDPI (:10801).' : 'Включен ByeDPI (:10801).');
    } else if (actionId === 'disable-fakeip') {
      this.fakeIp.set(false);
      this.pushLog('info', 'core', 'Conflict resolved: FakeIP disabled for Smart DNS');
      this.toast.show(this.i18n.lang() === 'en' ? 'FakeIP disabled.' : 'FakeIP отключен.');
    } else if (actionId === 'smartdns-proxy') {
      this.smartDnsMode.set('proxy');
      this.pushLog('info', 'core', 'Conflict resolved: Smart Hosts switched to HTTP proxy (:10809)');
      this.toast.show(this.i18n.lang() === 'en' ? 'Smart DNS set to :10809 proxy.' : 'Smart DNS переведен на прокси :10809.');
    } else if (actionId === 'disable-smartdns') {
      this.smartDnsActive.set(false);
      this.pushLog('info', 'core', 'Conflict resolved: Smart Hosts disabled');
      this.toast.show(this.i18n.lang() === 'en' ? 'Smart Hosts disabled.' : 'Smart Hosts отключен.');
    }
  }

  simulateConflict(): void {
    if (this.homeMethod() === 'novpn') this.homeMethod.set('all');
    this.serviceMode.set('vpn');
    this.connStatus.set('connected');
    if (this.platformSvc.platform() === 'desktop') {
      this.zapretActive.set(true);
      this.pushLog('error', 'core', 'ConflictGuard: TUN + Zapret active simultaneously');
    } else {
      this.fakeIp.set(true);
      this.smartDnsActive.set(true);
      this.pushLog('error', 'core', 'ConflictGuard: FakeIP conflicting with Smart DNS');
    }
    this.toast.show('ConflictGuard triggered');
  }

  runByedpiCheck(): void {
    this.byedpiTestStatus.set(this.i18n.t('common.probing'));
    setTimeout(() => {
      this.byedpiTestStatus.set('TLS 1.3 handshake OK (44 ms)');
      this.pushLog('info', 'dpi', 'ByeDPI local strategy check passed (44 ms)');
    }, 260);
  }

  runZapretBenchmark(): void {
    this.zapretStrategies.update((list) =>
      list.map((s, idx) => (idx === 1 ? { ...s, quic: Math.random() > 0.5 ? 'OK' : 'Timeout' } : s))
    );
    this.pushLog('info', 'dpi', 'Zapret benchmark finished across 4 strategies');
    this.toast.show('Zapret benchmark OK');
  }

  private stopUptimeTimer(): void {
    if (this.uptimeTimer !== null) {
      clearInterval(this.uptimeTimer);
      this.uptimeTimer = null;
    }
  }
}
