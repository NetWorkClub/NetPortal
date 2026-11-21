import { ChangeDetectionStrategy, Component, computed, effect, inject, signal, untracked } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { SignalHop } from '@core/models';
import { I18nService } from '@core/services/i18n.service';
import { NetPortalStateService } from '@core/services/netportal-state.service';
import { PlatformService } from '@core/services/platform.service';
import { RuleEngineService } from '@core/services/rule-engine.service';
import { SignalStripComponent } from '@shared/components/signal-strip/signal-strip.component';
import { TranslatePipe } from '@shared/pipes/translate.pipe';

@Component({
  selector: 'np-rule-tester',
  standalone: true,
  imports: [FormsModule, SignalStripComponent, TranslatePipe],
  templateUrl: './rule-tester.component.html',
  styleUrl: './rule-tester.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class RuleTesterComponent {
  readonly store = inject(NetPortalStateService);
  readonly platformSvc = inject(PlatformService);
  private readonly ruleEngine = inject(RuleEngineService);
  private readonly i18n = inject(I18nService);

  readonly query = signal('');

  readonly processChipQuery = computed(() =>
    this.platformSvc.platform() === 'desktop' ? 'cursor.exe' : 'com.anthropic.claude'
  );

  readonly evalOutcome = computed(() => {
    const q = this.query().trim();
    const profile = this.store.activeProfile();
    const res = this.ruleEngine.evaluateQuery(q, profile, this.platformSvc.platform());
    return { q, res, profile };
  });

  readonly resultHops = computed<SignalHop[]>(() => {
    this.i18n.dict();
    const { q, res, profile } = this.evalOutcome();

    if (!q) {
      return [{ title: this.i18n.t('routing.testerNoInput'), meta: this.i18n.t('routing.testerNoInputMeta') }];
    }
    if (res.matchedRule) {
      const target = res.matchedRule.target;
      return [
        { title: q, meta: this.i18n.t('routing.testerRequest') },
        { title: res.matchedRule.name, meta: res.reason ?? '', tone: 'active' },
        {
          title: this.store.getTargetLabel(target),
          meta: this.i18n.t(target === 'block' ? 'routing.testerDrop' : 'routing.testerTarget'),
          tone: target === 'block' ? 'failed' : 'active',
        },
      ];
    }
    const activeCount = profile.rules.filter((r) => r.enabled).length;
    return [
      { title: q, meta: this.i18n.t('routing.testerRequest') },
      { title: this.i18n.t('routing.testerNoMatch'), meta: this.i18n.t('routing.testerActiveCount', { count: activeCount }) },
      {
        title: this.i18n.t('routing.testerDefaultPrefix', { target: this.store.getTargetLabel(profile.defaultTarget) }),
        meta: 'Default Target',
        tone: 'active',
      },
    ];
  });

  constructor() {
    effect(() => {
      const { res } = this.evalOutcome();
      const hitId = res.matchedRule ? res.matchedRule.id : null;
      untracked(() => this.store.lastHitRuleId.set(hitId));
    });
  }

  pickQuickQuery(q: string): void {
    this.query.set(q);
  }
}
