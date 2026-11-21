import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { ToastService } from '@core/services/toast.service';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-logs-page',
  standalone: true,
  imports: [FormsModule, TranslatePipe],
  templateUrl: './logs.page.html',
  styleUrl: './logs.page.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LogsPageComponent {
  readonly store = inject(NetPortalStateService);
  private readonly toast = inject(ToastService);

  togglePause(): void {
    this.store.logsPaused.update((v) => !v);
  }

  clearLogs(): void {
    this.store.logs.set([]);
  }

  exportLogs(): void {
    this.toast.show(`Exported ${this.store.logs().length} lines`);
  }
}
