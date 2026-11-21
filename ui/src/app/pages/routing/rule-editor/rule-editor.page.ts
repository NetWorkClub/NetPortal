import { ChangeDetectionStrategy, Component, computed, effect, inject, signal } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import {
  AppMatchMode,
  InboundFilter,
  IpVersionFilter,
  RoutingRule,
  RuleTier,
  TransportNetwork,
} from '@core/models';
import { I18nService } from '@core/services/i18n.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { PlatformService } from '@core/services/platform.service';
import { RuleEngineService } from '@core/services/rule-engine.service';
import { ToastService } from '@core/services/toast.service';
import { splitTokens } from '@core/utils/format.util';
import { IconComponent } from '@shared/components/icon/icon.component';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';
import { map } from 'rxjs';

const DESKTOP_APP_CHIPS = ['cursor.exe', 'claude', 'discord.exe', 'chrome', 'steam.exe'];
const ANDROID_APP_CHIPS = [
  'com.anthropic.claude',
  'com.openai.chatgpt',
  'com.google.android.youtube',
  'org.telegram.messenger',
  'ru.sberbankmobile',
];
const GEOIP_CHIPS = ['RU', 'CN', 'IR', 'BY', 'PRIVATE'];
const L7_CHIPS = ['http', 'tls', 'quic', 'bittorrent', 'dns', 'ssh'];

