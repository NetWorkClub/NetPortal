import { provideHttpClient } from '@angular/common/http';
import { ApplicationConfig, provideZonelessChangeDetection } from '@angular/core';
import { provideRouter, withHashLocation } from '@angular/router';
import { createRuntimeTransport } from '@core/transport/transport.factory';
import { NETPORTAL_TRANSPORT } from '@core/transport/transport.token';
import { routes } from './app.routes';

export const appConfig: ApplicationConfig = {
  providers: [
    provideZonelessChangeDetection(),
    provideHttpClient(),
    provideRouter(routes, withHashLocation()),
    { provide: NETPORTAL_TRANSPORT, useFactory: createRuntimeTransport },
  ],
};
