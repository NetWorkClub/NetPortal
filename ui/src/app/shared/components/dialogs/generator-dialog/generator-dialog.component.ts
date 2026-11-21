import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  effect,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ModalService } from '@core/services/modal.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { getLatencyClass } from '@core/utils/format.util';
import { IconComponent } from '@shared/components/icon/icon.component';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-generator-dialog',
  standalone: true,
  imports: [FormsModule, IconComponent, ToggleSwitchComponent, TranslatePipe],
  templateUrl: './generator-dialog.component.html',
  styleUrl: './generator-dialog.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class GeneratorDialogComponent {
  readonly modals = inject(ModalService);
  readonly store = inject(NetPortalStateService);

  private readonly dialogRef = viewChild.required<ElementRef<HTMLDialogElement>>('dlg');

  readonly warpMtu = signal(1280);
  readonly psiphonRegion = signal('DE');
  readonly psiphonMode = signal<'node' | 'fallback'>('node');

  private downOutside = false;

  constructor() {
    effect(() => {
      const dlg = this.dialogRef().nativeElement;
      const shouldOpen = this.modals.activeDialog() === 'generator';
      if (shouldOpen && !dlg.open) {
        dlg.showModal();
      } else if (!shouldOpen && dlg.open) {
        dlg.close();
      }
    });
  }

  latencyClass(ms: number): string {
    return getLatencyClass(ms);
  }

  onNativeClose(): void {
    const dlg = this.dialogRef().nativeElement;
    const ret = dlg.returnValue;
    dlg.returnValue = '';
    if (ret !== 'back') {
      this.modals.close('generator');
    }
  }

  onPointerDown(e: PointerEvent): void {
    if ('closedBy' in HTMLDialogElement.prototype) return;
    this.downOutside = this.isOutside(e);
  }

  onPointerUp(e: PointerEvent): void {
    if ('closedBy' in HTMLDialogElement.prototype) return;
    if (this.downOutside && this.isOutside(e)) {
      this.dialogRef().nativeElement.close();
    }
  }

  goBack(): void {
    this.dialogRef().nativeElement.close('back');
    this.modals.backToAdd();
  }

  generateWarp(): void {
    this.store.generateWarpNode(this.warpMtu());
    this.dialogRef().nativeElement.close('ok');
  }

  addPsiphon(): void {
    this.store.addPsiphonNode(this.psiphonRegion(), this.psiphonMode());
    this.dialogRef().nativeElement.close('ok');
  }

  closeDialog(): void {
    this.dialogRef().nativeElement.close('cancel');
  }

  private isOutside(e: PointerEvent): boolean {
    const r = this.dialogRef().nativeElement.getBoundingClientRect();
    return e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom;
  }
}
