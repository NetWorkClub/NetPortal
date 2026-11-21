import { HttpClient } from '@angular/common/http';
import { Injectable, inject, signal } from '@angular/core';
import { AppLang } from '@core/models';
import enLocale from '@locales/en.json';
import ruLocale from '@locales/ru.json';

type LocaleNode = string | { [key: string]: LocaleNode };

const BUNDLED_LOCALES: Record<AppLang, Record<string, LocaleNode>> = {
  ru: ruLocale as Record<string, LocaleNode>,
  en: enLocale as Record<string, LocaleNode>,
};

@Injectable({ providedIn: 'root' })
export class I18nService {
  private readonly http = inject(HttpClient, { optional: true });

  readonly lang = signal<AppLang>('ru');
  readonly dict = signal<Record<string, LocaleNode>>(BUNDLED_LOCALES.ru);

  constructor() {
    document.documentElement.lang = this.lang();
  }

  setLang(next: AppLang): void {
    const normalized: AppLang = next === 'en' ? 'en' : 'ru';
    this.lang.set(normalized);
    document.documentElement.lang = normalized;
    this.dict.set(BUNDLED_LOCALES[normalized]);

    if (this.http && typeof window !== 'undefined' && window.location.protocol.startsWith('http')) {
      this.http.get<Record<string, LocaleNode>>(`./locales/${normalized}.json`).subscribe({
        next: (loaded) => {
          if (this.lang() === normalized && loaded) {
            this.dict.set(loaded);
          }
        },
        error: () => {
          // Keep bundled locale fallback on offline/asset errors
        },
      });
    }
  }

  t(key: string, vars?: Record<string, string | number>): string {
    const current = this.dict();
    let val = this.lookup(current, key);
    if (val === undefined) {
      val = this.lookup(BUNDLED_LOCALES.ru, key);
    }
    const base = val ?? key;
    if (!vars) return base;
    return base.replace(/\{(\w+)\}/g, (_, k: string) =>
      vars[k] !== undefined ? String(vars[k]) : `{${k}}`
    );
  }

  private lookup(tree: Record<string, LocaleNode>, key: string): string | undefined {
    const parts = key.split('.');
    let cur: LocaleNode | undefined = tree;
    for (const part of parts) {
      if (cur && typeof cur === 'object' && part in cur) {
        cur = cur[part];
      } else {
        return undefined;
      }
    }
    return typeof cur === 'string' ? cur : undefined;
  }
}
