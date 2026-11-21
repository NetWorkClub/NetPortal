import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterLinkActive } from '@angular/router';
import { I18nService } from '@core/services/i18n.service';
import { ModalService } from '@core/services/modal.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';
import { filter, map } from 'rxjs';

@Component({
  selector: 'np-mobile-topbar',
  standalone: true,
  imports: [IconComponent, TranslatePipe],
  templateUrl: './mobile-topbar.component.html',
  styleUrl: './mobile-topbar.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MobileTopbarComponent {
  private readonly router = inject(Router);
  private readonly i18n = inject(I18nService);
  private readonly modals = inject(ModalService);

  private readonly currentUrl = toSignal(
    this.router.events.pipe(
      filter((e): e is NavigationEnd => e instanceof NavigationEnd),
      map(() => this.router.url)
    ),
    { initialValue: this.router.url }
  );

  readonly showBack = computed(() => {
    const url = this.currentUrl();
    return (
      url.startsWith('/routing/rule') ||
      /^\/settings\/.+/.test(url) ||
      url.startsWith('/logs') ||
      url.startsWith('/about')
    );
  });

  readonly showAddAction = computed(() => {
    if (this.showBack()) return false;
    const url = this.currentUrl();
    return url.startsWith('/home') || url.startsWith('/nodes') || url === '/routing';
  });

  readonly barTitle = computed(() => {
    const url = this.currentUrl();
    this.i18n.dict();
    if (url.startsWith('/routing/rule/new')) return this.i18n.t('ruleEditor.createTitle');
    if (url.startsWith('/routing/rule/')) return this.i18n.t('ruleEditor.editTitle');
    if (url.startsWith('/settings/general')) return this.i18n.t('settings.catGeneral');
    if (url.startsWith('/settings/tunnel')) return this.i18n.t('settings.catTunnel');
    if (url.startsWith('/settings/obfuscation')) return this.i18n.t('settings.catObfuscation');
    if (url.startsWith('/settings/dns')) return this.i18n.t('settings.catDns');
    if (url.startsWith('/settings/conflicts')) return this.i18n.t('settings.catConflicts');
    if (url.startsWith('/nodes')) return this.i18n.t('nav.nodes');
    if (url.startsWith('/routing')) return this.i18n.t('nav.routing');
    if (url.startsWith('/settings')) return this.i18n.t('nav.settings');
    if (url.startsWith('/logs')) return this.i18n.t('nav.logs');
    if (url.startsWith('/about')) return this.i18n.t('nav.about');
    return 'NetPortal';
  });

  onBack(): void {
    const url = this.currentUrl();
    if (url.startsWith('/routing/rule')) {
      this.router.navigateByUrl('/routing');
    } else if (url.startsWith('/settings/') || url.startsWith('/logs') || url.startsWith('/about')) {
      this.router.navigateByUrl('/settings');
    }
  }

  onAction(): void {
    const url = this.currentUrl();
    if (url.startsWith('/routing')) {
      this.router.navigateByUrl('/routing/rule/new');
    } else {
      this.modals.openAdd();
    }
  }
}

@Component({
  selector: 'np-mobile-bottomnav',
  standalone: true,
  imports: [RouterLink, RouterLinkActive, IconComponent, TranslatePipe],
  templateUrl: './mobile-bottomnav.component.html',
  styleUrl: './mobile-bottomnav.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class MobileBottomnavComponent {}
