import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { AntiBotState } from '@core/models';
import { ModalService } from '@core/services/modal.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { getLatencyClass } from '@core/utils/format.util';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-nodes-table',
  standalone: true,
  imports: [TranslatePipe],
  templateUrl: './nodes-table.component.html',
  styleUrl: './nodes-table.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class NodesTableComponent {
  readonly store = inject(NetPortalStateService);
  readonly modals = inject(ModalService);

  readonly allVisibleChecked = computed(() => {
    const list = this.store.filteredNodes();
    const checked = this.store.checkedNodes();
    return list.length > 0 && list.every((n) => checked.has(n.id));
  });

  protoBadgeClass(proto: string): string {
    const map: Record<string, string> = {
      'AWG 3.1': 'accent',
      'AWG 3.0': 'accent',
      'AWG 2.0': '',
      VLESS: 'info',
      HY2: 'info',
      MASQUE: 'masque',
      WARP: 'warn',
      PSIPHON: 'ok',
    };
    return map[proto] ?? '';
  }

  antiBotClass(status: AntiBotState): string {
    if (status === 'Clean') return 'ok';
    if (status === 'Challenge') return 'warn';
    return 'danger';
  }

  latencyClass(ms: number | null): string {
    return getLatencyClass(ms);
  }

  resetFilters(): void {
    this.store.activeSource.set('all');
    this.store.protoFilter.set('all');
    this.store.searchQuery.set('');
  }
}
