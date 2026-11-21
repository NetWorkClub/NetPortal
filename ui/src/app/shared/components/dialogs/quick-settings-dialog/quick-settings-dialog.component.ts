import { ChangeDetectionStrategy, Component, ElementRef, effect, inject, viewChild } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { I18nService } from '@core/services/i18n.service';
import { ModalService } from '@core/services/modal.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { PlatformService } from '@core/services/platform.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-quick-settings-dialog',
  standalone: true,
  imports: [FormsModule, IconComponent, ToggleSwitchComponent, TranslatePipe],
  templateUrl: './quick-settings-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class QuickSettingsDialogComponent {
  readonly modals = inject(ModalService);
  readonly store = inject(NetPortalStateService);
  readonly platformSvc = inject(PlatformService);
  readonly i18n = inject(I18nService);

  private readonly dialogRef = viewChild.required<ElementRef<HTMLDialogElement>>('dlg');
  private downOutside = false;

  constructor() {
    effect(() => {
      const dlg = this.dialogRef().nativeElement;
      const shouldOpen = this.modals.activeDialog() === 'quick';
      if (shouldOpen && !dlg.open) {
        dlg.showModal();
      } else if (!shouldOpen && dlg.open) {
        dlg.close();
      }
    });
  }

  profileLabel(nameRu: string, nameEn: string): string {
    return this.i18n.lang() === 'en' ? nameEn : nameRu;
  }

  onNativeClose(): void {
    const dlg = this.dialogRef().nativeElement;
    dlg.returnValue = '';
    this.modals.close('quick');
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
    this.dialogRef().nativeElement.close('close');
  }

  private isOutside(e: PointerEvent): boolean {
    const r = this.dialogRef().nativeElement.getBoundingClientRect();
    return e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom;
  }
}
