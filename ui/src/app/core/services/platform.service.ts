import { DestroyRef, Injectable, effect, inject, signal } from '@angular/core';
import { PlatformTarget } from '@core/models';

@Injectable({ providedIn: 'root' })
export class PlatformService {
  private readonly destroyRef = inject(DestroyRef);

  readonly platform = signal<PlatformTarget>(this.detectPlatform());

  constructor() {
    effect(() => {
      document.documentElement.dataset['platform'] = this.platform();
    });

    if (typeof window !== 'undefined') {
      const onResize = (): void => {
        const next = this.detectPlatform();
        if (next !== this.platform()) {
          this.platform.set(next);
        }
      };
      window.addEventListener('resize', onResize);
      this.destroyRef.onDestroy(() => window.removeEventListener('resize', onResize));
    }
  }

  private detectPlatform(): PlatformTarget {
    if (typeof window === 'undefined') return 'desktop';
    const qp = new URLSearchParams(window.location.search).get('platform');
    if (qp === 'android' || qp === 'desktop') return qp;
    return window.innerWidth < 900 ? 'android' : 'desktop';
  }
}
