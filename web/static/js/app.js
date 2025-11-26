// OneDrive Cloud Drive Application
(function() {
    'use strict';

    // API Configuration
    const API_BASE = '/api/v1';

    // Application State
    const state = {
        currentBucket: null,
        currentPath: '/',
        items: [],
        selectedItems: new Set(),
        buckets: [],
        loading: false,
        contextMenuItem: null,
        moveMode: 'move' // 'move' or 'copy'
    };

    // DOM Elements
    const elements = {
        bucketList: document.getElementById('bucket-list'),
        fileList: document.getElementById('file-list'),
        emptyState: document.getElementById('empty-state'),
        breadcrumb: document.getElementById('breadcrumb'),
        loadingOverlay: document.getElementById('loading-overlay'),
        selectionInfo: document.getElementById('selection-info'),
        toastContainer: document.getElementById('toast-container'),

        // Buttons
        btnUpload: document.getElementById('btn-upload'),
        btnCreateFolder: document.getElementById('btn-create-folder'),
        btnCreateBucket: document.getElementById('btn-create-bucket'),
        btnDelete: document.getElementById('btn-delete'),
        btnMove: document.getElementById('btn-move'),
        btnCopy: document.getElementById('btn-copy'),
        fileInput: document.getElementById('file-input'),

        // Modals
        uploadModal: document.getElementById('upload-modal'),
        folderModal: document.getElementById('folder-modal'),
        bucketModal: document.getElementById('bucket-modal'),
        moveModal: document.getElementById('move-modal'),
        uploadList: document.getElementById('upload-list'),

        // Context Menu
        contextMenu: document.getElementById('context-menu')
    };

    // Utility Functions
    function formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    function getFileIcon(item) {
        if (item.type === 'directory') return '📁';
        
        const ext = item.name.split('.').pop().toLowerCase();
        const iconMap = {
            'pdf': '📄',
            'doc': '📝', 'docx': '📝',
            'xls': '📊', 'xlsx': '📊',
            'ppt': '📽️', 'pptx': '📽️',
            'jpg': '🖼️', 'jpeg': '🖼️', 'png': '🖼️', 'gif': '🖼️', 'bmp': '🖼️', 'webp': '🖼️',
            'mp3': '🎵', 'wav': '🎵', 'flac': '🎵', 'aac': '🎵',
            'mp4': '🎬', 'avi': '🎬', 'mkv': '🎬', 'mov': '🎬', 'wmv': '🎬',
            'zip': '📦', 'rar': '📦', '7z': '📦', 'tar': '📦', 'gz': '📦',
            'txt': '📃',
            'html': '🌐', 'css': '🎨', 'js': '📜',
            'json': '📋', 'xml': '📋',
            'py': '🐍', 'go': '🔷', 'java': '☕', 'c': '©️', 'cpp': '©️'
        };
        
        return iconMap[ext] || '📄';
    }

    // Build VFS API URL with proper encoding
    function buildVfsUrl(bucket, path) {
        const encodedBucket = encodeURIComponent(bucket);
        const encodedPath = encodeURIComponent(path);
        return `${API_BASE}/vfs/${encodedBucket}/${encodedPath}`;
    }

    function showLoading(show) {
        state.loading = show;
        elements.loadingOverlay.style.display = show ? 'flex' : 'none';
    }

    function showToast(message, type = 'info') {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        elements.toastContainer.appendChild(toast);
        
        setTimeout(() => {
            toast.style.animation = 'slideIn 0.3s ease reverse';
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }

    function showModal(modal) {
        modal.style.display = 'flex';
    }

    function hideModal(modal) {
        modal.style.display = 'none';
    }

    function hideContextMenu() {
        elements.contextMenu.style.display = 'none';
        state.contextMenuItem = null;
    }

    // API Functions
    async function apiRequest(url, options = {}) {
        try {
            const response = await fetch(API_BASE + url, {
                ...options,
                headers: {
                    ...options.headers
                }
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error?.message || `HTTP Error: ${response.status}`);
            }

            // Handle no content response
            if (response.status === 204) {
                return null;
            }

            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                return await response.json();
            }

            return response;
        } catch (error) {
            console.error('API Error:', error);
            throw error;
        }
    }

    // Bucket Functions
    async function loadBuckets() {
        try {
            const data = await apiRequest('/buckets');
            state.buckets = data.buckets || [];
            renderBuckets();
            
            // Select first bucket if none selected
            if (!state.currentBucket && state.buckets.length > 0) {
                selectBucket(state.buckets[0].name);
            } else if (state.currentBucket) {
                loadDirectory();
            }
        } catch (error) {
            showToast('加载存储桶失败: ' + error.message, 'error');
        }
    }

    function renderBuckets() {
        elements.bucketList.innerHTML = state.buckets.map(bucket => `
            <div class="bucket-item ${bucket.name === state.currentBucket ? 'active' : ''}" 
                 data-bucket="${bucket.name}">
                ${bucket.name}
            </div>
        `).join('');

        // Add click handlers
        elements.bucketList.querySelectorAll('.bucket-item').forEach(item => {
            item.addEventListener('click', () => selectBucket(item.dataset.bucket));
        });
    }

    function selectBucket(bucketName) {
        state.currentBucket = bucketName;
        state.currentPath = '/';
        state.selectedItems.clear();
        renderBuckets();
        loadDirectory();
        updateToolbar();
    }

    async function createBucket(name) {
        try {
            const encodedName = encodeURIComponent(name);
            await apiRequest(`/buckets/${encodedName}`, { method: 'PUT' });
            showToast('存储桶创建成功', 'success');
            await loadBuckets();
            selectBucket(name);
        } catch (error) {
            showToast('创建存储桶失败: ' + error.message, 'error');
        }
    }

    // Directory Functions
    async function loadDirectory() {
        if (!state.currentBucket) return;

        showLoading(true);
        try {
            let path = state.currentPath;
            if (!path.endsWith('/')) path += '/';
            
            const url = buildVfsUrl(state.currentBucket, path) + '?type=directory';
            const data = await apiRequest(url.replace(API_BASE, ''));
            state.items = data.items || [];
            renderFileList();
            renderBreadcrumb();
        } catch (error) {
            // If directory not found, show empty state
            if (error.message.includes('not found') || error.message.includes('Not Found')) {
                state.items = [];
                renderFileList();
                renderBreadcrumb();
            } else {
                showToast('加载目录失败: ' + error.message, 'error');
            }
        } finally {
            showLoading(false);
        }
    }

    function renderFileList() {
        if (state.items.length === 0) {
            elements.fileList.style.display = 'none';
            elements.emptyState.style.display = 'flex';
            return;
        }

        elements.emptyState.style.display = 'none';
        elements.fileList.style.display = 'grid';

        // Sort: directories first, then files
        const sorted = [...state.items].sort((a, b) => {
            if (a.type === 'directory' && b.type !== 'directory') return -1;
            if (a.type !== 'directory' && b.type === 'directory') return 1;
            return a.name.localeCompare(b.name);
        });

        elements.fileList.innerHTML = sorted.map(item => `
            <div class="file-item ${state.selectedItems.has(item.path) ? 'selected' : ''}" 
                 data-path="${item.path}" data-type="${item.type}" data-name="${item.name}">
                <input type="checkbox" class="file-checkbox" 
                       ${state.selectedItems.has(item.path) ? 'checked' : ''}>
                <div class="file-icon">${getFileIcon(item)}</div>
                <div class="file-name" title="${item.name}">${item.name}</div>
                ${item.type === 'file' ? `<div class="file-size">${formatFileSize(item.size)}</div>` : ''}
            </div>
        `).join('');

        // Add event handlers
        elements.fileList.querySelectorAll('.file-item').forEach(item => {
            // Double click to open
            item.addEventListener('dblclick', (e) => {
                e.preventDefault();
                const path = item.dataset.path;
                const type = item.dataset.type;
                
                if (type === 'directory') {
                    navigateTo(path);
                } else {
                    downloadFile(path);
                }
            });

            // Single click to select
            item.addEventListener('click', (e) => {
                if (e.target.classList.contains('file-checkbox')) {
                    toggleSelection(item.dataset.path);
                } else if (e.ctrlKey || e.metaKey) {
                    toggleSelection(item.dataset.path);
                } else {
                    state.selectedItems.clear();
                    state.selectedItems.add(item.dataset.path);
                    renderFileList();
                    updateToolbar();
                }
            });

            // Right click for context menu
            item.addEventListener('contextmenu', (e) => {
                e.preventDefault();
                showContextMenu(e, item);
            });
        });
    }

    function toggleSelection(path) {
        if (state.selectedItems.has(path)) {
            state.selectedItems.delete(path);
        } else {
            state.selectedItems.add(path);
        }
        renderFileList();
        updateToolbar();
    }

    function updateToolbar() {
        const count = state.selectedItems.size;
        elements.selectionInfo.textContent = count > 0 ? `已选择 ${count} 项` : '';
        elements.btnDelete.disabled = count === 0;
        elements.btnMove.disabled = count === 0;
        elements.btnCopy.disabled = count === 0;
    }

    function renderBreadcrumb() {
        const parts = state.currentPath.split('/').filter(p => p);
        let path = '/';
        
        let html = `<span class="breadcrumb-item" data-path="/">根目录</span>`;
        
        parts.forEach((part, i) => {
            path += part + '/';
            html += `<span class="breadcrumb-separator">/</span>`;
            html += `<span class="breadcrumb-item" data-path="${path}">${part}</span>`;
        });
        
        elements.breadcrumb.innerHTML = html;
        
        // Add click handlers
        elements.breadcrumb.querySelectorAll('.breadcrumb-item').forEach(item => {
            item.addEventListener('click', () => navigateTo(item.dataset.path));
        });
    }

    function navigateTo(path) {
        state.currentPath = path;
        state.selectedItems.clear();
        updateToolbar();
        loadDirectory();
    }

    // File Operations
    async function uploadFiles(files) {
        if (!state.currentBucket) {
            showToast('请先选择存储桶', 'error');
            return;
        }

        showModal(elements.uploadModal);
        elements.uploadList.innerHTML = '';

        for (const file of files) {
            const itemId = 'upload-' + Date.now() + Math.random();
            const path = state.currentPath + (state.currentPath.endsWith('/') ? '' : '/') + file.name;
            
            elements.uploadList.innerHTML += `
                <div class="upload-item" id="${itemId}">
                    <div class="upload-item-name">${file.name}</div>
                    <div class="upload-progress">
                        <div class="upload-progress-bar" style="width: 0%"></div>
                    </div>
                    <div class="upload-status">准备上传...</div>
                </div>
            `;

            try {
                const uploadItem = document.getElementById(itemId);
                const progressBar = uploadItem.querySelector('.upload-progress-bar');
                const statusEl = uploadItem.querySelector('.upload-status');

                statusEl.textContent = '上传中...';
                progressBar.style.width = '50%';

                const uploadUrl = buildVfsUrl(state.currentBucket, path);
                await fetch(uploadUrl, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': file.type || 'application/octet-stream'
                    },
                    body: file
                }).then(async response => {
                    if (!response.ok) {
                        const error = await response.json();
                        throw new Error(error.error?.message || 'Upload failed');
                    }
                    return response.json();
                });

                progressBar.style.width = '100%';
                statusEl.textContent = '上传成功';
                statusEl.classList.add('success');
            } catch (error) {
                const uploadItem = document.getElementById(itemId);
                const statusEl = uploadItem.querySelector('.upload-status');
                statusEl.textContent = '上传失败: ' + error.message;
                statusEl.classList.add('error');
            }
        }

        loadDirectory();
        showToast('上传完成', 'success');
    }

    async function downloadFile(path) {
        try {
            const downloadUrl = buildVfsUrl(state.currentBucket, path);
            const response = await fetch(downloadUrl);
            
            if (!response.ok) {
                throw new Error('Download failed');
            }

            const blob = await response.blob();
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = path.split('/').pop();
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            window.URL.revokeObjectURL(url);
        } catch (error) {
            showToast('下载失败: ' + error.message, 'error');
        }
    }

    async function deleteItems(paths) {
        if (!confirm(`确定要删除 ${paths.length} 个项目吗？此操作不可撤销。`)) {
            return;
        }

        showLoading(true);
        let successCount = 0;
        let failCount = 0;

        for (const path of paths) {
            try {
                const isDir = path.endsWith('/');
                const deleteUrl = buildVfsUrl(state.currentBucket, path);
                await apiRequest(deleteUrl.replace(API_BASE, '') + (isDir ? '?recursive=true' : ''), {
                    method: 'DELETE'
                });
                successCount++;
            } catch (error) {
                console.error('Delete error:', error);
                failCount++;
            }
        }

        showLoading(false);
        state.selectedItems.clear();
        updateToolbar();

        if (failCount === 0) {
            showToast(`成功删除 ${successCount} 个项目`, 'success');
        } else {
            showToast(`删除完成：成功 ${successCount}，失败 ${failCount}`, 'error');
        }

        loadDirectory();
    }

    async function createFolder(name) {
        if (!state.currentBucket) {
            showToast('请先选择存储桶', 'error');
            return;
        }

        try {
            const path = state.currentPath + (state.currentPath.endsWith('/') ? '' : '/') + name;
            const encodedBucket = encodeURIComponent(state.currentBucket);
            await apiRequest(`/vfs/${encodedBucket}/_mkdir`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ path: path })
            });
            showToast('文件夹创建成功', 'success');
            loadDirectory();
        } catch (error) {
            showToast('创建文件夹失败: ' + error.message, 'error');
        }
    }

    async function moveItem(source, destination, copy = false) {
        try {
            const endpoint = copy ? '_copy' : '_move';
            const encodedBucket = encodeURIComponent(state.currentBucket);
            await apiRequest(`/vfs/${encodedBucket}/${endpoint}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ source, destination })
            });
            showToast(copy ? '复制成功' : '移动成功', 'success');
            loadDirectory();
        } catch (error) {
            showToast((copy ? '复制' : '移动') + '失败: ' + error.message, 'error');
        }
    }

    // Context Menu
    function showContextMenu(event, item) {
        state.contextMenuItem = {
            path: item.dataset.path,
            type: item.dataset.type,
            name: item.dataset.name
        };

        const menu = elements.contextMenu;
        menu.style.display = 'block';
        menu.style.left = event.pageX + 'px';
        menu.style.top = event.pageY + 'px';

        // Adjust position if menu goes off screen
        const rect = menu.getBoundingClientRect();
        if (rect.right > window.innerWidth) {
            menu.style.left = (event.pageX - rect.width) + 'px';
        }
        if (rect.bottom > window.innerHeight) {
            menu.style.top = (event.pageY - rect.height) + 'px';
        }
    }

    // Event Listeners
    function initEventListeners() {
        // Upload button
        elements.btnUpload.addEventListener('click', () => {
            elements.fileInput.click();
        });

        elements.fileInput.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                uploadFiles(e.target.files);
            }
            e.target.value = '';
        });

        // Create folder button
        elements.btnCreateFolder.addEventListener('click', () => {
            if (!state.currentBucket) {
                showToast('请先选择存储桶', 'error');
                return;
            }
            document.getElementById('folder-name-input').value = '';
            showModal(elements.folderModal);
        });

        // Create bucket button
        elements.btnCreateBucket.addEventListener('click', () => {
            document.getElementById('bucket-name-input').value = '';
            showModal(elements.bucketModal);
        });

        // Delete button
        elements.btnDelete.addEventListener('click', () => {
            if (state.selectedItems.size > 0) {
                deleteItems(Array.from(state.selectedItems));
            }
        });

        // Move button
        elements.btnMove.addEventListener('click', () => {
            if (state.selectedItems.size > 0) {
                state.moveMode = 'move';
                document.getElementById('move-modal-title').textContent = '移动到';
                document.getElementById('destination-input').value = '';
                showModal(elements.moveModal);
            }
        });

        // Copy button
        elements.btnCopy.addEventListener('click', () => {
            if (state.selectedItems.size > 0) {
                state.moveMode = 'copy';
                document.getElementById('move-modal-title').textContent = '复制到';
                document.getElementById('destination-input').value = '';
                showModal(elements.moveModal);
            }
        });

        // Modal close buttons
        document.getElementById('close-upload-modal').addEventListener('click', () => hideModal(elements.uploadModal));
        document.getElementById('close-folder-modal').addEventListener('click', () => hideModal(elements.folderModal));
        document.getElementById('close-bucket-modal').addEventListener('click', () => hideModal(elements.bucketModal));
        document.getElementById('close-move-modal').addEventListener('click', () => hideModal(elements.moveModal));

        // Folder modal
        document.getElementById('cancel-folder-btn').addEventListener('click', () => hideModal(elements.folderModal));
        document.getElementById('confirm-folder-btn').addEventListener('click', () => {
            const name = document.getElementById('folder-name-input').value.trim();
            if (name) {
                createFolder(name);
                hideModal(elements.folderModal);
            } else {
                showToast('请输入文件夹名称', 'error');
            }
        });

        // Bucket modal
        document.getElementById('cancel-bucket-btn').addEventListener('click', () => hideModal(elements.bucketModal));
        document.getElementById('confirm-bucket-btn').addEventListener('click', () => {
            const name = document.getElementById('bucket-name-input').value.trim().toLowerCase();
            if (name && /^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$/.test(name)) {
                createBucket(name);
                hideModal(elements.bucketModal);
            } else {
                showToast('存储桶名称格式不正确', 'error');
            }
        });

        // Move modal
        document.getElementById('cancel-move-btn').addEventListener('click', () => hideModal(elements.moveModal));
        document.getElementById('confirm-move-btn').addEventListener('click', async () => {
            const destination = document.getElementById('destination-input').value.trim();
            if (!destination) {
                showToast('请输入目标路径', 'error');
                return;
            }

            hideModal(elements.moveModal);
            showLoading(true);

            const isCopy = state.moveMode === 'copy';
            for (const source of state.selectedItems) {
                const fileName = source.split('/').pop();
                const destPath = destination.endsWith('/') ? destination + fileName : destination + '/' + fileName;
                await moveItem(source, destPath, isCopy);
            }

            state.selectedItems.clear();
            updateToolbar();
            showLoading(false);
        });

        // Context menu handlers
        document.getElementById('ctx-download').addEventListener('click', () => {
            if (state.contextMenuItem && state.contextMenuItem.type === 'file') {
                downloadFile(state.contextMenuItem.path);
            }
            hideContextMenu();
        });

        document.getElementById('ctx-rename').addEventListener('click', () => {
            if (state.contextMenuItem) {
                const newName = prompt('输入新名称:', state.contextMenuItem.name);
                if (newName && newName !== state.contextMenuItem.name) {
                    const parentPath = state.contextMenuItem.path.substring(0, state.contextMenuItem.path.lastIndexOf('/') + 1);
                    const isDir = state.contextMenuItem.type === 'directory';
                    moveItem(
                        state.contextMenuItem.path + (isDir ? '/' : ''),
                        parentPath + newName + (isDir ? '/' : '')
                    );
                }
            }
            hideContextMenu();
        });

        document.getElementById('ctx-move').addEventListener('click', () => {
            if (state.contextMenuItem) {
                state.selectedItems.clear();
                state.selectedItems.add(state.contextMenuItem.path);
                state.moveMode = 'move';
                document.getElementById('move-modal-title').textContent = '移动到';
                document.getElementById('destination-input').value = '';
                showModal(elements.moveModal);
            }
            hideContextMenu();
        });

        document.getElementById('ctx-copy').addEventListener('click', () => {
            if (state.contextMenuItem) {
                state.selectedItems.clear();
                state.selectedItems.add(state.contextMenuItem.path);
                state.moveMode = 'copy';
                document.getElementById('move-modal-title').textContent = '复制到';
                document.getElementById('destination-input').value = '';
                showModal(elements.moveModal);
            }
            hideContextMenu();
        });

        document.getElementById('ctx-delete').addEventListener('click', () => {
            if (state.contextMenuItem) {
                const path = state.contextMenuItem.path + (state.contextMenuItem.type === 'directory' ? '/' : '');
                deleteItems([path]);
            }
            hideContextMenu();
        });

        // Hide context menu on click outside
        document.addEventListener('click', (e) => {
            if (!elements.contextMenu.contains(e.target)) {
                hideContextMenu();
            }
        });

        // Modal click outside to close
        [elements.uploadModal, elements.folderModal, elements.bucketModal, elements.moveModal].forEach(modal => {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    hideModal(modal);
                }
            });
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                hideContextMenu();
                hideModal(elements.uploadModal);
                hideModal(elements.folderModal);
                hideModal(elements.bucketModal);
                hideModal(elements.moveModal);
            }
            
            if (e.key === 'Delete' && state.selectedItems.size > 0) {
                deleteItems(Array.from(state.selectedItems));
            }
        });

        // Drag and drop
        const dropZone = document.querySelector('.file-browser');
        
        dropZone.addEventListener('dragover', (e) => {
            e.preventDefault();
            dropZone.style.background = 'rgba(0, 120, 212, 0.1)';
        });

        dropZone.addEventListener('dragleave', () => {
            dropZone.style.background = '';
        });

        dropZone.addEventListener('drop', (e) => {
            e.preventDefault();
            dropZone.style.background = '';
            
            if (!state.currentBucket) {
                showToast('请先选择存储桶', 'error');
                return;
            }

            if (e.dataTransfer.files.length > 0) {
                uploadFiles(e.dataTransfer.files);
            }
        });
    }

    // Initialize Application
    function init() {
        initEventListeners();
        loadBuckets();
    }

    // Start the app when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
