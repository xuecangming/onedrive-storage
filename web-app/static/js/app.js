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
        this.lastSelectedIndex = -1;
        
        // Wait for DOM to be ready
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => this.init());
        } else {
            this.init();
        }
    }

    async init() {
        console.log('Initializing CloudStorageApp...');
        
        // Check if CSS is loaded
        setTimeout(() => {
            const testEl = document.createElement('div');
            testEl.className = 'sidebar';
            document.body.appendChild(testEl);
            const width = getComputedStyle(testEl).width;
            document.body.removeChild(testEl);
            
            if (width !== '250px') {
                console.warn('CSS not loaded correctly!');
                // Try to reload CSS with cache buster
                const link = document.querySelector('link[href*="style.css"]');
                if (link) {
                    link.href = link.href.split('?')[0] + '?v=' + Date.now();
                }
            }
        }, 1000);

        this.bindEvents();
        
        try {
            await this.initializeBucket();
            await this.loadDirectory('/');
            await this.loadStorageInfo();
        } catch (error) {
            console.error('Initialization failed:', error);
            this.showToast('error', '初始化失败: ' + error.message);
        }
    }

    async initializeBucket() {
        try {
            const response = await api.listBuckets();
            const buckets = response.buckets || [];
            
            if (buckets.length === 0) {
                console.log('No buckets found, creating default bucket...');
                // Create default bucket
                await api.createBucket('default');
                this.currentBucket = 'default';
            } else {
                // Use the first bucket found
                this.currentBucket = buckets[0].name;
                console.log(`Using bucket: ${this.currentBucket}`);
            }
            
            api.setBucket(this.currentBucket);
        } catch (error) {
            console.error('Failed to initialize bucket:', error);
            // Try to create default bucket anyway if listing failed (e.g. 404)
            try {
                await api.createBucket('default');
                this.currentBucket = 'default';
                api.setBucket(this.currentBucket);
            } catch (e) {
                console.error('Failed to create default bucket:', e);
                throw new Error('无法连接到存储服务，请检查后端服务是否运行。');
            }
        }
    }

    bindEvents() {
        // Sidebar toggle
        const sidebarToggle = document.getElementById('sidebarToggle');
        if (sidebarToggle) {
            sidebarToggle.addEventListener('click', () => {
                document.getElementById('sidebar').classList.toggle('collapsed');
            });
        }

        // Mobile menu
        document.getElementById('mobileMenuBtn').addEventListener('click', () => {
            document.getElementById('sidebar').classList.toggle('mobile-open');
        });

        // Close mobile menu when clicking outside
        document.querySelector('.main-content').addEventListener('click', (e) => {
            if (window.innerWidth <= 768 && 
                !e.target.closest('.mobile-menu-btn') && 
                document.getElementById('sidebar').classList.contains('mobile-open')) {
                document.getElementById('sidebar').classList.remove('mobile-open');
            }
        });

        // Navigation items
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                const view = e.currentTarget.dataset.view;
                if (view === 'files') {
                    document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
                    e.currentTarget.classList.add('active');
                    this.loadDirectory('/');
                } else {
                    this.showToast('info', '该功能即将推出');
                }
            });
        });

        // View toggle
        document.getElementById('viewToggle').addEventListener('click', () => {
            this.toggleView();
        });

        // Refresh button
        document.getElementById('refreshBtn').addEventListener('click', () => {
            this.loadDirectory(this.currentPath);
            this.loadStorageInfo();
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
            const input = document.getElementById('folderName');
            input.value = '';
            setTimeout(() => input.focus(), 100);
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
        document.getElementById('downloadSelectedBtn').addEventListener('click', () => this.downloadSelected());
        document.getElementById('moveSelectedBtn').addEventListener('click', () => this.showMoveCopyDialog('move'));
        document.getElementById('copySelectedBtn').addEventListener('click', () => this.showMoveCopyDialog('copy'));
        document.getElementById('deleteSelectedBtn').addEventListener('click', () => this.showDeleteConfirmation());
        document.getElementById('cancelSelectionBtn').addEventListener('click', () => this.clearSelection());

        // Confirm delete
        document.getElementById('confirmDeleteBtn').addEventListener('click', () => this.deleteSelected());

        // Confirm rename
        document.getElementById('confirmRenameBtn').addEventListener('click', () => this.confirmRename());
        document.getElementById('newName').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.confirmRename();
        });

        // Confirm move/copy
        document.getElementById('confirmMoveCopyBtn').addEventListener('click', () => this.confirmMoveCopy());

        // Create bucket
        document.getElementById('createBucketBtn').addEventListener('click', () => this.createBucket());

        // Save API URL
        document.getElementById('saveApiUrlBtn').addEventListener('click', () => {
            const url = document.getElementById('apiBaseUrl').value.trim();
            if (url) {
                // Remove trailing slash
                const cleanUrl = url.replace(/\/+$/, '');
                localStorage.setItem('api_base_url', cleanUrl);
                this.showToast('success', '设置已保存，正在刷新...');
                setTimeout(() => window.location.reload(), 1000);
            }
        });

        // Upload panel close
        document.getElementById('closeUploadPanel').addEventListener('click', () => {
            document.getElementById('uploadPanel').classList.remove('visible');
        });

        // Context menu
        document.addEventListener('click', (e) => {
            if (!e.target.closest('.context-menu')) {
                this.hideContextMenu();
            }
        });

        document.getElementById('contextMenu').addEventListener('click', (e) => {
            const item = e.target.closest('.context-menu-item');
            if (item) {
                this.handleContextAction(item.dataset.action);
            }
        });

        // Drag and drop
        this.setupDragAndDrop();
        
        // Drag Selection
        this.setupDragSelection();

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
        
        // Click on empty space to clear selection
        document.getElementById('fileContainer').addEventListener('click', (e) => {
            if (e.target === e.currentTarget || e.target.id === 'fileGrid') {
                this.clearSelection();
            }
        });
    }

    setupDragAndDrop() {
        const fileContainer = document.querySelector('.main-content');
        const dropZone = document.getElementById('dropZone');

        let dragCounter = 0;

        fileContainer.addEventListener('dragenter', (e) => {
            e.preventDefault();
            e.stopPropagation();
            dragCounter++;
            dropZone.classList.add('active');
        });

        fileContainer.addEventListener('dragleave', (e) => {
            e.preventDefault();
            e.stopPropagation();
            dragCounter--;
            if (dragCounter === 0) {
                dropZone.classList.remove('active');
            }
        });

        fileContainer.addEventListener('dragover', (e) => {
            e.preventDefault();
            e.stopPropagation();
        });

        dropZone.addEventListener('drop', (e) => {
            e.preventDefault();
            e.stopPropagation();
            dropZone.classList.remove('active');
            dragCounter = 0;
            
            const files = e.dataTransfer.files;
            if (files.length > 0) {
                this.handleFileUpload(files);
            }
        });
    }

    setupDragSelection() {
        const container = document.getElementById('fileContainer');
        const selectionBox = document.getElementById('selectionBox');
        let isSelecting = false;
        let startX, startY;
        let initialSelection = new Set();

        container.addEventListener('mousedown', (e) => {
            // Ignore if clicking on a file item or scrollbar
            if (e.target.closest('.file-item') || e.target === container) {
                // If clicking directly on container (empty space), start selection
                // If clicking on file item, let the click handler handle it (unless we want to support drag start from item?)
                // Usually drag selection starts from empty space.
                if (e.target.closest('.file-item')) return;
            } else {
                return; // Clicked on something else?
            }

            // Only left mouse button
            if (e.button !== 0) return;

            isSelecting = true;
            startX = e.clientX;
            startY = e.clientY;

            // If ctrl/meta is pressed, keep existing selection
            if (e.ctrlKey || e.metaKey) {
                initialSelection = new Set(this.selectedItems);
            } else {
                this.clearSelection();
                initialSelection = new Set();
            }

            selectionBox.style.display = 'block';
            selectionBox.style.left = startX + 'px';
            selectionBox.style.top = startY + 'px';
            selectionBox.style.width = '0px';
            selectionBox.style.height = '0px';
        });

        document.addEventListener('mousemove', (e) => {
            if (!isSelecting) return;

            const currentX = e.clientX;
            const currentY = e.clientY;

            const left = Math.min(startX, currentX);
            const top = Math.min(startY, currentY);
            const width = Math.abs(currentX - startX);
            const height = Math.abs(currentY - startY);

            selectionBox.style.left = left + 'px';
            selectionBox.style.top = top + 'px';
            selectionBox.style.width = width + 'px';
            selectionBox.style.height = height + 'px';

            // Check collisions
            const boxRect = { left, top, right: left + width, bottom: top + height };
            
            document.querySelectorAll('.file-item').forEach(item => {
                const itemRect = item.getBoundingClientRect();
                const path = item.dataset.path;

                // Simple AABB collision
                const isIntersecting = !(boxRect.right < itemRect.left || 
                                       boxRect.left > itemRect.right || 
                                       boxRect.bottom < itemRect.top || 
                                       boxRect.top > itemRect.bottom);

                if (isIntersecting) {
                    this.selectedItems.add(path);
                } else if (!initialSelection.has(path)) {
                    // Only unselect if it wasn't initially selected (unless we want to toggle?)
                    // Standard behavior: new selection replaces old unless ctrl held.
                    // If ctrl held, we add to selection.
                    // If we drag over and then drag back, it should unselect.
                    this.selectedItems.delete(path);
                }
            });
            
            this.updateSelectionUI();
        });

        document.addEventListener('mouseup', () => {
            if (isSelecting) {
                isSelecting = false;
                selectionBox.style.display = 'none';
            }
        });
    }

    handleKeyboard(e) {
        // Escape - clear selection or close modal
        if (e.key === 'Escape') {
            const visibleModal = document.querySelector('.modal.visible');
            if (visibleModal) {
                this.hideModal(visibleModal.id);
            } else if (document.getElementById('contextMenu').classList.contains('visible')) {
                this.hideContextMenu();
            } else {
                this.clearSelection();
            }
        }

        // Delete - delete selected items
        if (e.key === 'Delete' && this.selectedItems.size > 0) {
            // Only if no modal is open
            if (!document.querySelector('.modal.visible')) {
                this.showDeleteConfirmation();
            }
        }

        // Ctrl+A - select all
        if (e.key === 'a' && (e.ctrlKey || e.metaKey)) {
            // Only if focus is not in an input
            if (!['INPUT', 'TEXTAREA'].includes(document.activeElement.tagName)) {
                e.preventDefault();
                this.selectAll();
            }
        }
    }

    // ============================================
    // Navigation
    // ============================================

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
            // Fix double slashes
            currentPath = currentPath.replace('//', '/');
            
            const isLast = index === parts.length - 1;
            html += `
                <span class="breadcrumb-separator">/</span>
                <span class="breadcrumb-item${isLast ? ' active' : ''}" data-path="${Utils.escapeHtml(currentPath)}">
                    ${Utils.escapeHtml(part)}
                </span>
            `;
        });

        breadcrumb.innerHTML = html;

        // Bind click events
        breadcrumb.querySelectorAll('.breadcrumb-item').forEach(item => {
            item.addEventListener('click', () => {
                if (!item.classList.contains('active')) {
                    const path = item.dataset.path;
                    this.loadDirectory(path);
                }
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

        fileGrid.innerHTML = sortedItems.map((item, index) => this.renderFileItem(item, index)).join('');

        // Update view class
        fileGrid.classList.toggle('list-view', this.currentView === 'list');

        // Bind events to file items
        this.bindFileItemEvents();
    }

    renderFileItem(item, index) {
        const { icon, category } = Utils.getFileIcon(item);
        const isSelected = this.selectedItems.has(item.path);
        const formattedSize = item.type === 'file' ? Utils.formatSize(item.size || 0) : '';
        const formattedDate = Utils.formatDate(item.created_at);
        const iconClass = category === 'folder' ? 'folder' : category;

        if (this.currentView === 'grid') {
            return `
                <div class="file-item ${isSelected ? 'selected' : ''}" 
                     data-path="${Utils.escapeHtml(item.path)}" 
                     data-type="${item.type}"
                     data-index="${index}">
                    <div class="file-item-icon ${iconClass}">
                        <i class="fas ${icon}"></i>
                    </div>
                    <div class="file-item-name" title="${Utils.escapeHtml(item.name)}">${Utils.escapeHtml(item.name)}</div>
                    <div class="file-item-meta">${formattedSize || formattedDate}</div>
                </div>
            `;
        } else {
            return `
                <div class="file-item ${isSelected ? 'selected' : ''}" 
                     data-path="${Utils.escapeHtml(item.path)}" 
                     data-type="${item.type}"
                     data-index="${index}">
                    <div class="file-item-icon ${iconClass}">
                        <i class="fas ${icon}"></i>
                    </div>
                    <div class="file-item-info">
                        <div class="file-item-name" title="${Utils.escapeHtml(item.name)}">${Utils.escapeHtml(item.name)}</div>
                        <div class="file-item-meta">${formattedSize}</div>
                        <div class="file-item-date">${formattedDate}</div>
                    </div>
                </div>
            `;
        }
    }

    bindFileItemEvents() {
        document.querySelectorAll('.file-item').forEach(item => {
            // Click to select
            item.addEventListener('click', (e) => {
                e.stopPropagation();
                const path = item.dataset.path;
                const index = parseInt(item.dataset.index);
                
                if (e.ctrlKey || e.metaKey) {
                    this.toggleSelection(path);
                    this.lastSelectedIndex = index;
                } else if (e.shiftKey && this.lastSelectedIndex !== -1) {
                    this.rangeSelect(index);
                } else {
                    this.clearSelection();
                    this.toggleSelection(path);
                    this.lastSelectedIndex = index;
                }
            });

            // Double click to open
            item.addEventListener('dblclick', (e) => {
                e.stopPropagation();
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
                e.stopPropagation();
                const path = item.dataset.path;
                
                // If item is not selected, select it (and clear others unless ctrl is pressed)
                if (!this.selectedItems.has(path)) {
                    if (!e.ctrlKey && !e.metaKey) {
                        this.clearSelection();
                    }
                    this.toggleSelection(path);
                    this.lastSelectedIndex = parseInt(item.dataset.index);
                }
                
                const itemData = this.items.find(i => i.path === path);
                this.showContextMenu(e.clientX, e.clientY, itemData);
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

    rangeSelect(endIndex) {
        const startIndex = this.lastSelectedIndex;
        if (startIndex === -1) return;

        const start = Math.min(startIndex, endIndex);
        const end = Math.max(startIndex, endIndex);

        // Sort items to match DOM order (which is sortedItems in renderFileList)
        // But wait, this.items is not sorted. We need to sort it same way as render.
        const sortedItems = [...this.items].sort((a, b) => {
            if (a.type === 'directory' && b.type !== 'directory') return -1;
            if (a.type !== 'directory' && b.type === 'directory') return 1;
            return (a.name || '').localeCompare(b.name || '');
        });

        this.clearSelection();
        for (let i = start; i <= end; i++) {
            this.selectedItems.add(sortedItems[i].path);
        }
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
        // Don't reset lastSelectedIndex to allow shift-select from last position
        this.updateSelectionUI();
    }

    updateSelectionUI() {
        // Update file item classes
        document.querySelectorAll('.file-item').forEach(item => {
            const isSelected = this.selectedItems.has(item.dataset.path);
            item.classList.toggle('selected', isSelected);
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
            this.loadDirectory(this.currentPath); // Reload to reset
            return;
        }

        // Client-side filtering for now
        // Ideally this should be a server-side search API
        const filtered = this.items.filter(item => 
            item.name.toLowerCase().includes(query.toLowerCase())
        );

        // We temporarily replace items for rendering, but we should probably keep original items
        // For simplicity, let's just filter what we have. 
        // A better approach would be to have a separate 'displayItems' list.
        
        // Re-render with filtered items
        const fileGrid = document.getElementById('fileGrid');
        fileGrid.innerHTML = filtered.map((item, index) => this.renderFileItem(item, index)).join('');
        this.bindFileItemEvents();
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
        // We need to show it first to get dimensions, but it's hidden.
        // Add visible class first but maybe offscreen?
        menu.classList.add('visible');
        
        const menuRect = menu.getBoundingClientRect();
        if (x + menuRect.width > window.innerWidth) {
            menu.style.left = (x - menuRect.width) + 'px';
        }
        if (y + menuRect.height > window.innerHeight) {
            menu.style.top = (y - menuRect.height) + 'px';
        }

        // Show/hide appropriate options
        const isDirectory = item.type === 'directory';
        menu.querySelector('[data-action="open"]').style.display = isDirectory ? 'flex' : 'none';
        menu.querySelector('[data-action="preview"]').style.display = !isDirectory && Utils.isPreviewable(item) ? 'flex' : 'none';
        menu.querySelector('[data-action="download"]').style.display = !isDirectory ? 'flex' : 'none';
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
            const uploadItem = document.createElement('div');
            uploadItem.className = 'upload-item';
            uploadItem.id = `upload-${uploadId}`;
            uploadItem.innerHTML = `
                <i class="fas fa-file upload-item-icon"></i>
                <div class="upload-item-info">
                    <div class="upload-item-name">${Utils.escapeHtml(file.name)}</div>
                    <div class="upload-item-progress">
                        <div class="upload-item-progress-bar" style="width: 0%"></div>
                    </div>
                    <div class="upload-item-status">准备上传...</div>
                </div>
            `;
            uploadList.prepend(uploadItem);

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
                    item.querySelector('.upload-item-progress-bar').style.backgroundColor = 'var(--success-color)';
                }
            } catch (error) {
                const item = document.getElementById(`upload-${uploadId}`);
                if (item) {
                    item.querySelector('.upload-item-status').textContent = '上传失败: ' + error.message;
                    item.querySelector('.upload-item-status').classList.add('error');
                    item.querySelector('.upload-item-progress-bar').style.backgroundColor = 'var(--danger-color)';
                }
            }
        }

        // Refresh file list
        this.loadDirectory(this.currentPath);
        this.loadStorageInfo();
        this.showToast('success', `上传任务已完成`);
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
            this.showToast('info', '开始下载...');
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
        const files = [...this.selectedItems].map(path => this.items.find(i => i.path === path)).filter(i => i && i.type === 'file');
        
        if (files.length === 0) {
            this.showToast('warning', '请选择要下载的文件');
            return;
        }

        if (files.length > 5) {
            if (!confirm(`确定要同时下载 ${files.length} 个文件吗？`)) return;
        }

        for (const item of files) {
            await this.downloadFile(item);
        }
    }

    showRenameDialog(item) {
        this.contextMenuItem = item; // Ensure context item is set
        document.getElementById('newName').value = item.name;
        this.showModal('renameModal');
        const input = document.getElementById('newName');
        setTimeout(() => {
            input.focus();
            input.select();
        }, 100);
    }

    async confirmRename() {
        const newName = document.getElementById('newName').value.trim();
        if (!newName) {
            this.showToast('warning', '请输入新名称');
            return;
        }

        const item = this.contextMenuItem;
        if (!item) return;

        if (newName === item.name) {
            this.hideModal('renameModal');
            return;
        }

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
        if (count === 0) return;
        
        document.getElementById('deleteMessage').textContent = 
            `确定要删除选中的 ${count} 个项目吗？此操作无法撤销。`;
        this.showModal('deleteModal');
    }

    async deleteSelected() {
        const paths = [...this.selectedItems];
        let successCount = 0;
        let failCount = 0;

        this.showToast('info', '正在删除...');

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
        this.loadStorageInfo();

        if (failCount === 0) {
            this.showToast('success', `成功删除 ${successCount} 个项目`);
        } else {
            this.showToast('warning', `成功 ${successCount} 个，失败 ${failCount} 个`);
        }
    }

    showMoveCopyDialog(action) {
        if (this.selectedItems.size === 0) {
            this.showToast('warning', '请先选择项目');
            return;
        }
        
        this.moveCopyAction = action;
        document.getElementById('moveCopyTitle').textContent = action === 'move' ? '移动到' : '复制到';
        this.loadFolderTree();
        this.showModal('moveCopyModal');
    }

    async loadFolderTree() {
        const tree = document.getElementById('folderTree');
        tree.innerHTML = '<div class="loading-state" style="position: relative; height: 100px;"><div class="spinner"></div></div>';

        try {
            // Load root directory
            // TODO: Ideally we should support recursive tree loading or lazy loading
            // For now, just load root folders
            const response = await api.listDirectory('/');
            const folders = (response.items || []).filter(i => i.type === 'directory');

            let html = `
                <div class="folder-tree-item selected" data-path="/" style="padding: 8px; cursor: pointer; display: flex; align-items: center;">
                    <i class="fas fa-folder" style="margin-right: 8px; color: #fcd147;"></i>
                    <span>根目录</span>
                </div>
            `;

            folders.forEach(folder => {
                html += `
                    <div class="folder-tree-item" data-path="${Utils.escapeHtml(folder.path)}" style="padding: 8px; cursor: pointer; display: flex; align-items: center; margin-left: 20px;">
                        <i class="fas fa-folder" style="margin-right: 8px; color: #fcd147;"></i>
                        <span>${Utils.escapeHtml(folder.name)}</span>
                    </div>
                `;
            });

            tree.innerHTML = html;

            // Bind click events
            tree.querySelectorAll('.folder-tree-item').forEach(item => {
                item.addEventListener('click', () => {
                    tree.querySelectorAll('.folder-tree-item').forEach(i => {
                        i.classList.remove('selected');
                        i.style.backgroundColor = 'transparent';
                    });
                    item.classList.add('selected');
                    item.style.backgroundColor = '#eff6fc';
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

        this.showToast('info', `正在${action === 'move' ? '移动' : '复制'}...`);

        for (const path of paths) {
            const item = this.items.find(i => i.path === path);
            if (!item) continue;

            // Skip if destination is same as source parent
            if (Utils.getParentPath(path) === destPath) {
                continue;
            }

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
        this.loadStorageInfo();

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
        content.innerHTML = '<div class="spinner"></div>';
        
        this.showModal('previewModal');

        const fileUrl = api.getFileUrl(path);

        switch (previewType) {
            case 'image':
                content.innerHTML = `<img src="${fileUrl}" alt="${Utils.escapeHtml(item.name)}">`;
                break;
            case 'video':
                content.innerHTML = `<video src="${fileUrl}" controls autoplay style="max-width: 100%; max-height: 100%"></video>`;
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
                    </div>
                `;
        }
    }

    // ============================================
    // Settings
    // ============================================

    async showSettings() {
        this.showModal('settingsModal');

        // Load API URL settings
        const currentUrl = localStorage.getItem('api_base_url') || window.API_BASE_URL || 'http://localhost:8080/api/v1';
        document.getElementById('apiBaseUrl').value = currentUrl;
        document.getElementById('currentApiUrl').textContent = currentUrl;

        await this.loadBucketList();
        await this.loadAccountList();
    }

    async loadBucketList() {
        const list = document.getElementById('bucketList');
        list.innerHTML = '<div class="spinner"></div>';
        
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
                    <button class="icon-btn bucket-delete-btn" data-bucket="${Utils.escapeHtml(bucket.name)}" title="删除" style="color: var(--danger-color);">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            `).join('');
            
            // Add event listeners for delete buttons
            list.querySelectorAll('.bucket-delete-btn').forEach(btn => {
                btn.addEventListener('click', () => {
                    const bucketName = btn.dataset.bucket;
                    this.deleteBucket(bucketName);
                });
            });
        } catch (error) {
            list.innerHTML = '<p>加载失败</p>';
        }
    }

    async loadAccountList() {
        const list = document.getElementById('accountList');
        list.innerHTML = '<div class="spinner"></div>';
        
        try {
            const response = await api.listAccounts();
            const accounts = response.accounts || [];
            
            if (accounts.length === 0) {
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
                    <button class="icon-btn account-sync-btn" data-account-id="${account.id}" title="同步">
                        <i class="fas fa-sync-alt"></i>
                    </button>
                </div>
            `).join('');
            
            // Add event listeners for sync buttons
            list.querySelectorAll('.account-sync-btn').forEach(btn => {
                btn.addEventListener('click', () => {
                    const accountId = btn.dataset.accountId;
                    this.syncAccount(accountId);
                });
            });
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
            this.showToast('info', '正在同步...');
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
            const used = data.used_space || 0;
            const total = data.total_space || 0;
            const percent = total > 0 ? (used / total * 100) : 0;

            document.getElementById('storageUsed').textContent = Utils.formatSize(used);
            document.getElementById('storageTotal').textContent = Utils.formatSize(total);
            document.getElementById('storageBar').style.width = Math.min(percent, 100) + '%';
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
            <button class="toast-close" data-toast-id="${id}">
                <i class="fas fa-times"></i>
            </button>
        `;
        
        // Add event listener for close button
        const closeBtn = toast.querySelector('.toast-close');
        closeBtn.addEventListener('click', () => {
            this.hideToast(id);
        });

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
