/**
 * Utility Functions for Cloud Storage App
 */

const Utils = {
    /**
     * Format file size to human readable format
     */
    formatSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    },

    /**
     * Format date to locale string
     */
    formatDate(dateString) {
        if (!dateString) return '-';
        const date = new Date(dateString);
        const now = new Date();
        const diff = now - date;
        
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
    },

    /**
     * Get file icon class based on type or extension
     */
    getFileIcon(item) {
        if (item.type === 'directory') {
            return { icon: 'fa-folder', category: 'folder' };
        }

        const name = item.name || '';
        const mimeType = item.mime_type || '';
        const ext = name.split('.').pop().toLowerCase();

        // Image files
        if (mimeType.startsWith('image/') || ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg', 'ico'].includes(ext)) {
            return { icon: 'fa-file-image', category: 'image' };
        }

        // Video files
        if (mimeType.startsWith('video/') || ['mp4', 'mkv', 'avi', 'mov', 'wmv', 'flv', 'webm'].includes(ext)) {
            return { icon: 'fa-file-video', category: 'video' };
        }

        // Audio files
        if (mimeType.startsWith('audio/') || ['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a'].includes(ext)) {
            return { icon: 'fa-file-audio', category: 'audio' };
        }

        // Document files
        if (['pdf'].includes(ext)) {
            return { icon: 'fa-file-pdf', category: 'document' };
        }
        if (['doc', 'docx'].includes(ext)) {
            return { icon: 'fa-file-word', category: 'document' };
        }
        if (['xls', 'xlsx'].includes(ext)) {
            return { icon: 'fa-file-excel', category: 'document' };
        }
        if (['ppt', 'pptx'].includes(ext)) {
            return { icon: 'fa-file-powerpoint', category: 'document' };
        }

        // Code files
        if (['js', 'ts', 'jsx', 'tsx', 'py', 'go', 'java', 'c', 'cpp', 'h', 'cs', 'php', 'rb', 'rs', 'swift'].includes(ext)) {
            return { icon: 'fa-file-code', category: 'code' };
        }

        // Text files
        if (['txt', 'md', 'json', 'xml', 'yaml', 'yml', 'csv', 'log', 'ini', 'conf', 'html', 'css'].includes(ext)) {
            return { icon: 'fa-file-alt', category: 'document' };
        }

        // Archive files
        if (['zip', 'rar', '7z', 'tar', 'gz', 'bz2'].includes(ext)) {
            return { icon: 'fa-file-archive', category: 'archive' };
        }

        // Default
        return { icon: 'fa-file', category: 'default' };
    },

    /**
     * Check if file is previewable
     */
    isPreviewable(item) {
        if (item.type === 'directory') return false;
        
        const name = item.name || '';
        const mimeType = item.mime_type || '';
        const ext = name.split('.').pop().toLowerCase();

        // Images
        if (mimeType.startsWith('image/') || ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'svg'].includes(ext)) {
            return 'image';
        }

        // Videos
        if (mimeType.startsWith('video/') || ['mp4', 'webm', 'ogg'].includes(ext)) {
            return 'video';
        }

        // Audio
        if (mimeType.startsWith('audio/') || ['mp3', 'wav', 'ogg', 'm4a'].includes(ext)) {
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

        return false;
    },

    /**
     * Get file name from path
     */
    getFileName(path) {
        return path.split('/').filter(p => p).pop() || path;
    },

    /**
     * Get parent directory from path
     */
    getParentPath(path) {
        const parts = path.split('/').filter(p => p);
        parts.pop();
        return '/' + parts.join('/');
    },

    /**
     * Join paths
     */
    joinPath(...paths) {
        return paths.join('/').replace(/\/+/g, '/');
    },

    /**
     * Normalize path
     */
    normalizePath(path) {
        if (!path) return '/';
        path = '/' + path.split('/').filter(p => p).join('/');
        return path || '/';
    },

    /**
     * Generate unique ID
     */
    generateId() {
        return Date.now().toString(36) + Math.random().toString(36).substr(2);
    },

    /**
     * Debounce function
     */
    debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    },

    /**
     * Escape HTML
     */
    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
};
