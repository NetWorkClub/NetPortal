import { ChangeDetectionStrategy, Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { RouterLink } from '@angular/router';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { ToastService } from '@core/services/toast.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-obfuscation-settings-page',
  standalone: true,
  imports: [FormsModule, RouterLink, IconComponent, ToggleSwitchComponent, TranslatePipe],
  templateUrl: './obfuscation-settings.page.html',
  styleUrl: './obfuscation-settings.page.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ObfuscationSettingsPageComponent {
  readonly store = inject(NetPortalStateService);
  private readonly toast = inject(ToastService);

  readonly tlsSize = signal('10-30');
  readonly tlsDelay = signal('1-5');
  readonly awgJc = signal(5);
  readonly awgJrange = signal('50 - 1000');

  onByedpiPresetChange(preset: 'default' | 'mobile' | 'aggressive'): void {
    this.store.byedpiPreset.set(preset);
    const map: Record<typeof preset, string> = {
      default: '--split 1 --disorder 3+s --tlsrec 1+s --port 10801',
      mobile: '--oob 1 --fake -1 --ttl 8 --port 10801',
      aggressive: '--split 2 --tlsrec 5+s --disoob 1 --port 10801',
    };
    this.store.byedpiFlags.set(map[preset]);
  }

  pickZapretStrategy(id: string): void {
    this.store.zapretSelected.set(id);
    this.store.pushLog('info', 'dpi', `Zapret strategy changed to ${id}`);
    this.toast.show(id);
  }
}
