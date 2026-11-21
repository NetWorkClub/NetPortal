import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { I18nService } from '@core/services/i18n.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { PlatformService } from '@core/services/platform.service';
import { RuleEngineService } from '@core/services/rule-engine.service';
import { IconComponent } from '@shared/components/icon/icon.component';
import { ToggleSwitchComponent } from '@shared/components/toggle-switch/toggle-switch.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';
import { PresetsPanelComponent } from './components/presets-panel/presets-panel.component';
import { RuleListTierComponent } from './components/rule-list-tier/rule-list-tier.component';
import { RuleTesterComponent } from './components/rule-tester/rule-tester.component';
import { StickyChainComponent } from './components/sticky-chain/sticky-chain.component';

@Component({
  selector: 'np-routing-page',
  standalone: true,
  imports: [
    FormsModule,
    RouterLink,
    IconComponent,
    ToggleSwitchComponent,
    RuleTesterComponent,
    RuleListTierComponent,
    PresetsPanelComponent,
    StickyChainComponent,
    TranslatePipe,
  ],
  templateUrl: './routing.page.html',
  styleUrl: './routing.page.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RoutingPageComponent {
  readonly store = inject(NetPortalStateService);
  readonly platformSvc = inject(PlatformService);
  private readonly ruleEngine = inject(RuleEngineService);
  private readonly router = inject(Router);
  readonly i18n = inject(I18nService);

  readonly tab = signal<'rules' | 'chain'>('rules');

  readonly tier1Rules = computed(() =>
    this.store.activeProfile().rules.filter((r) => this.ruleEngine.isTier1(r))
  );

  readonly tier2Rules = computed(() =>
    this.store.activeProfile().rules.filter((r) => !this.ruleEngine.isTier1(r))
  );

  readonly tier1Hint = computed(() => {
    this.i18n.dict();
    return this.i18n.t(
      this.platformSvc.platform() === 'android' ? 'routing.tier1HintAndroid' : 'routing.tier1HintDesktop'
    );
  });

  profileLabel(nameRu: string, nameEn: string): string {
    return this.i18n.lang() === 'en' ? nameEn : nameRu;
  }

  openNewRule(): void {
    this.router.navigateByUrl('/routing/rule/new');
  }
}
