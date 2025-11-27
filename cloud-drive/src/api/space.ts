/**
 * Space and Account API
 * Handles storage space and account management
 */

import apiClient from './client';
import type { SpaceOverview, Account, AccountListResponse } from '../types';

/**
 * Get overall space statistics
 */
export const getSpaceOverview = async (): Promise<SpaceOverview> => {
  const response = await apiClient.get<SpaceOverview>('/space');
  return response.data;
};

/**
 * List all accounts
 */
export const listAccounts = async (): Promise<Account[]> => {
  const response = await apiClient.get<AccountListResponse>('/accounts');
  return response.data.accounts || [];
};

/**
 * Sync account space information
 */
export const syncAccount = async (id: string): Promise<Account> => {
  const response = await apiClient.post<Account>(`/accounts/${id}/sync`);
  return response.data;
};

/**
 * Check system health
 */
export const checkHealth = async (): Promise<boolean> => {
  try {
    await apiClient.get('/health');
    return true;
  } catch {
    return false;
  }
};
