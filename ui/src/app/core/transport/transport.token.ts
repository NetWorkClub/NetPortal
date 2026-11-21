import { InjectionToken } from '@angular/core';
import { TransportContract } from './transport.interface';

export const NETPORTAL_TRANSPORT = new InjectionToken<TransportContract>('NETPORTAL_TRANSPORT');
