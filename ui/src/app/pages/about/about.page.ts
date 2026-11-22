import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { I18nService } from '@core/services/i18n.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { ToastService } from '@core/services/toast.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-about-page',
  standalone: true,
  imports: [IconComponent, TranslatePipe],
  templateUrl: './about.page.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AboutPageComponent {
  readonly toast = inject(ToastService);
  private readonly i18n = inject(I18nService);
  private readonly store = inject(NetPortalStateService);

  checkUpdates(): void {
    this.toast.show(this.i18n.t('about.upToDate'));
  }

  openDir(): void {
    this.toast.show('~/.config/netportal');
  }

  async exportDiagnostics(): Promise<void> {
    try {
      const report = await this.store.exportDiagnostics();
      const blob = new Blob([report], { type: 'application/json;charset=utf-8' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'netportal-diagnostics.json';
      a.click();
      URL.revokeObjectURL(url);
      this.toast.show('netportal-diagnostics.json');
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      this.toast.show(msg);
    }
  }
}
