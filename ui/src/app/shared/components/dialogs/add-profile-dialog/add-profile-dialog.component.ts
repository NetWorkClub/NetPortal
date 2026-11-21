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
import { I18nService } from '@core/services/i18n.service';
import { ModalService } from '@core/services/modal.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { ToastService } from '@core/services/toast.service';
import { detectImportFormat } from '@core/utils/format.util';
import { IconComponent } from '@shared/components/icon/icon.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-add-profile-dialog',
  standalone: true,
  imports: [FormsModule, IconComponent, TranslatePipe],
  templateUrl: './add-profile-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AddProfileDialogComponent {
  readonly modals = inject(ModalService);
  private readonly store = inject(NetPortalStateService);
  private readonly toast = inject(ToastService);
  private readonly i18n = inject(I18nService);

  private readonly dialogRef = viewChild.required<ElementRef<HTMLDialogElement>>('dlg');
  private readonly fileInputRef = viewChild.required<ElementRef<HTMLInputElement>>('fileInput');

  readonly nameInput = signal('');
  readonly rawInput = signal('');
  readonly fileName = signal('');
  readonly detectLabel = computed(() => {
    const raw = this.rawInput();
    const fname = this.fileName();
    if (!raw.trim() && !fname) {
      return this.i18n.lang() === 'en' ? 'Auto-detect: waiting for input' : 'Автоопределение: ожидание ввода';
    }
    const det = detectImportFormat(raw, fname);
    return fname ? `${fname} -> ${det.label}` : det.label;
  });

  private downOutside = false;

  constructor() {
    effect(() => {
      const dlg = this.dialogRef().nativeElement;
      const shouldOpen = this.modals.activeDialog() === 'add';
      if (shouldOpen && !dlg.open) {
        dlg.showModal();
      } else if (!shouldOpen && dlg.open) {
        dlg.close();
      }
    });
  }

  onNativeClose(): void {
    const dlg = this.dialogRef().nativeElement;
    dlg.returnValue = '';
    this.modals.close('add');
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

  fillFromClipboard(): void {
    this.fileName.set('');
    this.nameInput.set('VLESS Finland (Clipboard)');
    this.rawInput.set('vless://uuid@95.217.45.102:443?security=reality&sni=dl.google.com#FI-Node');
  }

  triggerFilePicker(): void {
    this.fileInputRef().nativeElement.click();
  }

  onFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const content = String(reader.result || '').slice(0, 16000);
      const det = detectImportFormat(content, file.name);
      this.fileName.set(file.name);
      this.nameInput.set(file.name.replace(/\.[^.]+$/, '') || 'File Import');
      this.rawInput.set(content);
      this.toast.show(`${file.name} (${det.label})`);
    };
    reader.readAsText(file);
    input.value = '';
  }

  openGeneratorFromAdd(): void {
    this.modals.openGenerator(true, 'warp');
  }

  submitImport(): void {
    this.store.importRawNode(this.nameInput(), this.rawInput(), this.fileName());
    this.nameInput.set('');
    this.rawInput.set('');
    this.fileName.set('');
    this.dialogRef().nativeElement.close('import');
  }

  closeDialog(): void {
    this.dialogRef().nativeElement.close('cancel');
  }

  private isOutside(e: PointerEvent): boolean {
    const r = this.dialogRef().nativeElement.getBoundingClientRect();
    return e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom;
  }
}
