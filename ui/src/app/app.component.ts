import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { I18nService } from '@core/services/i18n.service';
import { PlatformService } from '@core/services/platform.service';
import { ThemeService } from '@core/services/theme.service';

@Component({
  selector: 'np-root',
  standalone: true,
  imports: [RouterOutlet],
  template: '<router-outlet />',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppComponent {
  // Eagerly initialize root platform, theme, and i18n services
  private readonly _platform = inject(PlatformService);
  private readonly _theme = inject(ThemeService);
  private readonly _i18n = inject(I18nService);
}
