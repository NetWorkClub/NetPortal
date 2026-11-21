import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { I18nService } from '@core/services/i18n.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { PlatformService } from '@core/services/platform.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-sticky-chain',
  standalone: true,
  imports: [FormsModule, IconComponent, ToggleSwitchComponent, TranslatePipe],
  templateUrl: './sticky-chain.component.html',
  styleUrl: './sticky-chain.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class StickyChainComponent {
  readonly store = inject(NetPortalStateService);
  readonly platformSvc = inject(PlatformService);
  private readonly i18n = inject(I18nService);

  readonly entryBadgeKey = computed(() => {
    const map = {
      vpn: 'settings.modeVpn',
      'system-proxy': 'settings.modeSysProxy',
      'local-proxy': 'settings.modeLocalProxy',
    } as const;
    return map[this.store.serviceMode()];
  });

  readonly dpiLayer = computed<'none' | 'tls' | 'byedpi' | 'zapret'>(() => {
    if (this.store.zapretActive() && this.platformSvc.platform() === 'desktop') return 'zapret';
    if (this.store.byedpiActive()) return 'byedpi';
    if (this.store.tlsFrag()) return 'tls';
    return 'none';
  });

  readonly egressSummary = computed(() => {
    this.i18n.dict();
    const hop = this.store.activeStickyHop();
    const cascade = this.store.chainCascade();
    if (cascade === 'none') {
      return this.i18n.t('routing.chainEgressDefault', { node: hop.label });
    }
    return this.i18n.t('routing.chainEgressCascade', {
      cascade: cascade === 'psiphon' ? 'Psiphon DE' : 'WARP MASQUE',
      node: hop.label,
    });
  });

  protoBadgeClass(proto: string): string {
    const map: Record<string, string> = {
      VLESS: 'info',
      PSIPHON: 'ok',
      WARP: 'warn',
      MASQUE: 'masque',
    };
    return map[proto] ?? '';
  }
}
