import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { HomeMethod, NovpnMode, SignalHop } from '@core/models';
import { I18nService } from '@core/services/i18n.service';
import { ModalService } from '@core/services/modal.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { PlatformService } from '@core/services/platform.service';
import { formatDuration, formatSpeed } from '@core/utils/format.util';
import { ConflictBannerComponent } from '@shared/components/conflict-banner/conflict-banner.component';
import { IconComponent } from '@shared/components/icon/icon.component';
import { SignalStripComponent } from '@shared/components/signal-strip/signal-strip.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-home-page',
  standalone: true,
  imports: [FormsModule, RouterLink, IconComponent, ConflictBannerComponent, SignalStripComponent, TranslatePipe],
  templateUrl: './home.page.html',
  styleUrl: './home.page.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HomePageComponent {
  readonly store = inject(NetPortalStateService);
  readonly modals = inject(ModalService);
  readonly platformSvc = inject(PlatformService);
  private readonly i18n = inject(I18nService);

  readonly methodTabs: Array<{ id: HomeMethod; key: string }> = [
    { id: 'all', key: 'home.methodAll' },
    { id: 'awg', key: 'home.methodAwg' },
    { id: 'proxy', key: 'home.methodProxy' },
    { id: 'masque', key: 'home.methodMasque' },
    { id: 'psiphon', key: 'home.methodPsiphon' },
    { id: 'novpn', key: 'home.methodNovpn' },
  ];

  readonly novpnModes = computed<NovpnMode[]>(() =>
    this.platformSvc.platform() === 'desktop'
      ? ['byedpi', 'zapret', 'smartdns', 'byedpi_smartdns']
      : ['byedpi', 'smartdns', 'byedpi_smartdns']
  );

  readonly isFailover = computed(() => {
    const chain = this.store.stickyChain();
    return Boolean(chain[0]?.failed) && this.store.activeProfile().rules.length > 0 && this.store.homeMethod() !== 'novpn';
  });

  readonly svcModeLabel = computed(() => {
    this.i18n.dict();
    const map = {
      vpn: 'settings.modeVpn',
      'system-proxy': 'settings.modeSysProxy',
      'local-proxy': 'settings.modeLocalProxy',
    } as const;
    return this.i18n.t(map[this.store.serviceMode()]);
  });

  readonly profileName = computed(() => {
    const p = this.store.activeProfile();
    return this.i18n.lang() === 'en' ? p.nameEn : p.nameRu;
  });

  readonly mainCaption = computed(() => {
    this.i18n.dict();
    const st = this.store.connStatus();
    if (st === 'connecting') return this.i18n.t('home.connecting');
    if (st === 'connected') {
      if (this.store.homeMethod() === 'novpn') {
        return this.i18n.t('home.connectedNovpn', {
          name: this.i18n.t(`home.novpnModes.${this.store.novpnMode()}`),
        });
      }
      if (this.isFailover()) {
        return this.i18n.t('home.failoverBanner', { name: this.store.activeStickyHop().label });
      }
      return this.i18n.t('home.connectedNode', { name: this.store.activeNode().name });
    }
    return this.i18n.t('home.tapToConnect');
  });

  readonly subCaption = computed(() => {
    this.i18n.dict();
    const st = this.store.connStatus();
    const isNovpn = this.store.homeMethod() === 'novpn';
    if (st === 'connecting') {
      return this.i18n.t('home.connectingSub', {
        mode: isNovpn ? this.i18n.t('home.standalone') : this.svcModeLabel(),
      });
    }
    if (st === 'connected') {
      if (isNovpn) {
        return `${formatDuration(this.store.uptimeSec())} · ${this.i18n.t('common.direct')} + DPI/DNS`;
      }
      return `${formatDuration(this.store.uptimeSec())} · ${formatSpeed(this.store.rxMbps(), this.i18n.lang())} / ${formatSpeed(this.store.txMbps(), this.i18n.lang())}`;
    }
    if (isNovpn) {
      return this.i18n.t('home.disconnectedNovpnSub', {
        name: this.i18n.t(`home.novpnModes.${this.store.novpnMode()}`),
      });
    }
    const n = this.store.activeNode();
    const ping = n.urlMs ?? n.tcpMs;
    return this.i18n.t('home.disconnectedSub', { ping: ping !== null ? `${ping} ms` : '--' });
  });

  readonly signalHops = computed<SignalHop[]>(() => {
    this.i18n.dict();
    const devLabel = this.platformSvc.platform() === 'android' ? 'Android' : this.i18n.t('home.device');
    const st = this.store.connStatus();

    if (this.store.homeMethod() === 'novpn') {
      const novpnTitle = this.i18n.t(`home.novpnModes.${this.store.novpnMode()}`);
      return [
        { title: devLabel, meta: this.i18n.t('home.directExit') },
        {
          title: novpnTitle,
          meta: st === 'connected' ? this.i18n.t('common.active') : this.i18n.t('home.novpnReady'),
          tone: st === 'connected' ? 'active' : 'default',
        },
        { title: this.i18n.t('home.directExit'), meta: this.i18n.t('home.vpnOff') },
      ];
    }

    const hops: SignalHop[] = [
      { title: devLabel, meta: st === 'connected' ? this.svcModeLabel() : this.i18n.t('common.direct') },
    ];

    if (st === 'connected') {
      const dpi: string[] = [];
      if (this.store.tlsFrag()) dpi.push('TLS');
      if (this.store.byedpiActive()) dpi.push('ByeDPI');
      if (this.store.zapretActive() && this.platformSvc.platform() === 'desktop') dpi.push('Zapret');
      if (dpi.length > 0) {
        hops.push({ title: dpi.join(' + '), meta: this.i18n.t('home.dpiLayer'), tone: 'active' });
      }
      if (this.isFailover()) {
        hops.push({ title: this.store.stickyChain()[0].label, meta: this.i18n.t('common.failed'), tone: 'failed' });
        hops.push({ title: this.store.activeStickyHop().label, meta: 'Failover', tone: 'failover' });
      } else {
        const n = this.store.activeNode();
        const ping = n.urlMs ?? n.tcpMs;
        hops.push({
          title: n.name,
          meta: `${n.proto}${ping !== null ? ` · ${ping} ms` : ''}`,
          tone: 'active',
        });
      }
      const cascade = this.store.chainCascade();
      if (cascade !== 'none') {
        hops.push({
          title: cascade === 'psiphon' ? 'Psiphon DE' : 'WARP MASQUE',
          meta: this.i18n.t('home.cascade'),
          tone: 'active',
        });
      }
    } else {
      const stand: string[] = [];
      if (this.store.byedpiActive()) stand.push('ByeDPI');
      if (this.store.zapretActive() && this.platformSvc.platform() === 'desktop') stand.push('Zapret');
      if (this.store.smartDnsActive()) {
        stand.push(this.store.smartDnsMode() === 'hosts' ? 'Smart Hosts' : 'Smart DNS');
      }
      if (stand.length > 0) {
        hops.push({ title: stand.join(' + '), meta: this.i18n.t('home.standalone'), tone: 'active' });
      }
      hops.push({ title: this.i18n.t('home.directExit'), meta: this.i18n.t('home.vpnOff') });
    }
    return hops;
  });

  onTargetChange(val: string): void {
    if (this.store.homeMethod() === 'novpn') {
      this.store.applyStandaloneMode(val as NovpnMode);
      this.store.pushLog('info', 'dpi', `Standalone bypass engine switched to ${val}`);
    } else {
      this.store.selectedNodeId.set(val);
      this.store.pushLog('info', 'core', `Active node changed to ${this.store.activeNode().name}`);
    }
  }

  nodePingText(n: { urlMs: number | null; tcpMs: number | null }): string {
    if (n.urlMs !== null) return `${n.urlMs} ms`;
    if (n.tcpMs !== null) return `${n.tcpMs} ms TCP`;
    return this.i18n.t('common.notTested');
  }
}
