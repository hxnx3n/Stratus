import { useState, useRef, useEffect } from 'react'
import { useFileStore, FileItem } from '../stores/fileStore'
import { formatBytes } from '../lib/utils'
import { format } from 'date-fns'
import ConfirmDialog from './ConfirmDialog'
import { api } from '../lib/api'
import {
  Folder,
  File,
  Image,
  Video,
  Music,
  FileText,
  Archive,
  MoreVertical,
  Download,
  Trash2,
  Edit,
} from 'lucide-react'
import FilePreview from './FilePreview'

interface FileListProps {
  files: FileItem[]
  onNavigate: (file: FileItem) => void
}

export default function FileList({ files, onNavigate }: FileListProps) {
  const { moveToTrash, renameFile, selectedFiles, selectFile, deselectFile } = useFileStore()
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; file: FileItem } | null>(null)
  const [previewFile, setPreviewFile] = useState<FileItem | null>(null)
  const [editingFile, setEditingFile] = useState<string | null>(null)
  const [newName, setNewName] = useState('')
  const [deleteDialog, setDeleteDialog] = useState<{ isOpen: boolean; file?: FileItem }>({ isOpen: false })
  const inputRef = useRef<HTMLInputElement>(null)
  const contextMenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (contextMenu && contextMenuRef.current && !contextMenuRef.current.contains(e.target as Node)) {
        setContextMenu(null)
      }
    }
    document.addEventListener('click', handleClickOutside)
    return () => document.removeEventListener('click', handleClickOutside)
  }, [contextMenu])

  const getIcon = (file: FileItem) => {
    if (file.is_directory) return <Folder className="w-10 h-10 text-yellow-500" />
    if (file.mime_type?.startsWith('image/')) return <Image className="w-10 h-10 text-green-500" />
    if (file.mime_type?.startsWith('video/')) return <Video className="w-10 h-10 text-purple-500" />
    if (file.mime_type?.startsWith('audio/')) return <Music className="w-10 h-10 text-pink-500" />
    if (file.mime_type?.startsWith('text/')) return <FileText className="w-10 h-10 text-blue-500" />
    if (file.mime_type?.includes('zip') || file.mime_type?.includes('rar')) return <Archive className="w-10 h-10 text-orange-500" />
    return <File className="w-10 h-10 text-gray-500" />
  }

  const handleContextMenu = (e: React.MouseEvent, file: FileItem) => {
    e.preventDefault()
    setContextMenu({ x: e.clientX, y: e.clientY, file })
  }

  const closeContextMenu = () => setContextMenu(null)

  const handleDelete = async (file: FileItem) => {
    closeContextMenu()
    setDeleteDialog({ isOpen: true, file })
  }

  const confirmDelete = async () => {
    if (deleteDialog.file) {
      await moveToTrash(deleteDialog.file.id)
    }
    setDeleteDialog({ isOpen: false })
  }

  const getNameWithoutExtension = (name: string, isDirectory: boolean) => {
    if (isDirectory) return name
    const lastDot = name.lastIndexOf('.')
    if (lastDot === -1) return name
    return name.substring(0, lastDot)
  }

  const getExtension = (name: string, isDirectory: boolean) => {
    if (isDirectory) return ''
    const lastDot = name.lastIndexOf('.')
    if (lastDot === -1) return ''
    return name.substring(lastDot)
  }

  const handleRename = (file: FileItem) => {
    closeContextMenu()
    setEditingFile(file.id)
    setNewName(getNameWithoutExtension(file.name, file.is_directory))
    setTimeout(() => {
      inputRef.current?.focus()
      inputRef.current?.select()
    }, 0)
  }

  const submitRename = async (file: FileItem) => {
    const extension = getExtension(file.name, file.is_directory)
    const fullNewName = newName + extension
    if (newName && fullNewName !== file.name) {
      await renameFile(file.id, fullNewName)
    }
    setEditingFile(null)
    setNewName('')
  }

  const handleDownload = async (file: FileItem) => {
    closeContextMenu()
    try {
      const response = await api.get(`/api/files/${file.id}/download`, {
        responseType: 'blob',
      })
      const url = URL.createObjectURL(response.data)
      const a = document.createElement('a')
      a.href = url
      a.download = file.name
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    } catch (err) {
      console.error('Failed to download file:', err)
    }
  }

  const handleClick = (file: FileItem) => {
    if (file.is_directory) {
      onNavigate(file)
    } else {
      setPreviewFile(file)
    }
  }

  const handleSelect = (e: React.MouseEvent, file: FileItem) => {
    e.stopPropagation()
    if (selectedFiles.has(file.id)) {
      deselectFile(file.id)
    } else {
      selectFile(file.id)
    }
  }

  if (files.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-gray-500">
        <Folder className="w-16 h-16 mb-4 text-gray-300" />
        <p>이 폴더는 비어있습니다</p>
      </div>
    )
  }

  return (
    <>
      <div className="overflow-hidden" onClick={closeContextMenu}>
        <table className="w-full table-fixed">
          <thead className="bg-gray-50 border-b">
            <tr>
              <th className="w-12 px-4 py-3"></th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-600">이름</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-600 w-28">크기</th>
              <th className="px-4 py-3 text-left text-sm font-medium text-gray-600 w-36">수정일</th>
              <th className="px-4 py-3 w-14"></th>
            </tr>
          </thead>
          <tbody>
            {files.map((file) => (
              <tr
                key={file.id}
                className={`border-b hover:bg-gray-50 cursor-pointer ${selectedFiles.has(file.id) ? 'bg-primary-50' : ''
                  }`}
                onClick={() => handleClick(file)}
                onContextMenu={(e) => handleContextMenu(e, file)}
              >
                <td className="px-4 py-2" onClick={(e) => handleSelect(e, file)}>
                  <input
                    type="checkbox"
                    checked={selectedFiles.has(file.id)}
                    onChange={() => { }}
                    className="rounded border-gray-300"
                  />
                </td>
                <td className="px-4 py-2">
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="flex-shrink-0">{getIcon(file)}</div>
                    <div className="flex items-center gap-2 min-w-0 flex-1">
                      {editingFile === file.id ? (
                        <div className="inline-flex items-center rounded ring-2 ring-primary-500">
                          <input
                            ref={inputRef}
                            type="text"
                            value={newName}
                            onChange={(e) => setNewName(e.target.value)}
                            onBlur={() => submitRename(file)}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') submitRename(file)
                              if (e.key === 'Escape') setEditingFile(null)
                            }}
                            onClick={(e) => e.stopPropagation()}
                            className={`px-2 py-1 focus:outline-none ${!file.is_directory && getExtension(file.name, file.is_directory)
                              ? 'rounded-l'
                              : 'rounded'
                              }`}
                          />
                          {!file.is_directory && getExtension(file.name, file.is_directory) && (
                            <span className="px-2 py-1 bg-gray-100 rounded-r text-gray-600 text-sm border-l border-gray-300">
                              {getExtension(file.name, file.is_directory)}
                            </span>
                          )}
                        </div>
                      ) : (
                        <span className="text-gray-900 truncate">{file.name}</span>
                      )}

                    </div>
                  </div>
                </td>
                <td className="px-4 py-2 text-sm text-gray-600">
                  {file.is_directory ? '-' : formatBytes(file.size)}
                </td>
                <td className="px-4 py-2 text-sm text-gray-600">
                  {format(new Date(file.updated_at), 'yyyy-MM-dd HH:mm')}
                </td>
                <td className="px-4 py-2">
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      handleContextMenu(e, file)
                    }}
                    className="p-1 hover:bg-gray-200 rounded"
                  >
                    <MoreVertical className="w-5 h-5 text-gray-500" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {contextMenu && (
        <div
          ref={contextMenuRef}
          className="context-menu fixed bg-white shadow-lg rounded-lg border py-2 z-50 min-w-[160px]"
          style={{
            left: Math.min(Math.max(contextMenu.x, 10), window.innerWidth - 180),
            top: Math.min(Math.max(contextMenu.y, 10), window.innerHeight - 250),
          }}
        >
          {!contextMenu.file.is_directory && (
            <button
              onClick={() => handleDownload(contextMenu.file)}
              className="flex items-center gap-2 w-full px-4 py-2 hover:bg-gray-100 text-left"
            >
              <Download className="w-4 h-4" />
              <span>다운로드</span>
            </button>
          )}
          <button
            onClick={() => handleRename(contextMenu.file)}
            className="flex items-center gap-2 w-full px-4 py-2 hover:bg-gray-100 text-left"
          >
            <Edit className="w-4 h-4" />
            <span>이름 변경</span>
          </button>
          <hr className="my-1" />
          <button
            onClick={() => handleDelete(contextMenu.file)}
            className="flex items-center gap-2 w-full px-4 py-2 hover:bg-gray-100 text-left text-red-600"
          >
            <Trash2 className="w-4 h-4" />
            <span>삭제</span>
          </button>
        </div>
      )}

      {previewFile && (
        <FilePreview
          file={previewFile}
          files={files}
          onClose={() => setPreviewFile(null)}
          onFileChange={setPreviewFile}
        />
      )}

      <ConfirmDialog
        isOpen={deleteDialog.isOpen}
        title="휴지통으로 이동"
        message={`"${deleteDialog.file?.name}"을(를) 휴지통으로 이동하시겠습니까?`}
        confirmText="삭제"
        cancelText="취소"
        variant="danger"
        onConfirm={confirmDelete}
        onCancel={() => setDeleteDialog({ isOpen: false })}
      />
    </>
  )
}
