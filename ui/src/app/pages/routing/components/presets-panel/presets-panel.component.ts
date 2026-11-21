import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { PresetDef } from '@core/models';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-presets-panel',
  standalone: true,
  imports: [ToggleSwitchComponent, TranslatePipe],
  templateUrl: './presets-panel.component.html',
  styleUrl: './presets-panel.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PresetsPanelComponent {
  readonly store = inject(NetPortalStateService);

  isEnabled(preset: PresetDef): boolean {
    return this.store.activeProfile().rules.some((r) => r.presetId === preset.id && r.enabled);
  }

  badgeClass(target: string): string {
    if (target === 'block') return 'danger';
    if (target === 'direct') return 'ok';
    return 'accent';
  }
}
