import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { ToastService } from '@core/services/toast.service';
import { I18nService } from '@core/services/i18n.service';
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

  checkUpdates(): void {
    this.toast.show(this.i18n.t('about.upToDate'));
  }

  openDir(): void {
    this.toast.show('~/.config/netportal');
  }

  exportDiagnostics(): void {
    this.toast.show('diagnostics-masked.zip');
  }
}
