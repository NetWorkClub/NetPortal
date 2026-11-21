import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-dns-settings-page',
  standalone: true,
  imports: [FormsModule, RouterLink, IconComponent, ToggleSwitchComponent, TranslatePipe],
  templateUrl: './dns-settings.page.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DnsSettingsPageComponent {
  readonly store = inject(NetPortalStateService);
  readonly dnsRemote = signal('https://dns.quad9.net/dns-query');
  readonly dnsDirect = signal('https://1.1.1.1/dns-query');
}
