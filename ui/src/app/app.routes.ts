import { Routes } from '@angular/router';
import { AppShellComponent } from '@layout/app-shell/app-shell.component';

export const routes: Routes = [
  {
    path: '',
    component: AppShellComponent,
    children: [
      { path: '', redirectTo: 'home', pathMatch: 'full' },
      {
        path: 'home',
        loadComponent: () => import('@pages/home/home.page').then((m) => m.HomePageComponent),
      },
      {
        path: 'nodes',
        loadComponent: () => import('@pages/nodes/nodes.page').then((m) => m.NodesPageComponent),
      },
      {
        path: 'routing',
        loadChildren: () => import('@pages/routing/routing.routes').then((m) => m.ROUTING_ROUTES),
      },
      {
        path: 'settings',
        loadChildren: () => import('@pages/settings/settings.routes').then((m) => m.SETTINGS_ROUTES),
      },
      {
        path: 'logs',
        loadComponent: () => import('@pages/logs/logs.page').then((m) => m.LogsPageComponent),
      },
      {
        path: 'about',
        loadComponent: () => import('@pages/about/about.page').then((m) => m.AboutPageComponent),
      },
      { path: '**', redirectTo: 'home' },
    ],
  },
];
