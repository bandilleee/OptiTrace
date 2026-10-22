'use client';
import { type Identity } from './api';

export type Props = {
  tenant: Identity['tenants'][number];
  identity: Identity;
};
