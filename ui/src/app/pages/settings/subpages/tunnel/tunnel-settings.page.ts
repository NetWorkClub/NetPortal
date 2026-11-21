import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-tunnel-settings-page',
  standalone: true,
  imports: [FormsModule, RouterLink, IconComponent, ToggleSwitchComponent, TranslatePipe],
  templateUrl: './tunnel-settings.page.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class TunnelSettingsPageComponent {
  readonly store = inject(NetPortalStateService);
  readonly tunImpl = signal<'system' | 'gvisor' | 'mixed'>('system');
  readonly proxyPort = signal(10808);
}
