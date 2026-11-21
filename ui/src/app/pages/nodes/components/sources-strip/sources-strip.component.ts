import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { I18nService } from '@core/services/i18n.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-sources-strip',
  standalone: true,
  imports: [TranslatePipe],
  templateUrl: './sources-strip.component.html',
  styleUrl: './sources-strip.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SourcesStripComponent {
  readonly store = inject(NetPortalStateService);
  private readonly i18n = inject(I18nService);

  countForSource(sourceId: string): number {
    return this.store.nodes().filter((n) => n.sourceId === sourceId).length;
  }

  quotaText(quota?: string, quotaKey?: string): string {
    return quotaKey ? this.i18n.t(quotaKey) : quota ?? '';
  }
}