@Component({
  selector: 'np-rule-editor-page',
  standalone: true,
  imports: [FormsModule, RouterLink, IconComponent, ToggleSwitchComponent, TranslatePipe],
  templateUrl: './rule-editor.page.html',
  styleUrl: './rule-editor.page.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RuleEditorPageComponent {
  readonly store = inject(NetPortalStateService);
  readonly platformSvc = inject(PlatformService);
  private readonly ruleEngine = inject(RuleEngineService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly i18n = inject(I18nService);
  private readonly toast = inject(ToastService);

  readonly geoipChips = GEOIP_CHIPS;
  readonly l7Chips = L7_CHIPS;

  readonly ruleIdParam = toSignal(
    this.route.paramMap.pipe(map((pm) => pm.get('id'))),
    { initialValue: null }
  );

  readonly editingRuleId = computed(() => {
    const id = this.ruleIdParam();
    return id && id !== 'new' ? id : null;
  });

  // Form state signals
  readonly name = signal('');
  readonly tier = signal<RuleTier>('combined');
  readonly enabled = signal(true);
  readonly invert = signal(false);
  readonly continueToNetwork = signal(true);
  readonly appMode = signal<AppMatchMode>('include');
  readonly appsText = signal('');
  readonly otherPlatformApps = signal<string[]>([]);
  readonly appPathsText = signal('');
  readonly domainsText = signal('');
  readonly suffixesText = signal('');
  readonly keywordsText = signal('');
  readonly regex = signal('');
  readonly listsText = signal('');
  readonly cidrsText = signal('');
  readonly ipPoolsText = signal('');
  readonly selectedGeoip = signal<string[]>([]);
  readonly network = signal<TransportNetwork>('both');
  readonly portsText = signal('');
  readonly ipVer = signal<IpVersionFilter>('any');
  readonly inbound = signal<InboundFilter>('any');
  readonly selectedL7 = signal<string[]>([]);
  readonly target = signal('group:sticky-ai');

  readonly appPoolChips = computed(() =>
    this.platformSvc.platform() === 'android' ? ANDROID_APP_CHIPS : DESKTOP_APP_CHIPS
  );

  readonly activeAppsList = computed(() => splitTokens(this.appsText()));

  readonly regexValidation = computed<{ ok: boolean; error: string | null } | null>(() => {
    const raw = this.regex().trim();
    if (!raw) return null;
    try {
      new RegExp(raw);
      return { ok: true, error: null };
    } catch (err) {
      return { ok: false, error: err instanceof Error ? err.message : String(err) };
    }
  });

  readonly actionKind = computed<'node' | 'direct' | 'byedpi' | 'block'>(() => {
    const t = this.target();
    if (t.startsWith('group:') || t.startsWith('node:')) return 'node';
    if (t === 'direct' || t === 'byedpi' || t === 'block') return t;
    return 'node';
  });

  readonly conflictWarnings = computed(() => {
    const reState = this.regexValidation();
    const regexErr =
      reState && !reState.ok
        ? this.i18n.t('ruleEditor.regexInvalid', { err: reState.error ?? '' })
        : null;

    const draft = this.buildRuleObject();
    return this.ruleEngine.detectRuleWarnings(
      draft,
      this.store.activeProfile().rules,
      this.editingRuleId(),
      this.platformSvc.platform(),
      this.i18n.lang(),
      regexErr
    );
  });

  constructor() {
    effect(() => {
      const id = this.editingRuleId();
      const existing = id
        ? this.store.activeProfile().rules.find((r) => r.id === id) ?? null
        : null;

      if (existing) {
        const isAndroid = this.platformSvc.platform() === 'android';
        this.name.set(existing.name);
        this.tier.set(existing.tier);
        this.enabled.set(existing.enabled);
        this.invert.set(existing.invert);
        this.continueToNetwork.set(existing.continueToNetwork);
        this.appMode.set(existing.appMode);
        this.appsText.set(
          (isAndroid && existing.androidApps.length > 0 ? existing.androidApps : existing.apps).join(', ')
        );
        this.otherPlatformApps.set(isAndroid ? [...existing.apps] : [...existing.androidApps]);
        this.appPathsText.set(existing.appPaths.join(', '));
        this.domainsText.set(existing.domains.join(', '));
        this.suffixesText.set(existing.suffixes.join(', '));
        this.keywordsText.set(existing.keywords.join(', '));
        this.regex.set(existing.regex);
        this.listsText.set(existing.lists.join(', '));
        this.cidrsText.set(existing.cidrs.join(', '));
        this.ipPoolsText.set(existing.ipPools.join(', '));
        this.selectedGeoip.set([...existing.geoip]);
        this.network.set(existing.network);
        this.portsText.set(existing.ports.join(', '));
        this.ipVer.set(existing.ipVer);
        this.inbound.set(existing.inbound);
        this.selectedL7.set([...existing.l7]);
        this.target.set(existing.target);
      } else {
        this.name.set('');
        this.tier.set('combined');
        this.enabled.set(true);
        this.invert.set(false);
        this.continueToNetwork.set(true);
        this.appMode.set('include');
        this.appsText.set('');
        this.otherPlatformApps.set([]);
        this.appPathsText.set('');
        this.domainsText.set('');
        this.suffixesText.set('');
        this.keywordsText.set('');
        this.regex.set('');
        this.listsText.set('');
        this.cidrsText.set('');
        this.ipPoolsText.set('');
        this.selectedGeoip.set([]);
        this.network.set('both');
        this.portsText.set('');
        this.ipVer.set('any');
        this.inbound.set('any');
        this.selectedL7.set([]);
        this.target.set('group:sticky-ai');
      }
    });
  }

  onTierChange(next: RuleTier): void {
    this.tier.set(next);
    if (next === 'network') {
      this.appMode.set('off');
    } else if (this.appMode() === 'off') {
      this.appMode.set('include');
    }
  }

  hasAppChip(chip: string): boolean {
    const low = chip.toLowerCase();
    return this.activeAppsList().some((a) => a.toLowerCase() === low);
  }

  toggleAppChip(chip: string): void {
    const cur = this.activeAppsList();
    const low = chip.toLowerCase();
    const next = cur.some((a) => a.toLowerCase() === low)
      ? cur.filter((a) => a.toLowerCase() !== low)
      : [...cur, chip];
    this.appsText.set(next.join(', '));
  }

  hasGeoip(code: string): boolean {
    return this.selectedGeoip().includes(code);
  }

  toggleGeoip(code: string): void {
    this.selectedGeoip.update((list) =>
      list.includes(code) ? list.filter((x) => x !== code) : [...list, code]
    );
  }

  hasL7(proto: string): boolean {
    return this.selectedL7().includes(proto);
  }

  toggleL7(proto: string): void {
    this.selectedL7.update((list) =>
      list.includes(proto) ? list.filter((x) => x !== proto) : [...list, proto]
    );
  }

  pickActionKind(kind: 'node' | 'direct' | 'byedpi' | 'block'): void {
    if (kind === 'node') {
      if (!this.target().startsWith('group:') && !this.target().startsWith('node:')) {
        this.target.set('group:sticky-ai');
      }
    } else {
      this.target.set(kind);
    }
  }

  save(): void {
    const reState = this.regexValidation();
    if (reState && !reState.ok) {
      this.toast.show(this.i18n.t('ruleEditor.regexInvalid', { err: reState.error ?? '' }));
      return;
    }

    const rule = this.buildRuleObject();
    this.store.saveRule(rule, this.editingRuleId());
    this.router.navigateByUrl('/routing');
  }

  private buildRuleObject(): RoutingRule {
    const isAndroid = this.platformSvc.platform() === 'android';
    const currentApps = this.activeAppsList();
    const cleanName =
      this.name().trim() || (this.i18n.lang() === 'en' ? 'New routing rule' : 'Новое правило');

    return {
      id: this.editingRuleId() ?? `rule-${Date.now()}`,
      name: cleanName,
      tier: this.tier(),
      enabled: this.enabled(),
      invert: this.invert(),
      continueToNetwork: this.continueToNetwork(),
      appMode: this.tier() === 'network' ? 'off' : this.appMode(),
      apps: isAndroid ? [...this.otherPlatformApps()] : [...currentApps],
      androidApps: isAndroid ? [...currentApps] : [...this.otherPlatformApps()],
      appPaths: splitTokens(this.appPathsText()),
      domains: splitTokens(this.domainsText()),
      suffixes: splitTokens(this.suffixesText()),
      keywords: splitTokens(this.keywordsText()),
      regex: this.regex().trim(),
      lists: splitTokens(this.listsText()),
      cidrs: splitTokens(this.cidrsText()),
      ipPools: splitTokens(this.ipPoolsText()),
      geoip: [...this.selectedGeoip()],
      network: this.network(),
      ports: splitTokens(this.portsText()),
      l7: [...this.selectedL7()],
      ipVer: this.ipVer(),
      inbound: this.inbound(),
      target: this.target(),
    };
  }
}
