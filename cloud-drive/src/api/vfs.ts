/**
 * VFS (Virtual File System) API
 * Handles all file and directory operations
 */

import apiClient from './client';
import type { FileItem, DirectoryListing } from '../types';

// Current bucket (configurable)
let currentBucket = 'default';

export const setCurrentBucket = (bucket: string): void => {
  currentBucket = bucket;
};

export const getCurrentBucket = (): string => currentBucket;

// Encode path segments for URL
const encodePath = (path: string): string => {
  return path.split('/').map(p => encodeURIComponent(p)).join('/');
};

// Normalize path
const normalizePath = (path: string): string => {
  if (!path) return '/';
  return '/' + path.split('/').filter(p => p).join('/');
};

/**
 * List directory contents
 */
export const listDirectory = async (path: string = '/'): Promise<DirectoryListing> => {
  const normalizedPath = normalizePath(path);
  const pathWithSlash = normalizedPath.endsWith('/') ? normalizedPath : normalizedPath + '/';
  const encodedPath = encodePath(pathWithSlash);
  
  const response = await apiClient.get<DirectoryListing>(
    `/vfs/${encodeURIComponent(currentBucket)}${encodedPath}`
  );
  
  return response.data;
};

/**
 * Create directory
 */
export const createDirectory = async (path: string): Promise<void> => {
  await apiClient.post(`/vfs/${encodeURIComponent(currentBucket)}/_mkdir`, {
    path: normalizePath(path),
  });
};

/**
 * Upload file
 */
export const uploadFile = async (
  path: string,
  file: File,
  onProgress?: (percent: number) => void
): Promise<FileItem> => {
  const normalizedPath = normalizePath(path);
  const encodedPath = encodePath(normalizedPath);
  
  const response = await apiClient.put<FileItem>(
    `/vfs/${encodeURIComponent(currentBucket)}${encodedPath}`,
    file,
    {
      headers: {
        'Content-Type': file.type || 'application/octet-stream',
      },
      onUploadProgress: (progressEvent) => {
        if (progressEvent.total && onProgress) {
          const percent = Math.round((progressEvent.loaded / progressEvent.total) * 100);
          onProgress(percent);
        }
      },
    }
  );
  
  return response.data;
};

/**
 * Download file
 */
export const downloadFile = async (path: string): Promise<Blob> => {
  const normalizedPath = normalizePath(path);
  const encodedPath = encodePath(normalizedPath);
  
  const response = await apiClient.get(
    `/vfs/${encodeURIComponent(currentBucket)}${encodedPath}`,
    {
      responseType: 'blob',
    }
  );
  
  return response.data;
};

/**
 * Get file URL for preview
 */
export const getFileUrl = (path: string): string => {
  const normalizedPath = normalizePath(path);
  const encodedPath = encodePath(normalizedPath);
  return `${apiClient.defaults.baseURL}/vfs/${encodeURIComponent(currentBucket)}${encodedPath}`;
};

/**
 * Delete file
 */
export const deleteFile = async (path: string): Promise<void> => {
  const normalizedPath = normalizePath(path);
  const encodedPath = encodePath(normalizedPath);
  
  await apiClient.delete(`/vfs/${encodeURIComponent(currentBucket)}${encodedPath}`);
};

/**
 * Delete directory
 */
export const deleteDirectory = async (path: string, recursive: boolean = true): Promise<void> => {
  const normalizedPath = normalizePath(path);
  const pathWithSlash = normalizedPath.endsWith('/') ? normalizedPath : normalizedPath + '/';
  const encodedPath = encodePath(pathWithSlash);
  
  await apiClient.delete(
    `/vfs/${encodeURIComponent(currentBucket)}${encodedPath}?type=directory&recursive=${recursive}`
  );
};

/**
 * Move file or directory
 */
export const moveItem = async (source: string, destination: string): Promise<void> => {
  await apiClient.post(`/vfs/${encodeURIComponent(currentBucket)}/_move`, {
    source: normalizePath(source),
    destination: normalizePath(destination),
  });
};

/**
 * Copy file
 */
export const copyFile = async (source: string, destination: string): Promise<void> => {
  await apiClient.post(`/vfs/${encodeURIComponent(currentBucket)}/_copy`, {
    source: normalizePath(source),
    destination: normalizePath(destination),
  });
};
