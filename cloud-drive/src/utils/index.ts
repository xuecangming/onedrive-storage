/**
 * Utility functions for the Cloud Drive application
 */

/**
 * Format file size to human readable format
 */
export const formatSize = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

/**
 * Format date to relative or absolute string
 */
export const formatDate = (dateString?: string): string => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  
  // Less than 1 minute
  if (diff < 60000) {
    return '刚刚';
  }
  // Less than 1 hour
  if (diff < 3600000) {
    return Math.floor(diff / 60000) + ' 分钟前';
  }
  // Less than 1 day
  if (diff < 86400000) {
    return Math.floor(diff / 3600000) + ' 小时前';
  }
  // Less than 7 days
  if (diff < 604800000) {
    return Math.floor(diff / 86400000) + ' 天前';
  }
  
  // Otherwise show date
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit'
  });
};

/**
 * Get file extension from filename
 */
export const getFileExtension = (name: string): string => {
  if (!name || typeof name !== 'string') return '';
  const parts = name.split('.');
  return parts.length > 1 ? parts.pop()?.toLowerCase() || '' : '';
};

/**
 * Get file name from path
 */
export const getFileName = (path: string): string => {
  return path.split('/').filter(p => p).pop() || path;
};

/**
 * Get parent directory from path
 */
export const getParentPath = (path: string): string => {
  const parts = path.split('/').filter(p => p);
  parts.pop();
  return '/' + parts.join('/');
};

/**
 * Join paths
 */
export const joinPath = (...paths: string[]): string => {
  return paths.join('/').replace(/\/+/g, '/');
};

/**
 * Normalize path
 */
export const normalizePath = (path: string): string => {
  if (!path) return '/';
  const normalized = '/' + path.split('/').filter(p => p).join('/');
  return normalized || '/';
};

/**
 * Generate unique ID
 */
export const generateId = (): string => {
  return Date.now().toString(36) + Math.random().toString(36).substring(2);
};

/**
 * Check if file is previewable
 */
export const getPreviewType = (name: string, mimeType?: string): 'image' | 'video' | 'audio' | 'text' | 'pdf' | null => {
  const ext = getFileExtension(name);
  
  // Images
  if (mimeType?.startsWith('image/') || ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg'].includes(ext)) {
    return 'image';
  }
  
  // Videos
  if (mimeType?.startsWith('video/') || ['mp4', 'webm', 'ogg'].includes(ext)) {
    return 'video';
  }
  
  // Audio
  if (mimeType?.startsWith('audio/') || ['mp3', 'wav', 'ogg', 'm4a'].includes(ext)) {
    return 'audio';
  }
  
  // Text/Code
  if (['txt', 'md', 'json', 'xml', 'yaml', 'yml', 'csv', 'log', 'ini', 'conf', 'js', 'ts', 'py', 'go', 'java', 'html', 'css'].includes(ext)) {
    return 'text';
  }
  
  // PDF
  if (ext === 'pdf') {
    return 'pdf';
  }
  
  return null;
};

/**
 * Get file icon and category based on type or extension
 */
export interface FileIconInfo {
  icon: string;
  category: 'folder' | 'image' | 'video' | 'audio' | 'document' | 'code' | 'archive' | 'default';
}

export const getFileIcon = (type: 'file' | 'directory', name?: string, mimeType?: string): FileIconInfo => {
  if (type === 'directory') {
    return { icon: 'folder', category: 'folder' };
  }

  const ext = getFileExtension(name || '');

  // Image files
  if (mimeType?.startsWith('image/') || ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg', 'ico'].includes(ext)) {
    return { icon: 'file-image', category: 'image' };
  }

  // Video files
  if (mimeType?.startsWith('video/') || ['mp4', 'mkv', 'avi', 'mov', 'wmv', 'flv', 'webm'].includes(ext)) {
    return { icon: 'video-camera', category: 'video' };
  }

  // Audio files
  if (mimeType?.startsWith('audio/') || ['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a'].includes(ext)) {
    return { icon: 'sound', category: 'audio' };
  }

  // Document files
  if (['pdf'].includes(ext)) {
    return { icon: 'file-pdf', category: 'document' };
  }
  if (['doc', 'docx'].includes(ext)) {
    return { icon: 'file-word', category: 'document' };
  }
  if (['xls', 'xlsx'].includes(ext)) {
    return { icon: 'file-excel', category: 'document' };
  }
  if (['ppt', 'pptx'].includes(ext)) {
    return { icon: 'file-ppt', category: 'document' };
  }

  // Code files
  if (['js', 'ts', 'jsx', 'tsx', 'py', 'go', 'java', 'c', 'cpp', 'h', 'cs', 'php', 'rb', 'rs', 'swift'].includes(ext)) {
    return { icon: 'code', category: 'code' };
  }

  // Text files
  if (['txt', 'md', 'json', 'xml', 'yaml', 'yml', 'csv', 'log', 'ini', 'conf', 'html', 'css'].includes(ext)) {
    return { icon: 'file-text', category: 'document' };
  }

  // Archive files
  if (['zip', 'rar', '7z', 'tar', 'gz', 'bz2'].includes(ext)) {
    return { icon: 'file-zip', category: 'archive' };
  }

  // Default
  return { icon: 'file', category: 'default' };
};
