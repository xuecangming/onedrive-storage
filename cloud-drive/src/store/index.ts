/**
 * Application state management using Zustand
 */

import { create } from 'zustand';
import type { FileItem, UploadProgress, SpaceOverview } from '../types';

interface AppState {
  // Current path and bucket
  currentPath: string;
  currentBucket: string;
  
  // File list
  files: FileItem[];
  isLoading: boolean;
  
  // Selection
  selectedPaths: Set<string>;
  
  // View mode
  viewMode: 'grid' | 'list';
  
  // Upload progress
  uploads: Map<string, UploadProgress>;
  
  // Storage info
  spaceInfo: SpaceOverview | null;
  
  // Actions
  setCurrentPath: (path: string) => void;
  setCurrentBucket: (bucket: string) => void;
  setFiles: (files: FileItem[]) => void;
  setLoading: (loading: boolean) => void;
  setSelectedPaths: (paths: Set<string>) => void;
  toggleSelection: (path: string) => void;
  selectAll: () => void;
  clearSelection: () => void;
  setViewMode: (mode: 'grid' | 'list') => void;
  addUpload: (upload: UploadProgress) => void;
  updateUpload: (id: string, update: Partial<UploadProgress>) => void;
  removeUpload: (id: string) => void;
  setSpaceInfo: (info: SpaceOverview | null) => void;
}

export const useAppStore = create<AppState>((set, get) => ({
  // Initial state
  currentPath: '/',
  currentBucket: 'default',
  files: [],
  isLoading: false,
  selectedPaths: new Set(),
  viewMode: 'grid',
  uploads: new Map(),
  spaceInfo: null,
  
  // Actions
  setCurrentPath: (path) => set({ currentPath: path }),
  
  setCurrentBucket: (bucket) => set({ currentBucket: bucket }),
  
  setFiles: (files) => set({ files }),
  
  setLoading: (loading) => set({ isLoading: loading }),
  
  setSelectedPaths: (paths) => set({ selectedPaths: paths }),
  
  toggleSelection: (path) => {
    const current = new Set(get().selectedPaths);
    if (current.has(path)) {
      current.delete(path);
    } else {
      current.add(path);
    }
    set({ selectedPaths: current });
  },
  
  selectAll: () => {
    const allPaths = new Set(get().files.map(f => f.path));
    set({ selectedPaths: allPaths });
  },
  
  clearSelection: () => set({ selectedPaths: new Set() }),
  
  setViewMode: (mode) => set({ viewMode: mode }),
  
  addUpload: (upload) => {
    const uploads = new Map(get().uploads);
    uploads.set(upload.id, upload);
    set({ uploads });
  },
  
  updateUpload: (id, update) => {
    const uploads = new Map(get().uploads);
    const existing = uploads.get(id);
    if (existing) {
      uploads.set(id, { ...existing, ...update });
      set({ uploads });
    }
  },
  
  removeUpload: (id) => {
    const uploads = new Map(get().uploads);
    uploads.delete(id);
    set({ uploads });
  },
  
  setSpaceInfo: (info) => set({ spaceInfo: info }),
}));
