import { ChangeDetectionStrategy, Component, input, output } from '@angular/core';
import { ConflictReport } from '@core/models';
import { TranslatePipe } from '@shared/pipes/translate.pipe';
import { IconComponent } from '../icon/icon.component';

@Component({
  selector: 'np-conflict-banner',
  standalone: true,
  imports: [IconComponent, TranslatePipe],
  templateUrl: './conflict-banner.component.html',
  styleUrl: './conflict-banner.component.scss',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class ConflictBannerComponent {
  readonly conflicts = input.required<ConflictReport[]>();
  readonly showClean = input<boolean>(false);
  readonly resolved = output<string>();
}
