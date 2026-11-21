import {
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  inject,
  viewChild,
} from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { ToastService } from '@core/services/toast.service';
import { AddProfileDialogComponent } from '@shared/components/dialogs/add-profile-dialog/add-profile-dialog.component';
import { GeneratorDialogComponent } from '@shared/components/dialogs/generator-dialog/generator-dialog.component';
import { NodeInspectorDialogComponent } from '@shared/components/dialogs/node-inspector-dialog/node-inspector-dialog.component';
import { QuickSettingsDialogComponent } from '@shared/components/dialogs/quick-settings-dialog/quick-settings-dialog.component';
import { DesktopRailComponent } from '@layout/desktop-rail/desktop-rail.component';
import {
  MobileBottomnavComponent,
  MobileTopbarComponent,
} from '@layout/mobile-chrome/mobile-chrome.components';
import { filter } from 'rxjs';

@Component({
  selector: 'np-app-shell',
  standalone: true,
  imports: [
    RouterOutlet,
    DesktopRailComponent,
    MobileTopbarComponent,
    MobileBottomnavComponent,
    AddProfileDialogComponent,
    GeneratorDialogComponent,
    QuickSettingsDialogComponent,
    NodeInspectorDialogComponent,
  ],
  templateUrl: './app-shell.component.html',
  styleUrl: './app-shell.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AppShellComponent {
  readonly toast = inject(ToastService);
  private readonly router = inject(Router);
  private readonly mainRef = viewChild.required<ElementRef<HTMLElement>>('mainWorkspace');

  constructor() {
    this.router.events
      .pipe(
        filter((e): e is NavigationEnd => e instanceof NavigationEnd),
        takeUntilDestroyed()
      )
      .subscribe(() => {
        const el = this.mainRef().nativeElement;
        el.scrollTop = 0;
      });
  }
}
