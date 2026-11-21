import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { I18nService } from '@core/services/i18n.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { ThemeService } from '@core/services/theme.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-settings-menu-page',
  standalone: true,
  imports: [RouterLink, IconComponent, TranslatePipe],
  templateUrl: './settings-menu.page.html',
  styleUrl: './settings-menu.page.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SettingsMenuPageComponent {
  readonly store = inject(NetPortalStateService);
  private readonly themeSvc = inject(ThemeService);
  private readonly i18n = inject(I18nService);

  readonly generalSub = computed(() => {
    this.i18n.dict();
    const langLabel = this.i18n.lang() === 'en' ? 'English' : 'Русский';
    const themeKey = {
      dark: 'settings.themeDark',
      light: 'settings.themeLight',
      system: 'settings.themeSystem',
    }[this.themeSvc.theme()];
    return `${langLabel} · ${this.i18n.t(themeKey)}`;
  });

  readonly tunnelSub = computed(() => {
    this.i18n.dict();
    const svcKey = {
      vpn: 'settings.modeVpn',
      'system-proxy': 'settings.modeSysProxy',
      'local-proxy': 'settings.modeLocalProxy',
    }[this.store.serviceMode()];
    const muxState = this.i18n.t(this.store.mux() ? 'common.on' : 'common.off');
    return `${this.i18n.t(svcKey)} · MTU ${this.store.mtu()} · Mux ${muxState}`;
  });

  readonly obfSub = computed(() => {
    this.i18n.dict();
    const tls = this.i18n.t(this.store.tlsFrag() ? 'common.on' : 'common.off');
    const dpi = this.i18n.t(this.store.byedpiActive() ? 'common.on' : 'common.off');
    return `TLS ${tls} · AWG 2/3.1 · ByeDPI ${dpi}`;
  });
}
