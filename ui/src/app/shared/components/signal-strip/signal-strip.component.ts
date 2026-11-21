import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { SignalHop } from '@core/models';
import { IconComponent } from '../icon/icon.component';

@Component({
  selector: 'np-signal-strip',
  standalone: true,
  imports: [IconComponent],
  template: `
    <div class="signal-strip">
      @for (hop of hops(); track $index; let last = $last) {
        <div
          class="signal-hop"
          [class.is-active]="hop.tone === 'active'"
          [class.is-failover]="hop.tone === 'failover'"
          [class.is-failed]="hop.tone === 'failed'"
        >
          <strong>{{ hop.title }}</strong>
          @if (hop.meta) {
            <span class="signal-hop-meta">{{ hop.meta }}</span>
          }
        </div>
        @if (!last) {
          <span
            class="signal-arrow"
            [class.is-active]="hops()[$index + 1].tone === 'active'"
            [class.is-failover]="hops()[$index + 1].tone === 'failover'"
          >
            <np-icon name="arrow-right" [size]="15" />
          </span>
        }
      }
    </div>
  `,
  styleUrl: './signal-strip.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class SignalStripComponent {
  readonly hops = input.required<SignalHop[]>();
}
