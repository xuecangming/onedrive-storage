/**
 * Type definitions for the Cloud Drive application
 * These types match the middleware API response structures
 */

// File System Types
export interface FileItem {
  name: string;
  path: string;
  type: 'file' | 'directory';
  size?: number;
  mime_type?: string;
  created_at?: string;
  updated_at?: string;
}

export interface DirectoryListing {
  path: string;
  items: FileItem[];
}

// Bucket Types
export interface Bucket {
  name: string;
  object_count: number;
  total_size: number;
  created_at?: string;
  updated_at?: string;
}

export interface BucketListResponse {
  buckets: Bucket[];
}

// Storage Space Types
export interface SpaceOverview {
  total_accounts: number;
  active_accounts: number;
  total_space: number;
  used_space: number;
  available_space: number;
  usage_percent: number;
}

// Account Types
export interface Account {
  id: string;
  name: string;
  email: string;
  status: string;
  total_space: number;
  used_space: number;
  priority: number;
  last_sync?: string;
  created_at?: string;
  updated_at?: string;
}

export interface AccountListResponse {
  accounts: Account[];
}

// Upload Progress
export interface UploadProgress {
  id: string;
  filename: string;
  progress: number;
  status: 'pending' | 'uploading' | 'success' | 'error';
  error?: string;
}

// API Response Types
export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}

export interface HealthResponse {
  status: string;
  timestamp?: string;
  components?: {
    database: string;
    cache: string;
    onedrive: string;
  };
}

export interface InfoResponse {
  name: string;
  version: string;
  api_version: string;
}
