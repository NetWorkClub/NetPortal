import { Injectable, signal } from '@angular/core';

export type ActiveDialogKind = 'add' | 'generator' | 'quick' | 'inspector' | null;
export type GeneratorTab = 'warp' | 'psiphon';

@Injectable({ providedIn: 'root' })
export class ModalService {
  readonly activeDialog = signal<ActiveDialogKind>(null);
  readonly genTab = signal<GeneratorTab>('warp');
  readonly genOpenedFromAdd = signal(false);
  readonly inspectNodeId = signal<string | null>(null);

  openAdd(): void {
    this.activeDialog.set('add');
  }

  openQuick(): void {
    this.activeDialog.set('quick');
  }

  openGenerator(fromAdd = false, tab: GeneratorTab = 'warp'): void {
    this.genOpenedFromAdd.set(fromAdd);
    this.genTab.set(tab);
    this.activeDialog.set('generator');
  }

  setGenTab(tab: GeneratorTab): void {
    this.genTab.set(tab);
  }

  backToAdd(): void {
    this.activeDialog.set('add');
  }

  openInspector(nodeId: string): void {
    this.inspectNodeId.set(nodeId);
    this.activeDialog.set('inspector');
  }

  close(kind?: ActiveDialogKind): void {
    if (!kind || this.activeDialog() === kind) {
      this.activeDialog.set(null);
    }
  }
}
