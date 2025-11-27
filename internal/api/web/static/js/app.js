/**
 * Main Application for Cloud Storage
 */

class CloudStorageApp {
    constructor() {
        this.currentPath = '/';
        this.currentView = 'grid'; // 'grid' or 'list'
        this.selectedItems = new Set();
        this.clipboard = null; // { action: 'copy'|'move', items: [...] }
        this.contextMenuItem = null;
        this.moveCopyAction = null;
        this.currentBucket = 'default';
        this.items = [];
        
        this.init();
    }

    async init() {
        this.bindEvents();
        await this.initializeBucket();
        await this.loadDirectory('/');
        await this.loadStorageInfo();
    }

    async initializeBucket() {
        try {
            const response = await api.listBuckets();
            const buckets = response.buckets || [];
            
            if (buckets.length === 0) {
                // Create default bucket
                await api.createBucket('default');
                this.currentBucket = 'default';
            } else {
                this.currentBucket = buckets[0].name;
            }
            
            api.setBucket(this.currentBucket);
        } catch (error) {
            console.error('Failed to initialize bucket:', error);
            // Try to create default bucket anyway
            try {
                await api.createBucket('default');
                this.currentBucket = 'default';
                api.setBucket(this.currentBucket);
            } catch (e) {
                console.error('Failed to create default bucket:', e);
            }
        }
    }

