import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  computed,
  effect,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ModalService } from '@core/services/modal.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-node-inspector-dialog',
  standalone: true,
  imports: [FormsModule, IconComponent, TranslatePipe],
  templateUrl: './node-inspector-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class NodeInspectorDialogComponent {
  readonly modals = inject(ModalService);
  private readonly store = inject(NetPortalStateService);

  private readonly dialogRef = viewChild.required<ElementRef<HTMLDialogElement>>('dlg');
  private downOutside = false;

  readonly node = computed(() => {
    const id = this.modals.inspectNodeId();
    return id ? this.store.nodes().find((n) => n.id === id) ?? null : null;
  });

  readonly editName = signal('');
  readonly editMtu = signal(1280);
  readonly editParams = signal<Array<{ key: string; value: string }>>([]);

  constructor() {
    effect(() => {
      const dlg = this.dialogRef().nativeElement;
      const shouldOpen = this.modals.activeDialog() === 'inspector';
      if (shouldOpen) {
        const current = this.node();
        if (current) {
          this.editName.set(current.name);
          this.editMtu.set(current.mtu);
          this.editParams.set(
            Object.entries(current.params || {}).map(([key, value]) => ({ key, value }))
          );
        }
        if (!dlg.open) dlg.showModal();
      } else if (dlg.open) {
        dlg.close();
      }
    });
  }

  updateParamValue(idx: number, value: string): void {
    this.editParams.update((list) => list.map((p, i) => (i === idx ? { ...p, value } : p)));
  }

  save(): void {
    const current = this.node();
    if (!current) return;
    const paramsRecord: Record<string, string> = {};
    for (const p of this.editParams()) {
      paramsRecord[p.key] = p.value;
    }
    this.store.updateNodeParams(current.id, this.editName(), this.editMtu(), paramsRecord);
    this.dialogRef().nativeElement.close('save');
  }

  onNativeClose(): void {
    const dlg = this.dialogRef().nativeElement;
    dlg.returnValue = '';
    this.modals.close('inspector');
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

  closeDialog(): void {
    this.dialogRef().nativeElement.close('cancel');
  }

  private isOutside(e: PointerEvent): boolean {
    const r = this.dialogRef().nativeElement.getBoundingClientRect();
    return e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom;
  }
}
