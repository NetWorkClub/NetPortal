import { Routes } from '@angular/router';

export const ROUTING_ROUTES: Routes = [
  {
    path: '',
    loadComponent: () => import('./routing.page').then((m) => m.RoutingPageComponent),
  },
  {
    path: 'rule/new',
    loadComponent: () => import('./rule-editor/rule-editor.page').then((m) => m.RuleEditorPageComponent),
  },
  {
    path: 'rule/:id',
    loadComponent: () => import('./rule-editor/rule-editor.page').then((m) => m.RuleEditorPageComponent),
  },
];