    bindEvents() {
        // Sidebar toggle
        document.getElementById('sidebarToggle').addEventListener('click', () => {
            document.getElementById('sidebar').classList.toggle('collapsed');
        });

        // Mobile menu
        document.getElementById('mobileMenuBtn').addEventListener('click', () => {
            document.getElementById('sidebar').classList.toggle('mobile-open');
        });

        // Close mobile menu when clicking outside
        document.querySelector('.main-content').addEventListener('click', () => {
            document.getElementById('sidebar').classList.remove('mobile-open');
        });

        // Navigation items
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
                e.currentTarget.classList.add('active');
                const view = e.currentTarget.dataset.view;
                this.handleNavigation(view);
            });
        });

        // View toggle
        document.getElementById('viewToggle').addEventListener('click', () => {
            this.toggleView();
        });

        // Refresh button
        document.getElementById('refreshBtn').addEventListener('click', () => {
            this.loadDirectory(this.currentPath);
        });

        // Search
        document.getElementById('searchInput').addEventListener('input', 
            Utils.debounce((e) => this.handleSearch(e.target.value), 300)
        );

        // Upload button
        document.getElementById('uploadBtn').addEventListener('click', () => {
            document.getElementById('fileInput').click();
        });

        // File input change
        document.getElementById('fileInput').addEventListener('change', (e) => {
            this.handleFileUpload(e.target.files);
            e.target.value = ''; // Reset input
        });

        // New folder button
        document.getElementById('newFolderBtn').addEventListener('click', () => {
            this.showModal('newFolderModal');
            document.getElementById('folderName').value = '';
            document.getElementById('folderName').focus();
        });

        // Create folder
        document.getElementById('createFolderBtn').addEventListener('click', () => {
            this.createFolder();
        });

        // Folder name enter key
        document.getElementById('folderName').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.createFolder();
            }
        });

        // Settings button
        document.getElementById('settingsBtn').addEventListener('click', () => {
            this.showSettings();
        });

        // Modal close buttons
        document.querySelectorAll('.modal-close').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const modal = e.target.closest('.modal');
                this.hideModal(modal.id);
            });
        });

        // Close modals on backdrop click
        document.querySelectorAll('.modal').forEach(modal => {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    this.hideModal(modal.id);
                }
            });
        });

        // Selection actions
        document.getElementById('downloadSelectedBtn').addEventListener('click', () => {
            this.downloadSelected();
        });

        document.getElementById('moveSelectedBtn').addEventListener('click', () => {
            this.showMoveCopyDialog('move');
        });

        document.getElementById('copySelectedBtn').addEventListener('click', () => {
            this.showMoveCopyDialog('copy');
        });

        document.getElementById('deleteSelectedBtn').addEventListener('click', () => {
            this.showDeleteConfirmation();
        });

        document.getElementById('cancelSelectionBtn').addEventListener('click', () => {
            this.clearSelection();
        });

        // Confirm delete
        document.getElementById('confirmDeleteBtn').addEventListener('click', () => {
            this.deleteSelected();
        });

        // Confirm rename
        document.getElementById('confirmRenameBtn').addEventListener('click', () => {
            this.confirmRename();
        });

        // New name enter key
        document.getElementById('newName').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.confirmRename();
            }
        });

        // Confirm move/copy
        document.getElementById('confirmMoveCopyBtn').addEventListener('click', () => {
            this.confirmMoveCopy();
        });

        // Create bucket
        document.getElementById('createBucketBtn').addEventListener('click', () => {
            this.createBucket();
        });

        // Upload panel close
        document.getElementById('closeUploadPanel').addEventListener('click', () => {
            document.getElementById('uploadPanel').classList.remove('visible');
        });

        // Context menu
        document.addEventListener('click', () => {
            this.hideContextMenu();
        });

        document.getElementById('contextMenu').addEventListener('click', (e) => {
            const item = e.target.closest('.context-menu-item');
            if (item) {
                this.handleContextAction(item.dataset.action);
            }
        });

        // Drag and drop
        this.setupDragAndDrop();

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            this.handleKeyboard(e);
        });

        // Preview download
        document.getElementById('previewDownloadBtn').addEventListener('click', () => {
            if (this.contextMenuItem) {
                this.downloadFile(this.contextMenuItem);
            }
        });
    }

    setupDragAndDrop() {
        const fileContainer = document.querySelector('.main-content');
        const dropZone = document.getElementById('dropZone');

        ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
            fileContainer.addEventListener(eventName, (e) => {
                e.preventDefault();
                e.stopPropagation();
            });
        });

        ['dragenter', 'dragover'].forEach(eventName => {
            fileContainer.addEventListener(eventName, () => {
                dropZone.classList.add('active');
            });
        });

        ['dragleave', 'drop'].forEach(eventName => {
            dropZone.addEventListener(eventName, () => {
                dropZone.classList.remove('active');
            });
        });

        dropZone.addEventListener('drop', (e) => {
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                this.handleFileUpload(files);
            }
        });
    }

    handleKeyboard(e) {
        // Escape - clear selection or close modal
        if (e.key === 'Escape') {
            if (document.querySelector('.modal.visible')) {
                const modal = document.querySelector('.modal.visible');
                this.hideModal(modal.id);
            } else {
                this.clearSelection();
            }
        }

        // Delete - delete selected items
        if (e.key === 'Delete' && this.selectedItems.size > 0) {
            this.showDeleteConfirmation();
        }

        // Ctrl+A - select all
        if (e.key === 'a' && (e.ctrlKey || e.metaKey)) {
            e.preventDefault();
            this.selectAll();
        }
    }

    // ============================================
    // Navigation
    // ============================================

    handleNavigation(view) {
        switch (view) {
            case 'files':
                this.loadDirectory('/');
                break;
            case 'recent':
            case 'starred':
            case 'trash':
                this.showToast('info', '该功能即将推出');
                break;
        }
    }

    updateBreadcrumb() {
        const breadcrumb = document.getElementById('breadcrumb');
        const parts = this.currentPath.split('/').filter(p => p);
        
        let html = `
            <span class="breadcrumb-item${parts.length === 0 ? ' active' : ''}" data-path="/">
                <i class="fas fa-home"></i>
            </span>
        `;

        let currentPath = '';
        parts.forEach((part, index) => {
            currentPath += '/' + part;
            const isLast = index === parts.length - 1;
            html += `
                <span class="breadcrumb-separator">/</span>
                <span class="breadcrumb-item${isLast ? ' active' : ''}" data-path="${currentPath}">
                    ${Utils.escapeHtml(part)}
                </span>
            `;
        });

        breadcrumb.innerHTML = html;

        // Bind click events
        breadcrumb.querySelectorAll('.breadcrumb-item').forEach(item => {
            item.addEventListener('click', () => {
                const path = item.dataset.path;
                this.loadDirectory(path);
            });
        });
    }

    // ============================================
    // File List
    // ============================================

    async loadDirectory(path) {
        this.showLoading();
        this.clearSelection();

        try {
            const response = await api.listDirectory(path);
            this.currentPath = Utils.normalizePath(path);
            this.items = response.items || [];
            this.renderFileList();
            this.updateBreadcrumb();
        } catch (error) {
            console.error('Failed to load directory:', error);
            this.showToast('error', '加载目录失败: ' + error.message);
            this.hideLoading();
        }
    }

    showLoading() {
        document.getElementById('loadingState').style.display = 'flex';
        document.getElementById('emptyState').style.display = 'none';
        document.getElementById('fileGrid').innerHTML = '';
    }

    hideLoading() {
        document.getElementById('loadingState').style.display = 'none';
    }

    renderFileList() {
        this.hideLoading();

        const fileGrid = document.getElementById('fileGrid');
        const emptyState = document.getElementById('emptyState');

        if (this.items.length === 0) {
            emptyState.style.display = 'flex';
            fileGrid.innerHTML = '';
            return;
        }

        emptyState.style.display = 'none';

        // Sort: directories first, then by name
        const sortedItems = [...this.items].sort((a, b) => {
            if (a.type === 'directory' && b.type !== 'directory') return -1;
            if (a.type !== 'directory' && b.type === 'directory') return 1;
            return (a.name || '').localeCompare(b.name || '');
        });

        fileGrid.innerHTML = sortedItems.map(item => this.renderFileItem(item)).join('');

        // Update view class
        fileGrid.classList.toggle('list-view', this.currentView === 'list');

        // Bind events to file items
        this.bindFileItemEvents();
    }

    renderFileItem(item) {
        const { icon, category } = Utils.getFileIcon(item);
        const isSelected = this.selectedItems.has(item.path);
        const formattedSize = item.type === 'file' ? Utils.formatSize(item.size || 0) : '';
        const formattedDate = Utils.formatDate(item.created_at);

        if (this.currentView === 'grid') {
            return `
                <div class="file-item ${isSelected ? 'selected' : ''}" data-path="${Utils.escapeHtml(item.path)}" data-type="${item.type}">
                    <div class="file-item-checkbox">
                        ${isSelected ? '<i class="fas fa-check"></i>' : ''}
                    </div>
                    <div class="file-item-icon ${category}">
                        <i class="fas ${icon}"></i>
                    </div>
                    <div class="file-item-info">
                        <div class="file-item-name" title="${Utils.escapeHtml(item.name)}">${Utils.escapeHtml(item.name)}</div>
                        <div class="file-item-meta">${formattedSize || formattedDate}</div>
                    </div>
                    <div class="file-item-actions">
                        <button class="file-item-menu-btn" title="更多操作">
                            <i class="fas fa-ellipsis-v"></i>
                        </button>
                    </div>
                </div>
            `;
        } else {
            return `
                <div class="file-item ${isSelected ? 'selected' : ''}" data-path="${Utils.escapeHtml(item.path)}" data-type="${item.type}">
                    <div class="file-item-checkbox">
                        ${isSelected ? '<i class="fas fa-check"></i>' : ''}
                    </div>
                    <div class="file-item-icon ${category}">
                        <i class="fas ${icon}"></i>
                    </div>
                    <div class="file-item-info">
                        <div class="file-item-name" title="${Utils.escapeHtml(item.name)}">${Utils.escapeHtml(item.name)}</div>
                        <div class="file-item-meta">${formattedSize}</div>
                        <div class="file-item-date">${formattedDate}</div>
                    </div>
                    <div class="file-item-actions">
                        <button class="file-item-menu-btn" title="更多操作">
                            <i class="fas fa-ellipsis-v"></i>
                        </button>
                    </div>
                </div>
            `;
        }
    }

    bindFileItemEvents() {
        document.querySelectorAll('.file-item').forEach(item => {
            // Click to select
            item.addEventListener('click', (e) => {
                if (e.target.closest('.file-item-menu-btn')) return;
                
                if (e.ctrlKey || e.metaKey) {
                    this.toggleSelection(item.dataset.path);
                } else if (e.shiftKey) {
                    this.rangeSelect(item.dataset.path);
                } else {
                    this.clearSelection();
                    this.toggleSelection(item.dataset.path);
                }
            });

            // Double click to open
            item.addEventListener('dblclick', () => {
                const path = item.dataset.path;
                const type = item.dataset.type;
                
                if (type === 'directory') {
                    this.loadDirectory(path);
                } else {
                    this.previewFile(path);
                }
            });

            // Context menu
            item.addEventListener('contextmenu', (e) => {
                e.preventDefault();
                const path = item.dataset.path;
                const itemData = this.items.find(i => i.path === path);
                this.showContextMenu(e.clientX, e.clientY, itemData);
            });

            // Menu button
            item.querySelector('.file-item-menu-btn').addEventListener('click', (e) => {
                e.stopPropagation();
                const path = item.dataset.path;
                const itemData = this.items.find(i => i.path === path);
                const rect = e.target.getBoundingClientRect();
                this.showContextMenu(rect.left, rect.bottom, itemData);
            });
        });
    }

    // ============================================
    // Selection
    // ============================================

    toggleSelection(path) {
        if (this.selectedItems.has(path)) {
            this.selectedItems.delete(path);
        } else {
            this.selectedItems.add(path);
        }
        this.updateSelectionUI();
    }

    rangeSelect(endPath) {
        const paths = this.items.map(i => i.path);
        const startIndex = this.lastSelectedIndex || 0;
        const endIndex = paths.indexOf(endPath);

        if (endIndex === -1) return;

        const start = Math.min(startIndex, endIndex);
        const end = Math.max(startIndex, endIndex);

        for (let i = start; i <= end; i++) {
            this.selectedItems.add(paths[i]);
        }

        this.lastSelectedIndex = endIndex;
        this.updateSelectionUI();
    }

    selectAll() {
        this.items.forEach(item => {
            this.selectedItems.add(item.path);
        });
        this.updateSelectionUI();
    }

    clearSelection() {
        this.selectedItems.clear();
        this.lastSelectedIndex = undefined;
        this.updateSelectionUI();
    }

    updateSelectionUI() {
        // Update file item classes
        document.querySelectorAll('.file-item').forEach(item => {
            const isSelected = this.selectedItems.has(item.dataset.path);
            item.classList.toggle('selected', isSelected);
            item.querySelector('.file-item-checkbox').innerHTML = isSelected ? '<i class="fas fa-check"></i>' : '';
        });

        // Update selection bar
        const selectionBar = document.getElementById('selectionBar');
        if (this.selectedItems.size > 0) {
            selectionBar.classList.add('visible');
            document.getElementById('selectedCount').textContent = this.selectedItems.size;
        } else {
            selectionBar.classList.remove('visible');
        }
    }

    // ============================================
    // View Toggle
    // ============================================

    toggleView() {
        this.currentView = this.currentView === 'grid' ? 'list' : 'grid';
        const icon = this.currentView === 'grid' ? 'fa-th-large' : 'fa-list';
        document.getElementById('viewToggle').innerHTML = `<i class="fas ${icon}"></i>`;
        this.renderFileList();
    }

    // ============================================
    // Search
    // ============================================

    handleSearch(query) {
        if (!query) {
            this.renderFileList();
            return;
        }

        const filtered = this.items.filter(item => 
            item.name.toLowerCase().includes(query.toLowerCase())
        );

        const originalItems = this.items;
        this.items = filtered;
        this.renderFileList();
        this.items = originalItems;
    }

    // ============================================
    // Context Menu
    // ============================================

    showContextMenu(x, y, item) {
        const menu = document.getElementById('contextMenu');
        this.contextMenuItem = item;

        // Position menu
        menu.style.left = x + 'px';
        menu.style.top = y + 'px';

        // Adjust position if off screen
        const rect = menu.getBoundingClientRect();
        if (rect.right > window.innerWidth) {
            menu.style.left = (x - rect.width) + 'px';
        }
        if (rect.bottom > window.innerHeight) {
            menu.style.top = (y - rect.height) + 'px';
        }

        // Show/hide appropriate options
        const isDirectory = item.type === 'directory';
        menu.querySelector('[data-action="open"]').style.display = isDirectory ? 'flex' : 'none';
        menu.querySelector('[data-action="preview"]').style.display = !isDirectory && Utils.isPreviewable(item) ? 'flex' : 'none';
        menu.querySelector('[data-action="download"]').style.display = !isDirectory ? 'flex' : 'none';

        menu.classList.add('visible');
    }

    hideContextMenu() {
        document.getElementById('contextMenu').classList.remove('visible');
    }

    handleContextAction(action) {
        const item = this.contextMenuItem;
        if (!item) return;

        switch (action) {
            case 'open':
                this.loadDirectory(item.path);
                break;
            case 'preview':
                this.previewFile(item.path);
                break;
            case 'download':
                this.downloadFile(item);
                break;
            case 'rename':
                this.showRenameDialog(item);
                break;
            case 'move':
                this.selectedItems.add(item.path);
                this.showMoveCopyDialog('move');
                break;
            case 'copy':
                this.selectedItems.add(item.path);
                this.showMoveCopyDialog('copy');
                break;
            case 'delete':
                this.selectedItems.add(item.path);
                this.showDeleteConfirmation();
                break;
        }

        this.hideContextMenu();
    }

    // ============================================
    // File Operations
    // ============================================

    async handleFileUpload(files) {
        if (files.length === 0) return;

        const uploadPanel = document.getElementById('uploadPanel');
        const uploadList = document.getElementById('uploadList');
        uploadPanel.classList.add('visible');

        for (const file of files) {
            const uploadId = Utils.generateId();
            const path = Utils.joinPath(this.currentPath, file.name);

            // Add upload item to panel
            uploadList.innerHTML += `
                <div class="upload-item" id="upload-${uploadId}">
                    <i class="fas fa-file upload-item-icon"></i>
                    <div class="upload-item-info">
                        <div class="upload-item-name">${Utils.escapeHtml(file.name)}</div>
                        <div class="upload-item-progress">
                            <div class="upload-item-progress-bar" style="width: 0%"></div>
                        </div>
                        <div class="upload-item-status">准备上传...</div>
                    </div>
                </div>
            `;

            try {
                await api.uploadFile(path, file, (percent) => {
                    const item = document.getElementById(`upload-${uploadId}`);
                    if (item) {
                        item.querySelector('.upload-item-progress-bar').style.width = percent + '%';
                        item.querySelector('.upload-item-status').textContent = `上传中 ${percent}%`;
                    }
                });

                const item = document.getElementById(`upload-${uploadId}`);
                if (item) {
                    item.querySelector('.upload-item-status').textContent = '上传完成';
                    item.querySelector('.upload-item-status').classList.add('success');
                }
            } catch (error) {
                const item = document.getElementById(`upload-${uploadId}`);
                if (item) {
                    item.querySelector('.upload-item-status').textContent = '上传失败: ' + error.message;
                    item.querySelector('.upload-item-status').classList.add('error');
                }
            }
        }

        // Refresh file list
        this.loadDirectory(this.currentPath);
        this.showToast('success', `成功上传 ${files.length} 个文件`);
    }

    async createFolder() {
        const name = document.getElementById('folderName').value.trim();
        if (!name) {
            this.showToast('warning', '请输入文件夹名称');
            return;
        }

        try {
            const path = Utils.joinPath(this.currentPath, name);
            await api.createDirectory(path);
            this.hideModal('newFolderModal');
            this.loadDirectory(this.currentPath);
            this.showToast('success', '文件夹创建成功');
        } catch (error) {
            this.showToast('error', '创建文件夹失败: ' + error.message);
        }
    }

    async downloadFile(item) {
        try {
            const blob = await api.downloadFile(item.path);
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = item.name;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        } catch (error) {
            this.showToast('error', '下载失败: ' + error.message);
        }
    }

    async downloadSelected() {
        for (const path of this.selectedItems) {
            const item = this.items.find(i => i.path === path);
            if (item && item.type === 'file') {
                await this.downloadFile(item);
            }
        }
    }

    showRenameDialog(item) {
        document.getElementById('newName').value = item.name;
        this.showModal('renameModal');
        document.getElementById('newName').focus();
        document.getElementById('newName').select();
    }

    async confirmRename() {
        const newName = document.getElementById('newName').value.trim();
        if (!newName) {
            this.showToast('warning', '请输入新名称');
            return;
        }

        const item = this.contextMenuItem;
        if (!item) return;

        try {
            const newPath = Utils.joinPath(Utils.getParentPath(item.path), newName);
            await api.moveFile(item.path, newPath);
            this.hideModal('renameModal');
            this.loadDirectory(this.currentPath);
            this.showToast('success', '重命名成功');
        } catch (error) {
            this.showToast('error', '重命名失败: ' + error.message);
        }
    }

    showDeleteConfirmation() {
        const count = this.selectedItems.size;
        document.getElementById('deleteMessage').textContent = 
            `确定要删除选中的 ${count} 个项目吗？此操作无法撤销。`;
        this.showModal('deleteModal');
    }

    async deleteSelected() {
        const paths = [...this.selectedItems];
        let successCount = 0;
        let failCount = 0;

        for (const path of paths) {
            const item = this.items.find(i => i.path === path);
            if (!item) continue;

            try {
                if (item.type === 'directory') {
                    await api.deleteDirectory(path, true);
                } else {
                    await api.deleteFile(path);
                }
                successCount++;
            } catch (error) {
                console.error('Delete failed:', error);
                failCount++;
            }
        }

        this.hideModal('deleteModal');
        this.clearSelection();
        this.loadDirectory(this.currentPath);

        if (failCount === 0) {
            this.showToast('success', `成功删除 ${successCount} 个项目`);
        } else {
            this.showToast('warning', `成功 ${successCount} 个，失败 ${failCount} 个`);
        }
    }

    showMoveCopyDialog(action) {
        this.moveCopyAction = action;
        document.getElementById('moveCopyTitle').textContent = action === 'move' ? '移动到' : '复制到';
        this.loadFolderTree();
        this.showModal('moveCopyModal');
    }

    async loadFolderTree() {
        const tree = document.getElementById('folderTree');
        tree.innerHTML = '<div class="loading-state"><div class="spinner"></div></div>';

        try {
            // Load root directory
            const response = await api.listDirectory('/');
            const folders = (response.items || []).filter(i => i.type === 'directory');

            let html = `
                <div class="folder-tree-item selected" data-path="/">
                    <i class="fas fa-folder"></i>
                    <span>根目录</span>
                </div>
            `;

            folders.forEach(folder => {
                html += `
                    <div class="folder-tree-item" data-path="${Utils.escapeHtml(folder.path)}">
                        <i class="fas fa-folder"></i>
                        <span>${Utils.escapeHtml(folder.name)}</span>
                    </div>
                `;
            });

            tree.innerHTML = html;

            // Bind click events
            tree.querySelectorAll('.folder-tree-item').forEach(item => {
                item.addEventListener('click', () => {
                    tree.querySelectorAll('.folder-tree-item').forEach(i => i.classList.remove('selected'));
                    item.classList.add('selected');
                });
            });
        } catch (error) {
            tree.innerHTML = '<p>加载失败</p>';
        }
    }

    async confirmMoveCopy() {
        const selectedFolder = document.querySelector('.folder-tree-item.selected');
        if (!selectedFolder) {
            this.showToast('warning', '请选择目标文件夹');
            return;
        }

        const destPath = selectedFolder.dataset.path;
        const action = this.moveCopyAction;
        const paths = [...this.selectedItems];

        let successCount = 0;
        let failCount = 0;

        for (const path of paths) {
            const item = this.items.find(i => i.path === path);
            if (!item) continue;

            const newPath = Utils.joinPath(destPath, item.name);

            try {
                if (action === 'move') {
                    await api.moveFile(path, newPath);
                } else {
                    await api.copyFile(path, newPath);
                }
                successCount++;
            } catch (error) {
                console.error(`${action} failed:`, error);
                failCount++;
            }
        }

        this.hideModal('moveCopyModal');
        this.clearSelection();
        this.loadDirectory(this.currentPath);

        const actionText = action === 'move' ? '移动' : '复制';
        if (failCount === 0) {
            this.showToast('success', `成功${actionText} ${successCount} 个项目`);
        } else {
            this.showToast('warning', `成功 ${successCount} 个，失败 ${failCount} 个`);
        }
    }

    // ============================================
    // Preview
    // ============================================

    async previewFile(path) {
        const item = this.items.find(i => i.path === path);
        if (!item) return;

        this.contextMenuItem = item;
        const previewType = Utils.isPreviewable(item);
        if (!previewType) {
            this.showToast('info', '该文件类型不支持预览');
            return;
        }

        document.getElementById('previewFileName').textContent = item.name;
        const content = document.getElementById('previewContent');

        const fileUrl = api.getFileUrl(path);

        switch (previewType) {
            case 'image':
                content.innerHTML = `<img src="${fileUrl}" alt="${Utils.escapeHtml(item.name)}">`;
                break;
            case 'video':
                content.innerHTML = `<video src="${fileUrl}" controls autoplay></video>`;
                break;
            case 'audio':
                content.innerHTML = `<audio src="${fileUrl}" controls autoplay></audio>`;
                break;
            case 'pdf':
                content.innerHTML = `<iframe src="${fileUrl}" style="width: 100%; height: 100%; border: none;"></iframe>`;
                break;
            case 'text':
                try {
                    const blob = await api.downloadFile(path);
                    const text = await blob.text();
                    content.innerHTML = `<div class="text-preview"><pre>${Utils.escapeHtml(text)}</pre></div>`;
                } catch (error) {
                    content.innerHTML = `<div class="preview-unsupported"><i class="fas fa-exclamation-circle"></i><p>无法加载文件内容</p></div>`;
                }
                break;
            default:
                content.innerHTML = `
                    <div class="preview-unsupported">
                        <i class="fas fa-file"></i>
                        <p>该文件类型不支持预览</p>
                        <button class="btn btn-primary" onclick="app.downloadFile(app.contextMenuItem)">
                            <i class="fas fa-download"></i> 下载文件
                        </button>
                    </div>
                `;
        }

        this.showModal('previewModal');
    }

    // ============================================
    // Settings
    // ============================================

    async showSettings() {
        this.showModal('settingsModal');
        await this.loadBucketList();
        await this.loadAccountList();
    }

    async loadBucketList() {
        const list = document.getElementById('bucketList');
        try {
            const response = await api.listBuckets();
            const buckets = response.buckets || [];

            if (buckets.length === 0) {
                list.innerHTML = '<p>暂无存储桶</p>';
                return;
            }

            list.innerHTML = buckets.map(bucket => `
                <div class="bucket-item">
                    <div>
                        <div class="bucket-item-name">${Utils.escapeHtml(bucket.name)}</div>
                        <div class="bucket-item-info">${bucket.object_count || 0} 个文件 · ${Utils.formatSize(bucket.total_size || 0)}</div>
                    </div>
                    <button class="action-btn danger" onclick="app.deleteBucket('${Utils.escapeHtml(bucket.name)}')" title="删除">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            `).join('');
        } catch (error) {
            list.innerHTML = '<p>加载失败</p>';
        }
    }

    async loadAccountList() {
        const list = document.getElementById('accountList');
        try {
            const accounts = await api.listAccounts();
            
            if (!accounts || accounts.length === 0) {
                list.innerHTML = '<p>暂无账号，请添加 OneDrive 账号</p>';
                return;
            }

            list.innerHTML = accounts.map(account => `
                <div class="account-item">
                    <div>
                        <div class="account-item-name">${Utils.escapeHtml(account.name || account.email)}</div>
                        <div class="account-item-info">
                            ${Utils.formatSize(account.used_space || 0)} / ${Utils.formatSize(account.total_space || 0)}
                            · ${account.status || 'unknown'}
                        </div>
                    </div>
                    <button class="action-btn" onclick="app.syncAccount('${account.id}')" title="同步">
                        <i class="fas fa-sync-alt"></i>
                    </button>
                </div>
            `).join('');
        } catch (error) {
            list.innerHTML = '<p>加载失败</p>';
        }
    }

    async createBucket() {
        const name = document.getElementById('newBucketName').value.trim();
        if (!name) {
            this.showToast('warning', '请输入存储桶名称');
            return;
        }

        try {
            await api.createBucket(name);
            document.getElementById('newBucketName').value = '';
            await this.loadBucketList();
            this.showToast('success', '存储桶创建成功');
        } catch (error) {
            this.showToast('error', '创建失败: ' + error.message);
        }
    }

    async deleteBucket(name) {
        if (!confirm(`确定要删除存储桶 "${name}" 吗？`)) return;

        try {
            await api.deleteBucket(name);
            await this.loadBucketList();
            this.showToast('success', '存储桶删除成功');
        } catch (error) {
            this.showToast('error', '删除失败: ' + error.message);
        }
    }

    async syncAccount(id) {
        try {
            await api.syncAccount(id);
            await this.loadAccountList();
            await this.loadStorageInfo();
            this.showToast('success', '同步成功');
        } catch (error) {
            this.showToast('error', '同步失败: ' + error.message);
        }
    }

    // ============================================
    // Storage Info
    // ============================================

    async loadStorageInfo() {
        try {
            const data = await api.getSpaceOverview();
            const used = data.total_used || 0;
            const total = data.total_space || 0;
            const percent = total > 0 ? (used / total * 100) : 0;

            document.getElementById('storageUsed').textContent = Utils.formatSize(used);
            document.getElementById('storageTotal').textContent = Utils.formatSize(total);
            document.getElementById('storageBar').style.width = percent + '%';
        } catch (error) {
            console.error('Failed to load storage info:', error);
        }
    }

    // ============================================
    // Modal Helpers
    // ============================================

    showModal(id) {
        document.getElementById(id).classList.add('visible');
    }

    hideModal(id) {
        document.getElementById(id).classList.remove('visible');
    }

    // ============================================
    // Toast Notifications
    // ============================================

    showToast(type, message) {
        const container = document.getElementById('toastContainer');
        const id = Utils.generateId();
        
        const icons = {
            success: 'fa-check-circle',
            error: 'fa-times-circle',
            warning: 'fa-exclamation-circle',
            info: 'fa-info-circle'
        };

        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.id = `toast-${id}`;
        toast.innerHTML = `
            <i class="fas ${icons[type]} toast-icon"></i>
            <span class="toast-message">${Utils.escapeHtml(message)}</span>
            <button class="toast-close" onclick="app.hideToast('${id}')">
                <i class="fas fa-times"></i>
            </button>
        `;

        container.appendChild(toast);

        // Auto remove after 5 seconds
        setTimeout(() => {
            this.hideToast(id);
        }, 5000);
    }

    hideToast(id) {
        const toast = document.getElementById(`toast-${id}`);
        if (toast) {
            toast.classList.add('hiding');
            setTimeout(() => {
                toast.remove();
            }, 300);
        }
    }
}

// Initialize app
const app = new CloudStorageApp();
