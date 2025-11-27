/**
 * API Client for OneDrive Cloud Storage
 * 
 * This web application is separate from the middleware.
 * Configure API_BASE_URL in index.html to point to your middleware API endpoint.
 */

// Configure the middleware API endpoint
// Priority: localStorage > window.API_BASE_URL > default
const getApiBaseUrl = () => {
    // Remove trailing slash if present
    const url = localStorage.getItem('api_base_url') || window.API_BASE_URL || 'http://localhost:8080/api/v1';
    return url.replace(/\/+$/, '');
};

class CloudAPI {
    constructor() {
        this.currentBucket = 'default';
    }

    // Set current bucket
    setBucket(bucket) {
        this.currentBucket = bucket;
    }

    // Generic request method
    async request(method, path, options = {}) {
        const API_BASE = getApiBaseUrl();
        const url = `${API_BASE}${path}`;
        const config = {
            method,
            headers: {
                ...options.headers
            }
        };

        if (options.body) {
            if (options.body instanceof FormData) {
                config.body = options.body;
            } else if (typeof options.body === 'object') {
                config.headers['Content-Type'] = 'application/json';
                config.body = JSON.stringify(options.body);
            } else {
                config.body = options.body;
            }
        }

        try {
            const response = await fetch(url, config);
            
            if (!response.ok) {
                const error = await response.json().catch(() => ({ message: response.statusText }));
                throw new Error(error.message || error.error || 'Request failed');
            }

            // Handle 204 No Content
            if (response.status === 204) {
                return null;
            }

            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                return response.json();
            }

            return response.blob();
        } catch (error) {
            console.error('API Error:', error);
            if (error.name === 'TypeError' && error.message === 'Failed to fetch') {
                throw new Error('无法连接到服务器，请检查网络或 API 地址配置');
            }
            throw error;
        }
    }

    // ============================================
    // Health & Info
    // ============================================

    async health() {
        return this.request('GET', '/health');
    }

    async info() {
        return this.request('GET', '/info');
    }

    // ============================================
    // Bucket Operations
    // ============================================

    async listBuckets() {
        return this.request('GET', '/buckets');
    }

    async createBucket(name) {
        return this.request('PUT', `/buckets/${encodeURIComponent(name)}`);
    }

    async deleteBucket(name) {
        return this.request('DELETE', `/buckets/${encodeURIComponent(name)}`);
    }

    // ============================================
    // VFS Operations (Virtual File System)
    // ============================================

    async listDirectory(path = '/') {
        const bucket = this.currentBucket;
        // Ensure path ends with / for directory listing
        const normalizedPath = path.endsWith('/') ? path : path + '/';
        const encodedPath = normalizedPath.split('/').map(p => encodeURIComponent(p)).join('/');
        return this.request('GET', `/vfs/${encodeURIComponent(bucket)}${encodedPath}`);
    }

    async createDirectory(path) {
        const bucket = this.currentBucket;
        return this.request('POST', `/vfs/${encodeURIComponent(bucket)}/_mkdir`, {
            body: { path }
        });
    }

    async uploadFile(path, file, onProgress) {
        const bucket = this.currentBucket;
        const encodedPath = path.split('/').map(p => encodeURIComponent(p)).join('/');
        const API_BASE = getApiBaseUrl();
        const url = `${API_BASE}/vfs/${encodeURIComponent(bucket)}${encodedPath}`;

        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();
            
            xhr.open('PUT', url, true);
            xhr.setRequestHeader('Content-Type', file.type || 'application/octet-stream');
            
            xhr.upload.onprogress = (e) => {
                if (e.lengthComputable && onProgress) {
                    const percent = Math.round((e.loaded / e.total) * 100);
                    onProgress(percent);
                }
            };

            xhr.onload = () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    try {
                        resolve(JSON.parse(xhr.responseText));
                    } catch {
                        resolve(null);
                    }
                } else {
                    reject(new Error(xhr.statusText || 'Upload failed'));
                }
            };

            xhr.onerror = () => reject(new Error('Network error'));
            xhr.onabort = () => reject(new Error('Upload cancelled'));

            xhr.send(file);
        });
    }

    async downloadFile(path) {
        const bucket = this.currentBucket;
        const encodedPath = path.split('/').map(p => encodeURIComponent(p)).join('/');
        const API_BASE = getApiBaseUrl();
        const url = `${API_BASE}/vfs/${encodeURIComponent(bucket)}${encodedPath}`;
        
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error('Download failed');
        }
        return response.blob();
    }

    getFileUrl(path) {
        const bucket = this.currentBucket;
        const encodedPath = path.split('/').map(p => encodeURIComponent(p)).join('/');
        const API_BASE = getApiBaseUrl();
        return `${API_BASE}/vfs/${encodeURIComponent(bucket)}${encodedPath}`;
    }

    async deleteFile(path) {
        const bucket = this.currentBucket;
        const encodedPath = path.split('/').map(p => encodeURIComponent(p)).join('/');
        return this.request('DELETE', `/vfs/${encodeURIComponent(bucket)}${encodedPath}`);
    }

    async deleteDirectory(path, recursive = true) {
        const bucket = this.currentBucket;
        const normalizedPath = path.endsWith('/') ? path : path + '/';
        const encodedPath = normalizedPath.split('/').map(p => encodeURIComponent(p)).join('/');
        return this.request('DELETE', `/vfs/${encodeURIComponent(bucket)}${encodedPath}?type=directory&recursive=${recursive}`);
    }

    async moveFile(source, destination) {
        const bucket = this.currentBucket;
        return this.request('POST', `/vfs/${encodeURIComponent(bucket)}/_move`, {
            body: { source, destination }
        });
    }

    async copyFile(source, destination) {
        const bucket = this.currentBucket;
        return this.request('POST', `/vfs/${encodeURIComponent(bucket)}/_copy`, {
            body: { source, destination }
        });
    }

    // ============================================
    // Space & Account Operations
    // ============================================

    async getSpaceOverview() {
        return this.request('GET', '/space');
    }

    async listAccounts() {
        return this.request('GET', '/accounts');
    }

    async syncAccount(id) {
        return this.request('POST', `/accounts/${id}/sync`);
    }
}

// Export singleton instance
const api = new CloudAPI();
