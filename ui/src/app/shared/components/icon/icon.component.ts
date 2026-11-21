import { ChangeDetectionStrategy, Component, input } from '@angular/core';

export type IconName =
  | 'home'
  | 'nodes'
  | 'routing'
  | 'settings'
  | 'logs'
  | 'about'
  | 'plus'
  | 'back'
  | 'close'
  | 'chevron-up'
  | 'chevron-down'
  | 'chevron-right'
  | 'edit'
  | 'trash'
  | 'clipboard'
  | 'file'
  | 'spark'
  | 'shield'
  | 'shield-ok'
  | 'warning'
  | 'power'
  | 'arrow-right'
  | 'sliders'
  | 'bolt';

@Component({
  selector: 'np-icon',
  standalone: true,
  templateUrl: './icon.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  styles: `:host { display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; }`,
})
export class IconComponent {
  readonly name = input.required<IconName>();
  readonly size = input<number>(18);
}
