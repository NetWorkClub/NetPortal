import { Routes } from '@angular/router';

export const SETTINGS_ROUTES: Routes = [
  {
    path: '',
    loadComponent: () => import('./settings-menu.page').then((m) => m.SettingsMenuPageComponent),
  },
  {
    path: 'general',
    loadComponent: () => import('./subpages/general/general-settings.page').then((m) => m.GeneralSettingsPageComponent),
  },
  {
    path: 'tunnel',
    loadComponent: () => import('./subpages/tunnel/tunnel-settings.page').then((m) => m.TunnelSettingsPageComponent),
  },
  {
    path: 'obfuscation',
    loadComponent: () =>
      import('./subpages/obfuscation/obfuscation-settings.page').then((m) => m.ObfuscationSettingsPageComponent),
  },
  {
    path: 'dns',
    loadComponent: () => import('./subpages/dns/dns-settings.page').then((m) => m.DnsSettingsPageComponent),
  },
  {
    path: 'conflicts',
    loadComponent: () =>
      import('./subpages/conflicts/conflicts-settings.page').then((m) => m.ConflictsSettingsPageComponent),
  },
];
