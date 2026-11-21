import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { I18nService } from '@core/services/i18n.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { ThemeService } from '@core/services/theme.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-general-settings-page',
  standalone: true,
  imports: [FormsModule, RouterLink, IconComponent, ToggleSwitchComponent, TranslatePipe],
  templateUrl: './general-settings.page.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GeneralSettingsPageComponent {
  readonly i18n = inject(I18nService);
  readonly themeSvc = inject(ThemeService);
  readonly store = inject(NetPortalStateService);

  readonly closeAction = signal<'tray' | 'exit'>('tray');
  readonly testUrl = signal('https://cp.cloudflare.com/generate_204');
  readonly testInterval = signal(300);
}
