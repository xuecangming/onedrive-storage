/**
 * React Query hooks for file operations
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { 
  listDirectory, 
  createDirectory, 
  deleteFile, 
  deleteDirectory, 
  moveItem, 
  copyFile,
  uploadFile as uploadFileApi
} from '../api';
import { useAppStore } from '../store';
import { message } from 'antd';
import type { FileItem } from '../types';
import { generateId, joinPath } from '../utils';

// Query keys
export const queryKeys = {
  directory: (path: string) => ['directory', path],
  buckets: ['buckets'],
  space: ['space'],
  accounts: ['accounts'],
};

/**
 * Hook for fetching directory contents
 */
export const useDirectory = (path: string) => {
  const setFiles = useAppStore(state => state.setFiles);
  const setLoading = useAppStore(state => state.setLoading);
  
  return useQuery({
    queryKey: queryKeys.directory(path),
    queryFn: async () => {
      setLoading(true);
      try {
        const data = await listDirectory(path);
        setFiles(data.items || []);
        return data;
      } finally {
        setLoading(false);
      }
    },
    refetchOnWindowFocus: false,
  });
};

/**
 * Hook for creating directories
 */
export const useCreateDirectory = () => {
  const queryClient = useQueryClient();
  const currentPath = useAppStore(state => state.currentPath);
  
  return useMutation({
    mutationFn: async (name: string) => {
      const fullPath = joinPath(currentPath, name);
      await createDirectory(fullPath);
    },
    onSuccess: () => {
      message.success('文件夹创建成功');
      queryClient.invalidateQueries({ queryKey: queryKeys.directory(currentPath) });
    },
    onError: (error: Error) => {
      message.error('创建文件夹失败: ' + error.message);
    },
  });
};

/**
 * Hook for deleting items
 */
export const useDeleteItems = () => {
  const queryClient = useQueryClient();
  const currentPath = useAppStore(state => state.currentPath);
  const files = useAppStore(state => state.files);
  const clearSelection = useAppStore(state => state.clearSelection);
  
  return useMutation({
    mutationFn: async (paths: string[]) => {
      const results = await Promise.allSettled(
        paths.map(async (path) => {
          const item = files.find(f => f.path === path);
          if (item?.type === 'directory') {
            await deleteDirectory(path, true);
          } else {
            await deleteFile(path);
          }
        })
      );
      
      const failed = results.filter(r => r.status === 'rejected').length;
      const succeeded = results.filter(r => r.status === 'fulfilled').length;
      
      return { failed, succeeded };
    },
    onSuccess: ({ failed, succeeded }) => {
      clearSelection();
      if (failed === 0) {
        message.success(`成功删除 ${succeeded} 个项目`);
      } else {
        message.warning(`成功 ${succeeded} 个，失败 ${failed} 个`);
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.directory(currentPath) });
    },
    onError: (error: Error) => {
      message.error('删除失败: ' + error.message);
    },
  });
};

/**
 * Hook for renaming items (using move)
 */
export const useRenameItem = () => {
  const queryClient = useQueryClient();
  const currentPath = useAppStore(state => state.currentPath);
  
  return useMutation({
    mutationFn: async ({ oldPath, newPath }: { oldPath: string; newPath: string }) => {
      await moveItem(oldPath, newPath);
    },
    onSuccess: () => {
      message.success('重命名成功');
      queryClient.invalidateQueries({ queryKey: queryKeys.directory(currentPath) });
    },
    onError: (error: Error) => {
      message.error('重命名失败: ' + error.message);
    },
  });
};

/**
 * Hook for moving items
 */
export const useMoveItems = () => {
  const queryClient = useQueryClient();
  const currentPath = useAppStore(state => state.currentPath);
  const clearSelection = useAppStore(state => state.clearSelection);
  const files = useAppStore(state => state.files);
  
  return useMutation({
    mutationFn: async ({ paths, destination }: { paths: string[]; destination: string }) => {
      const results = await Promise.allSettled(
        paths.map(async (path) => {
          const item = files.find(f => f.path === path);
          if (item) {
            const newPath = joinPath(destination, item.name);
            await moveItem(path, newPath);
          }
        })
      );
      
      const failed = results.filter(r => r.status === 'rejected').length;
      const succeeded = results.filter(r => r.status === 'fulfilled').length;
      
      return { failed, succeeded };
    },
    onSuccess: ({ failed, succeeded }) => {
      clearSelection();
      if (failed === 0) {
        message.success(`成功移动 ${succeeded} 个项目`);
      } else {
        message.warning(`成功 ${succeeded} 个，失败 ${failed} 个`);
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.directory(currentPath) });
    },
    onError: (error: Error) => {
      message.error('移动失败: ' + error.message);
    },
  });
};

/**
 * Hook for copying items
 */
export const useCopyItems = () => {
  const queryClient = useQueryClient();
  const currentPath = useAppStore(state => state.currentPath);
  const clearSelection = useAppStore(state => state.clearSelection);
  const files = useAppStore(state => state.files);
  
  return useMutation({
    mutationFn: async ({ paths, destination }: { paths: string[]; destination: string }) => {
      const results = await Promise.allSettled(
        paths.map(async (path) => {
          const item = files.find(f => f.path === path);
          if (item && item.type === 'file') {
            const newPath = joinPath(destination, item.name);
            await copyFile(path, newPath);
          }
        })
      );
      
      const failed = results.filter(r => r.status === 'rejected').length;
      const succeeded = results.filter(r => r.status === 'fulfilled').length;
      
      return { failed, succeeded };
    },
    onSuccess: ({ failed, succeeded }) => {
      clearSelection();
      if (failed === 0) {
        message.success(`成功复制 ${succeeded} 个项目`);
      } else {
        message.warning(`成功 ${succeeded} 个，失败 ${failed} 个`);
      }
      queryClient.invalidateQueries({ queryKey: queryKeys.directory(currentPath) });
    },
    onError: (error: Error) => {
      message.error('复制失败: ' + error.message);
    },
  });
};

/**
 * Hook for uploading files
 */
export const useUploadFiles = () => {
  const queryClient = useQueryClient();
  const currentPath = useAppStore(state => state.currentPath);
  const addUpload = useAppStore(state => state.addUpload);
  const updateUpload = useAppStore(state => state.updateUpload);
  
  const uploadFile = async (file: File): Promise<FileItem> => {
    const id = generateId();
    const filePath = joinPath(currentPath, file.name);
    
    addUpload({
      id,
      filename: file.name,
      progress: 0,
      status: 'uploading',
    });
    
    try {
      const result = await uploadFileApi(filePath, file, (percent) => {
        updateUpload(id, { progress: percent });
      });
      
      updateUpload(id, { progress: 100, status: 'success' });
      return result;
    } catch (error) {
      updateUpload(id, { 
        status: 'error', 
        error: error instanceof Error ? error.message : '上传失败' 
      });
      throw error;
    }
  };
  
  const uploadFiles = async (files: File[]) => {
    const results = await Promise.allSettled(files.map(uploadFile));
    
    const succeeded = results.filter(r => r.status === 'fulfilled').length;
    const failed = results.filter(r => r.status === 'rejected').length;
    
    if (failed === 0) {
      message.success(`成功上传 ${succeeded} 个文件`);
    } else {
      message.warning(`成功 ${succeeded} 个，失败 ${failed} 个`);
    }
    
    queryClient.invalidateQueries({ queryKey: queryKeys.directory(currentPath) });
    
    return { succeeded, failed };
  };
  
  return { uploadFile, uploadFiles };
};
