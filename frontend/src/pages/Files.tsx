import { useState, useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useFileStore, FileItem } from '../stores/fileStore'
import FileList from '../components/FileList'
import UploadModal from '../components/UploadModal'
import {
  Upload,
  FolderPlus,
  Search,
  ChevronRight,
  Home,
  RefreshCw,
} from 'lucide-react'

export default function Files() {
  const location = useLocation()
  const navigate = useNavigate()
  const { files, loading, fetchFiles, createFolder, currentPath } = useFileStore()
  const [showUpload, setShowUpload] = useState(false)
  const [showNewFolder, setShowNewFolder] = useState(false)
  const [newFolderName, setNewFolderName] = useState('')
  const [searchQuery, setSearchQuery] = useState('')

  const pathFromUrl = location.pathname.replace('/files', '') || '/'

  useEffect(() => {
    fetchFiles(pathFromUrl)
  }, [pathFromUrl]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleNavigate = (file: FileItem) => {
    if (file.is_directory) {
      const fullPath = file.path === '/' ? `/${file.name}` : `${file.path}/${file.name}`
      navigate(`/files${fullPath}`)
    }
  }

  const handleCreateFolder = async () => {
    if (!newFolderName.trim()) return
    try {
      await createFolder(newFolderName, currentPath)
      setNewFolderName('')
      setShowNewFolder(false)
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to create folder'
      alert(errorMessage)
    }
  }

  const breadcrumbs = pathFromUrl.split('/').filter(Boolean)

  const filteredFiles = files.filter((file) =>
    file.name.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <div className="h-full flex flex-col">
      <header className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-2xl font-bold text-gray-900">내 파일</h1>
          <div className="flex items-center gap-2">
            <button
              onClick={() => fetchFiles(currentPath)}
              className="p-2 hover:bg-gray-100 rounded-lg"
              title="새로고침"
            >
              <RefreshCw className="w-5 h-5 text-gray-600" />
            </button>
            <button
              onClick={() => setShowNewFolder(true)}
              className="flex items-center gap-2 px-4 py-2 border rounded-lg hover:bg-gray-100"
            >
              <FolderPlus className="w-5 h-5" />
              <span>새 폴더</span>
            </button>
            <button
              onClick={() => setShowUpload(true)}
              className="flex items-center gap-2 px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
            >
              <Upload className="w-5 h-5" />
              <span>업로드</span>
            </button>
          </div>
        </div>

        <div className="flex items-center justify-between">
          <nav className="flex items-center gap-1 text-sm">
            <button
              onClick={() => navigate('/files')}
              className="flex items-center gap-1 px-2 py-1 hover:bg-gray-100 rounded"
            >
              <Home className="w-4 h-4" />
              <span>홈</span>
            </button>
            {breadcrumbs.map((crumb, index) => (
              <div key={index} className="flex items-center">
                <ChevronRight className="w-4 h-4 text-gray-400" />
                <button
                  onClick={() =>
                    navigate(`/files/${breadcrumbs.slice(0, index + 1).join('/')}`)
                  }
                  className="px-2 py-1 hover:bg-gray-100 rounded"
                >
                  {crumb}
                </button>
              </div>
            ))}
          </nav>

          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="파일 검색..."
              className="pl-9 pr-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 w-64"
            />
          </div>
        </div>
      </header>

      <div className="flex-1 overflow-auto bg-white">
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
          </div>
        ) : (
          <FileList
            files={filteredFiles}
            onNavigate={handleNavigate}
          />
        )}
      </div>

      {showUpload && (
        <UploadModal
          path={currentPath}
          onClose={() => setShowUpload(false)}
        />
      )}

      {showNewFolder && (
        <div
          className="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
          onClick={() => setShowNewFolder(false)}
        >
          <div
            className="bg-white rounded-lg shadow-xl p-6 w-full max-w-sm"
            onClick={(e) => e.stopPropagation()}
          >
            <h2 className="text-lg font-semibold mb-4">새 폴더 만들기</h2>
            <input
              type="text"
              value={newFolderName}
              onChange={(e) => setNewFolderName(e.target.value)}
              placeholder="폴더 이름"
              className="w-full px-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500 mb-4"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleCreateFolder()
                if (e.key === 'Escape') setShowNewFolder(false)
              }}
            />
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowNewFolder(false)}
                className="px-4 py-2 border rounded-lg hover:bg-gray-100"
              >
                취소
              </button>
              <button
                onClick={handleCreateFolder}
                className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
              >
                만들기
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
