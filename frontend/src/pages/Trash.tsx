import { useState, useEffect, useRef } from 'react'
import { api } from '../lib/api'
import { formatBytes } from '../lib/utils'
import { format } from 'date-fns'
import { Trash2, RotateCcw, Folder, File, AlertTriangle } from 'lucide-react'
import { AxiosError } from 'axios'
import ConfirmDialog from '../components/ConfirmDialog'

interface ApiError {
  error?: string
}

interface TrashItem {
  id: string
  name: string
  path: string
  size: number
  is_directory: boolean
  trashed_at: string
}

export default function Trash() {
  const [items, setItems] = useState<TrashItem[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set())
  const [deleteDialog, setDeleteDialog] = useState<{ isOpen: boolean; item?: TrashItem }>({ isOpen: false })
  const [emptyTrashDialog, setEmptyTrashDialog] = useState(false)
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number; item: TrashItem } | null>(null)
  const contextMenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    fetchTrash()
  }, [])

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (contextMenu && contextMenuRef.current && !contextMenuRef.current.contains(e.target as Node)) {
        setContextMenu(null)
      }
    }
    document.addEventListener('click', handleClickOutside)
    return () => document.removeEventListener('click', handleClickOutside)
  }, [contextMenu])

  const fetchTrash = async () => {
    setLoading(true)
    try {
      const response = await api.get('/api/trash')
      setItems(response.data.files || [])
    } catch (error) {
      console.error('Failed to fetch trash:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleRestore = async (id: string) => {
    setContextMenu(null)
    try {
      await api.post(`/api/files/${id}/restore`)
      await fetchTrash()
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      alert(axiosError.response?.data?.error || '복원 실패')
    }
  }

  const handleDelete = async (id: string) => {
    setContextMenu(null)
    try {
      await api.delete(`/api/files/${id}`, { params: { permanent: true } })
      setDeleteDialog({ isOpen: false })
      await fetchTrash()
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      alert(axiosError.response?.data?.error || '삭제 실패')
    }
  }

  const handleContextMenu = (e: React.MouseEvent, item: TrashItem) => {
    e.preventDefault()
    setContextMenu({ x: e.clientX, y: e.clientY, item })
  }

  const handleEmptyTrash = async () => {
    try {
      await api.delete('/api/trash')
      setEmptyTrashDialog(false)
      await fetchTrash()
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      alert(axiosError.response?.data?.error || '휴지통 비우기 실패')
    }
  }

  const toggleSelect = (id: string) => {
    const newSelected = new Set(selectedItems)
    if (newSelected.has(id)) {
      newSelected.delete(id)
    } else {
      newSelected.add(id)
    }
    setSelectedItems(newSelected)
  }

  return (
    <div className="h-full flex flex-col">
      <header className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Trash2 className="w-6 h-6 text-red-500" />
            <h1 className="text-2xl font-bold text-gray-900">휴지통</h1>
          </div>
          {items.length > 0 && (
            <button
              onClick={() => setEmptyTrashDialog(true)}
              className="flex items-center gap-2 px-4 py-2 text-red-600 border border-red-300 rounded-lg hover:bg-red-50"
            >
              <Trash2 className="w-4 h-4" />
              <span>휴지통 비우기</span>
            </button>
          )}
        </div>
      </header>

      <div className="flex-1 overflow-auto bg-white">
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
          </div>
        ) : items.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 text-gray-500">
            <Trash2 className="w-16 h-16 mb-4 text-gray-300" />
            <p>휴지통이 비어있습니다</p>
          </div>
        ) : (
          <>
            <div className="px-6 py-3 bg-yellow-50 border-b flex items-center gap-2 text-yellow-800">
              <AlertTriangle className="w-4 h-4" />
              <span className="text-sm">휴지통의 항목은 30일 후 자동으로 삭제됩니다.</span>
            </div>
            <table className="w-full">
              <thead className="bg-gray-50 border-b">
                <tr>
                  <th className="w-8 px-4 py-3"></th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-600">이름</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-600 w-32">크기</th>
                  <th className="px-4 py-3 text-left text-sm font-medium text-gray-600 w-40">삭제일</th>
                  <th className="px-4 py-3 w-24"></th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr 
                    key={item.id} 
                    className="border-b hover:bg-gray-50 cursor-pointer"
                    onContextMenu={(e) => handleContextMenu(e, item)}
                  >
                    <td className="px-4 py-2">
                      <input
                        type="checkbox"
                        checked={selectedItems.has(item.id)}
                        onChange={() => toggleSelect(item.id)}
                        className="rounded border-gray-300"
                      />
                    </td>
                    <td className="px-4 py-2">
                      <div className="flex items-center gap-3">
                        {item.is_directory ? (
                          <Folder className="w-10 h-10 text-yellow-500 opacity-50" />
                        ) : (
                          <File className="w-10 h-10 text-gray-400" />
                        )}
                        <span className="text-gray-600">{item.name}</span>
                      </div>
                    </td>
                    <td className="px-4 py-2 text-sm text-gray-600">
                      {item.is_directory ? '-' : formatBytes(item.size)}
                    </td>
                    <td className="px-4 py-2 text-sm text-gray-600">
                      {item.trashed_at ? format(new Date(item.trashed_at), 'yyyy-MM-dd HH:mm') : '-'}
                    </td>
                    <td className="px-4 py-2">
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => handleRestore(item.id)}
                          className="p-2 hover:bg-gray-200 rounded"
                          title="복원"
                        >
                          <RotateCcw className="w-4 h-4 text-gray-500" />
                        </button>
                        <button
                          onClick={() => setDeleteDialog({ isOpen: true, item })}
                          className="p-2 hover:bg-gray-200 rounded"
                          title="영구 삭제"
                        >
                          <Trash2 className="w-4 h-4 text-red-500" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        )}
      </div>
      <ConfirmDialog
        isOpen={deleteDialog.isOpen}
        title="영구 삭제"
        message={`"${deleteDialog.item?.name}"을(를) 영구적으로 삭제하시겠습니까? 이 작업은 되돌릴 수 없습니다.`}
        confirmText="삭제"
        cancelText="취소"
        variant="danger"
        onConfirm={() => deleteDialog.item && handleDelete(deleteDialog.item.id)}
        onCancel={() => setDeleteDialog({ isOpen: false })}
      />

      <ConfirmDialog
        isOpen={emptyTrashDialog}
        title="휴지통 비우기"
        message="휴지통의 모든 항목을 영구적으로 삭제하시겠습니까? 이 작업은 되돌릴 수 없습니다."
        confirmText="비우기"
        cancelText="취소"
        variant="danger"
        onConfirm={handleEmptyTrash}
        onCancel={() => setEmptyTrashDialog(false)}
      />

      {contextMenu && (
        <div
          ref={contextMenuRef}
          className="fixed bg-white border rounded-lg shadow-lg py-2 min-w-[160px] z-50"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <button
            onClick={() => handleRestore(contextMenu.item.id)}
            className="flex items-center gap-2 w-full px-4 py-2 hover:bg-gray-100 text-left"
          >
            <RotateCcw className="w-4 h-4" />
            <span>복원</span>
          </button>
          <hr className="my-1" />
          <button
            onClick={() => {
              setDeleteDialog({ isOpen: true, item: contextMenu.item })
              setContextMenu(null)
            }}
            className="flex items-center gap-2 w-full px-4 py-2 hover:bg-gray-100 text-left text-red-600"
          >
            <Trash2 className="w-4 h-4" />
            <span>영구 삭제</span>
          </button>
        </div>
      )}
    </div>
  )
}
