/**
 * Main Files Page component
 */

import React, { useState, useEffect, useCallback } from 'react';
import { Layout, Modal, Input, message, TreeSelect } from 'antd';
import { Sidebar, Header, Toolbar } from '../components/layout';
import {
  FileGrid,
  FileList,
  FilePreview,
  ContextMenu,
  UploadPanel,
  SelectionBar,
} from '../components/file';
import { useAppStore } from '../store';
import {
  useDirectory,
  useCreateDirectory,
  useDeleteItems,
  useRenameItem,
  useMoveItems,
  useCopyItems,
  useUploadFiles,
} from '../hooks';
import { getSpaceOverview, downloadFile, listBuckets, createBucket, listDirectory } from '../api';
import { setCurrentBucket } from '../api/vfs';
import type { FileItem, DirectoryListing } from '../types';
import { normalizePath, joinPath, getParentPath, getPreviewType } from '../utils';
import './FilesPage.css';

const { Sider, Content } = Layout;

// Tree node type for folder selection
interface TreeNode {
  value: string;
  title: string;
  children?: TreeNode[];
}

const FilesPage: React.FC = () => {
  // Store state
  const currentPath = useAppStore(state => state.currentPath);
  const setCurrentPath = useAppStore(state => state.setCurrentPath);
  const files = useAppStore(state => state.files);
  const isLoading = useAppStore(state => state.isLoading);
  const selectedPaths = useAppStore(state => state.selectedPaths);
  const setSelectedPaths = useAppStore(state => state.setSelectedPaths);
  const clearSelection = useAppStore(state => state.clearSelection);
  const viewMode = useAppStore(state => state.viewMode);
  const setSpaceInfo = useAppStore(state => state.setSpaceInfo);
  const uploads = useAppStore(state => state.uploads);

  // Local state
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [contextMenu, setContextMenu] = useState<{
    visible: boolean;
    x: number;
    y: number;
    item: FileItem | null;
  }>({ visible: false, x: 0, y: 0, item: null });
  const [previewItem, setPreviewItem] = useState<FileItem | null>(null);
  const [newFolderModalVisible, setNewFolderModalVisible] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');
  const [renameModalVisible, setRenameModalVisible] = useState(false);
  const [newName, setNewName] = useState('');
  const [renameItem, setRenameItem] = useState<FileItem | null>(null);
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [moveModalVisible, setMoveModalVisible] = useState(false);
  const [copyModalVisible, setCopyModalVisible] = useState(false);
  const [destinationPath, setDestinationPath] = useState('/');
  const [folderTreeData, setFolderTreeData] = useState<TreeNode[]>([]);
  const [uploadPanelVisible, setUploadPanelVisible] = useState(false);
  const [settingsModalVisible, setSettingsModalVisible] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [lastSelectedIndex, setLastSelectedIndex] = useState(-1);

  // Hooks
  const { refetch } = useDirectory(currentPath);
  const createDirectory = useCreateDirectory();
  const deleteItems = useDeleteItems();
  const renameItemMutation = useRenameItem();
  const moveItems = useMoveItems();
  const copyItems = useCopyItems();
  const { uploadFiles } = useUploadFiles();

  // Initialize bucket and load space info
  useEffect(() => {
    const init = async () => {
      try {
        // Check and create default bucket
        const bucketsResponse = await listBuckets();
        if (bucketsResponse.length === 0) {
          await createBucket('default');
        }
        setCurrentBucket(bucketsResponse[0]?.name || 'default');

        // Load space info
        const spaceInfo = await getSpaceOverview();
        setSpaceInfo(spaceInfo);
      } catch (error) {
        console.error('Initialization failed:', error);
      }
    };
    init();
  }, [setSpaceInfo]);

  // Show upload panel when uploads exist - derive from state instead of using effect
  const shouldShowUploadPanel = uploadPanelVisible || uploads.size > 0;

  // Navigation
  const navigate = useCallback((path: string) => {
    const normalizedPath = normalizePath(path);
    setCurrentPath(normalizedPath);
    clearSelection();
  }, [setCurrentPath, clearSelection]);

  // File selection
  const handleSelect = useCallback((path: string, ctrlKey: boolean, shiftKey: boolean) => {
    const sortedFiles = [...files].sort((a, b) => {
      if (a.type === 'directory' && b.type !== 'directory') return -1;
      if (a.type !== 'directory' && b.type === 'directory') return 1;
      return (a.name || '').localeCompare(b.name || '');
    });

    const currentIndex = sortedFiles.findIndex(f => f.path === path);

    if (shiftKey && lastSelectedIndex !== -1) {
      // Range selection
      const start = Math.min(lastSelectedIndex, currentIndex);
      const end = Math.max(lastSelectedIndex, currentIndex);
      const newSelection = new Set<string>();
      for (let i = start; i <= end; i++) {
        newSelection.add(sortedFiles[i].path);
      }
      setSelectedPaths(newSelection);
    } else if (ctrlKey) {
      // Toggle selection
      const newSelection = new Set(selectedPaths);
      if (newSelection.has(path)) {
        newSelection.delete(path);
      } else {
        newSelection.add(path);
      }
      setSelectedPaths(newSelection);
      setLastSelectedIndex(currentIndex);
    } else {
      // Single selection
      setSelectedPaths(new Set([path]));
      setLastSelectedIndex(currentIndex);
    }
  }, [files, selectedPaths, setSelectedPaths, lastSelectedIndex]);

  // Download files
  const handleDownload = useCallback(async (paths: string[]) => {
    for (const path of paths) {
      const item = files.find(f => f.path === path);
      if (item && item.type === 'file') {
        try {
          const blob = await downloadFile(path);
          const url = URL.createObjectURL(blob);
          const a = document.createElement('a');
          a.href = url;
          a.download = item.name;
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
          URL.revokeObjectURL(url);
        } catch (error) {
          message.error('下载失败: ' + (error instanceof Error ? error.message : '未知错误'));
        }
      }
    }
  }, [files]);

  // Double click handler
  const handleDoubleClick = useCallback((item: FileItem) => {
    if (item.type === 'directory') {
      navigate(item.path);
    } else {
      // Preview file
      const previewType = getPreviewType(item.name, item.mime_type);
      if (previewType) {
        setPreviewItem(item);
      } else {
        handleDownload([item.path]);
      }
    }
  }, [navigate, handleDownload]);

  // Context menu
  const handleContextMenu = useCallback((e: React.MouseEvent, item: FileItem) => {
    e.preventDefault();
    if (!selectedPaths.has(item.path)) {
      setSelectedPaths(new Set([item.path]));
    }
    setContextMenu({
      visible: true,
      x: e.clientX,
      y: e.clientY,
      item,
    });
  }, [selectedPaths, setSelectedPaths]);

  // Create folder
  const handleCreateFolder = useCallback(() => {
    if (!newFolderName.trim()) {
      message.warning('请输入文件夹名称');
      return;
    }
    createDirectory.mutate(newFolderName.trim(), {
      onSuccess: () => {
        setNewFolderModalVisible(false);
        setNewFolderName('');
        refetch();
      },
    });
  }, [newFolderName, createDirectory, refetch]);

  // Rename
  const handleRename = useCallback(() => {
    if (!renameItem || !newName.trim()) {
      message.warning('请输入新名称');
      return;
    }
    const newPath = joinPath(getParentPath(renameItem.path), newName.trim());
    renameItemMutation.mutate(
      { oldPath: renameItem.path, newPath },
      {
        onSuccess: () => {
          setRenameModalVisible(false);
          setRenameItem(null);
          setNewName('');
          refetch();
        },
      }
    );
  }, [renameItem, newName, renameItemMutation, refetch]);

  // Delete
  const handleDelete = useCallback(() => {
    const paths = Array.from(selectedPaths);
    deleteItems.mutate(paths, {
      onSuccess: () => {
        setDeleteModalVisible(false);
        refetch();
      },
    });
  }, [selectedPaths, deleteItems, refetch]);

  // Move/Copy
  const loadFolderTree = useCallback(async () => {
    try {
      const buildTree = async (path: string, depth: number = 0): Promise<TreeNode[]> => {
        if (depth > 2) return []; // Limit depth
        const listing: DirectoryListing = await listDirectory(path);
        const folders = (listing.items || []).filter(i => i.type === 'directory');
        const result: typeof folderTreeData = [];
        for (const folder of folders) {
          const children = await buildTree(folder.path, depth + 1);
          result.push({
            value: folder.path,
            title: folder.name,
            children: children.length > 0 ? children : undefined,
          });
        }
        return result;
      };

      const tree = await buildTree('/');
      setFolderTreeData([
        {
          value: '/',
          title: '根目录',
          children: tree,
        },
      ]);
    } catch (error) {
      console.error('Failed to load folder tree:', error);
    }
  }, []);

  const handleMove = useCallback(() => {
    const paths = Array.from(selectedPaths);
    moveItems.mutate(
      { paths, destination: destinationPath },
      {
        onSuccess: () => {
          setMoveModalVisible(false);
          setDestinationPath('/');
          refetch();
        },
      }
    );
  }, [selectedPaths, destinationPath, moveItems, refetch]);

  const handleCopy = useCallback(() => {
    const paths = Array.from(selectedPaths);
    copyItems.mutate(
      { paths, destination: destinationPath },
      {
        onSuccess: () => {
          setCopyModalVisible(false);
          setDestinationPath('/');
          refetch();
        },
      }
    );
  }, [selectedPaths, destinationPath, copyItems, refetch]);

  // Upload
  const handleUpload = useCallback(async (fileList: File[]) => {
    await uploadFiles(fileList);
    refetch();
  }, [uploadFiles, refetch]);

  // Search filter
  const filteredFiles = searchQuery
    ? files.filter(f => f.name.toLowerCase().includes(searchQuery.toLowerCase()))
    : files;

  return (
    <Layout className="files-page">
      <Sider
        width={250}
        collapsible
        collapsed={sidebarCollapsed}
        onCollapse={setSidebarCollapsed}
        breakpoint="lg"
        collapsedWidth={0}
        trigger={null}
        className="files-sider"
      >
        <Sidebar
          collapsed={sidebarCollapsed}
          onSettingsClick={() => setSettingsModalVisible(true)}
        />
      </Sider>

      <Layout className="files-main">
        <Header
          onSearch={setSearchQuery}
          onRefresh={() => refetch()}
          onMenuClick={() => setSidebarCollapsed(!sidebarCollapsed)}
        />

        {selectedPaths.size > 0 && (
          <SelectionBar
            count={selectedPaths.size}
            onDownload={() => handleDownload(Array.from(selectedPaths))}
            onMove={() => {
              loadFolderTree();
              setMoveModalVisible(true);
            }}
            onCopy={() => {
              loadFolderTree();
              setCopyModalVisible(true);
            }}
            onDelete={() => setDeleteModalVisible(true)}
            onCancel={clearSelection}
          />
        )}

        <Toolbar
          onNavigate={navigate}
          onUpload={handleUpload}
          onNewFolder={() => setNewFolderModalVisible(true)}
        />

        <Content className="files-content" onClick={() => {
          clearSelection();
          setContextMenu({ ...contextMenu, visible: false });
        }}>
          {viewMode === 'grid' ? (
            <FileGrid
              items={filteredFiles}
              loading={isLoading}
              selectedPaths={selectedPaths}
              onSelect={handleSelect}
              onDoubleClick={handleDoubleClick}
              onContextMenu={handleContextMenu}
            />
          ) : (
            <FileList
              items={filteredFiles}
              selectedPaths={selectedPaths}
              onSelect={(path, selected) => {
                const newSelection = new Set(selectedPaths);
                if (selected) {
                  newSelection.add(path);
                } else {
                  newSelection.delete(path);
                }
                setSelectedPaths(newSelection);
              }}
              onDoubleClick={handleDoubleClick}
              onContextMenu={handleContextMenu}
            />
          )}
        </Content>
      </Layout>

      {/* Context Menu */}
      <ContextMenu
        {...contextMenu}
        onClose={() => setContextMenu({ ...contextMenu, visible: false })}
        onOpen={() => {
          if (contextMenu.item) {
            navigate(contextMenu.item.path);
          }
        }}
        onPreview={() => {
          if (contextMenu.item) {
            setPreviewItem(contextMenu.item);
          }
        }}
        onDownload={() => {
          if (contextMenu.item) {
            handleDownload([contextMenu.item.path]);
          }
        }}
        onRename={() => {
          if (contextMenu.item) {
            setRenameItem(contextMenu.item);
            setNewName(contextMenu.item.name);
            setRenameModalVisible(true);
          }
        }}
        onMove={() => {
          loadFolderTree();
          setMoveModalVisible(true);
        }}
        onCopy={() => {
          loadFolderTree();
          setCopyModalVisible(true);
        }}
        onDelete={() => setDeleteModalVisible(true)}
      />

      {/* File Preview */}
      {previewItem && (
        <FilePreview
          visible={!!previewItem}
          name={previewItem.name}
          path={previewItem.path}
          mimeType={previewItem.mime_type}
          onClose={() => setPreviewItem(null)}
          onDownload={() => handleDownload([previewItem.path])}
        />
      )}

      {/* Upload Panel */}
      <UploadPanel
        visible={shouldShowUploadPanel}
        onClose={() => setUploadPanelVisible(false)}
      />

      {/* New Folder Modal */}
      <Modal
        title="新建文件夹"
        open={newFolderModalVisible}
        onOk={handleCreateFolder}
        onCancel={() => {
          setNewFolderModalVisible(false);
          setNewFolderName('');
        }}
        confirmLoading={createDirectory.isPending}
      >
        <Input
          placeholder="请输入文件夹名称"
          value={newFolderName}
          onChange={(e) => setNewFolderName(e.target.value)}
          onPressEnter={handleCreateFolder}
          autoFocus
        />
      </Modal>

      {/* Rename Modal */}
      <Modal
        title="重命名"
        open={renameModalVisible}
        onOk={handleRename}
        onCancel={() => {
          setRenameModalVisible(false);
          setRenameItem(null);
          setNewName('');
        }}
        confirmLoading={renameItemMutation.isPending}
      >
        <Input
          placeholder="请输入新名称"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onPressEnter={handleRename}
          autoFocus
        />
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        title="确认删除"
        open={deleteModalVisible}
        onOk={handleDelete}
        onCancel={() => setDeleteModalVisible(false)}
        okText="删除"
        okButtonProps={{ danger: true }}
        confirmLoading={deleteItems.isPending}
      >
        <p>确定要删除选中的 {selectedPaths.size} 个项目吗？此操作无法撤销。</p>
      </Modal>

      {/* Move Modal */}
      <Modal
        title="移动到"
        open={moveModalVisible}
        onOk={handleMove}
        onCancel={() => {
          setMoveModalVisible(false);
          setDestinationPath('/');
        }}
        confirmLoading={moveItems.isPending}
      >
        <TreeSelect
          style={{ width: '100%' }}
          value={destinationPath}
          dropdownStyle={{ maxHeight: 400, overflow: 'auto' }}
          treeData={folderTreeData}
          placeholder="选择目标文件夹"
          treeDefaultExpandAll
          onChange={setDestinationPath}
        />
      </Modal>

      {/* Copy Modal */}
      <Modal
        title="复制到"
        open={copyModalVisible}
        onOk={handleCopy}
        onCancel={() => {
          setCopyModalVisible(false);
          setDestinationPath('/');
        }}
        confirmLoading={copyItems.isPending}
      >
        <TreeSelect
          style={{ width: '100%' }}
          value={destinationPath}
          dropdownStyle={{ maxHeight: 400, overflow: 'auto' }}
          treeData={folderTreeData}
          placeholder="选择目标文件夹"
          treeDefaultExpandAll
          onChange={setDestinationPath}
        />
      </Modal>

      {/* Settings Modal */}
      <Modal
        title="设置"
        open={settingsModalVisible}
        onCancel={() => setSettingsModalVisible(false)}
        footer={null}
        width={600}
      >
        <p>设置功能即将推出...</p>
      </Modal>
    </Layout>
  );
};

export default FilesPage;
