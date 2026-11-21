import { ChangeDetectionStrategy, Component, inject, input, output } from '@angular/core';
import { RouterLink } from '@angular/router';
import { RoutingRule } from '@core/models';
import { I18nService } from '@core/services/i18n.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { PlatformService } from '@core/services/platform.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

interface MatcherBadge {
  text: string;
  title?: string;
  tone?: 'whitelist' | 'blacklist';
}

@Component({
  selector: 'np-rule-list-tier',
  standalone: true,
  imports: [RouterLink, IconComponent, ToggleSwitchComponent, TranslatePipe],
  templateUrl: './rule-list-tier.component.html',
  styleUrl: './rule-list-tier.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RuleListTierComponent {
  readonly store = inject(NetPortalStateService);
  readonly platformSvc = inject(PlatformService);
  private readonly i18n = inject(I18nService);

  readonly tierNumber = input.required<1 | 2>();
  readonly titleKey = input.required<string>();
  readonly hintText = input<string>('');
  readonly emptyKey = input.required<string>();
  readonly rules = input.required<RoutingRule[]>();
  readonly showCreateButton = input<boolean>(false);
  readonly createClicked = output<void>();

  targetBadgeClass(target: string): string {
    if (target === 'block') return 'danger';
    if (target === 'direct') return 'ok';
    return 'accent';
  }

  matcherChips(rule: RoutingRule): MatcherBadge[] {
    const chips: MatcherBadge[] = [];
    const activeApps =
      this.platformSvc.platform() === 'android' && rule.androidApps.length > 0
        ? rule.androidApps
        : rule.apps;

    if (rule.appMode === 'include' && activeApps.length > 0) {
      chips.push({
        text: `${this.i18n.t('routing.whitelistBadge')} | ${activeApps.length}`,
        title: activeApps.join(', '),
        tone: 'whitelist',
      });
    } else if (rule.appMode === 'exclude' && activeApps.length > 0) {
      chips.push({
        text: `${this.i18n.t('routing.blacklistBadge')} | ${activeApps.length}`,
        title: activeApps.join(', '),
        tone: 'blacklist',
      });
    }

    if (rule.appPaths.length > 0) chips.push({ text: `path | ${rule.appPaths.length}` });
    if (rule.domains.length > 0) chips.push({ text: `domain | ${rule.domains.length}` });
    if (rule.suffixes.length > 0) chips.push({ text: `domain_suffix | ${rule.suffixes.length}` });
    if (rule.keywords.length > 0) chips.push({ text: `keyword | ${rule.keywords.length}` });
    if (rule.regex) chips.push({ text: 'regex' });
    if (rule.lists.length > 0) chips.push({ text: `rule_set | ${rule.lists.length}` });
    if (rule.cidrs.length > 0) chips.push({ text: `ip_cidr | ${rule.cidrs.length}` });
    if (rule.ipPools.length > 0) chips.push({ text: `ip_pool | ${rule.ipPools.length}` });
    if (rule.geoip.length > 0) chips.push({ text: `geoip | ${rule.geoip.length}` });
    if (rule.ports.length > 0) {
      chips.push({ text: `${rule.network}:${rule.ports.join(',')}` });
    } else if (rule.network !== 'both') {
      chips.push({ text: rule.network.toUpperCase() });
    }
    if (rule.l7.length > 0) chips.push({ text: `l7 | ${rule.l7.join(',')}` });
    if (rule.invert) chips.push({ text: 'NOT (!)' });
    return chips;
  }
}
