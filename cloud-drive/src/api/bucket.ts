/**
 * Bucket API
 * Handles bucket management operations
 */

import apiClient from './client';
import type { Bucket, BucketListResponse } from '../types';

/**
 * List all buckets
 */
export const listBuckets = async (): Promise<Bucket[]> => {
  const response = await apiClient.get<BucketListResponse>('/buckets');
  return response.data.buckets || [];
};

/**
 * Create a new bucket
 */
export const createBucket = async (name: string): Promise<Bucket> => {
  const response = await apiClient.put<Bucket>(`/buckets/${encodeURIComponent(name)}`);
  return response.data;
};

/**
 * Delete a bucket
 */
export const deleteBucket = async (name: string): Promise<void> => {
  await apiClient.delete(`/buckets/${encodeURIComponent(name)}`);
};
