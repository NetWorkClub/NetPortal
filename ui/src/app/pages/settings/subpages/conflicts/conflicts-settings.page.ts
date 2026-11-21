import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { RouterLink } from '@angular/router';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { ConflictBannerComponent } from '@shared/components/conflict-banner/conflict-banner.component';
import { IconComponent } from '@shared/components/icon/icon.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-conflicts-settings-page',
  standalone: true,
  imports: [RouterLink, IconComponent, ConflictBannerComponent, TranslatePipe],
  templateUrl: './conflicts-settings.page.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ConflictsSettingsPageComponent {
  readonly store = inject(NetPortalStateService);
}
