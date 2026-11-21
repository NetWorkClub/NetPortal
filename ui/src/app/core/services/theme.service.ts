import { Injectable, effect, signal } from '@angular/core';
import { ThemeMode } from '@core/models';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  readonly theme = signal<ThemeMode>('dark');

  constructor() {
    effect(() => {
      const current = this.theme();
      document.documentElement.dataset['theme'] = current;
      const isLight =
        current === 'light' ||
        (current === 'system' &&
          typeof window !== 'undefined' &&
          window.matchMedia('(prefers-color-scheme: light)').matches);
      const meta = document.getElementById('meta-theme-color');
      if (meta instanceof HTMLMetaElement) {
        meta.content = isLight ? '#f5f7fa' : '#11161d';
      }
    });
  }

  setTheme(mode: ThemeMode): void {
    this.theme.set(mode);
  }
}
