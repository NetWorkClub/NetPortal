import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { I18nService } from '@core/services/i18n.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { PlatformService } from '@core/services/platform.service';
import { formatDuration, formatSpeed } from '@core/utils/format.util';
import { IconComponent } from '@shared/components/icon/icon.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-desktop-rail',
  standalone: true,
  imports: [RouterLink, RouterLinkActive, IconComponent, TranslatePipe],
  templateUrl: './desktop-rail.component.html',
  styleUrl: './desktop-rail.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class DesktopRailComponent {
  readonly store = inject(NetPortalStateService);
  readonly platformSvc = inject(PlatformService);
  readonly i18n = inject(I18nService);

  rxText(): string {
    return formatSpeed(this.store.rxMbps(), this.i18n.lang());
  }

  txText(): string {
    return formatSpeed(this.store.txMbps(), this.i18n.lang());
  }

  uptimeText(): string {
    return formatDuration(this.store.uptimeSec());
  }
}
