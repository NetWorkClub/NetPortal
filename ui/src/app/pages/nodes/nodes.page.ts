import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ModalService } from '@core/services/modal.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';
import { NodesTableComponent } from './components/nodes-table/nodes-table.component';
import { SourcesStripComponent } from './components/sources-strip/sources-strip.component';

@Component({
  selector: 'np-nodes-page',
  standalone: true,
  imports: [FormsModule, IconComponent, SourcesStripComponent, NodesTableComponent, TranslatePipe],
  templateUrl: './nodes.page.html',
  styleUrl: './nodes.page.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class NodesPageComponent {
  readonly store = inject(NetPortalStateService);
  readonly modals = inject(ModalService);

  readonly protoFilters = [
    { id: 'all', key: 'nodes.filterAll' },
    { id: 'awg', key: 'nodes.filterAwg' },
    { id: 'proxy', key: 'nodes.filterProxy' },
    { id: 'masque', key: 'nodes.filterMasque' },
    { id: 'warp', key: 'nodes.filterWarp' },
    { id: 'psiphon', key: 'nodes.filterPsiphon' },
  ];
}
