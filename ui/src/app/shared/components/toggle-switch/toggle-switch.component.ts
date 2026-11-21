import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';

@Component({
  selector: 'np-toggle-switch',
  standalone: true,
  template: `
    <button
      type="button"
      class="switch"
      role="switch"
      [attr.id]="inputId() || null"
      [attr.aria-checked]="checked()"
      [attr.aria-label]="ariaLabel() || null"
      [disabled]="disabled()"
      (click)="onClick()"
    ></button>
  `,
  styleUrl: './toggle-switch.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ToggleSwitchComponent {
  readonly checked = input<boolean>(false);
  readonly disabled = input<boolean>(false);
  readonly ariaLabel = input<string>('');
  readonly inputId = input<string>('');
  readonly toggled = output<boolean>();

  onClick(): void {
    if (!this.disabled()) {
      this.toggled.emit(!this.checked());
    }
  }
}
