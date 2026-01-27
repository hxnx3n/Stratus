import { useState, useCallback } from 'react'
import { Upload, X } from 'lucide-react'
import { useFileStore } from '../stores/fileStore'

interface UploadModalProps {
  parentId?: string
  onClose: () => void
}

interface UploadingFile {
  file: File
  progress: number
  status: 'pending' | 'uploading' | 'done' | 'error'
  error?: string
}

export default function UploadModal({ parentId, onClose }: UploadModalProps) {
  const { uploadFile } = useFileStore()
  const [files, setFiles] = useState<UploadingFile[]>([])
  const [isDragging, setIsDragging] = useState(false)

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
    const droppedFiles = Array.from(e.dataTransfer.files)
    addFiles(droppedFiles)
  }, [])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }, [])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
  }, [])

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      const selectedFiles = Array.from(e.target.files)
      addFiles(selectedFiles)
    }
  }

  const addFiles = (newFiles: File[]) => {
    const uploadingFiles: UploadingFile[] = newFiles.map((file) => ({
      file,
      progress: 0,
      status: 'pending',
    }))
    setFiles((prev) => [...prev, ...uploadingFiles])
  }

  const removeFile = (index: number) => {
    setFiles((prev) => prev.filter((_, i) => i !== index))
  }

  const startUpload = async () => {
    for (let i = 0; i < files.length; i++) {
      if (files[i].status !== 'pending') continue

      setFiles((prev) =>
        prev.map((f, idx) =>
          idx === i ? { ...f, status: 'uploading' } : f
        )
      )

      try {
        await uploadFile(files[i].file, parentId, (progress) => {
          setFiles((prev) =>
            prev.map((f, idx) =>
              idx === i ? { ...f, progress } : f
            )
          )
        })

        setFiles((prev) =>
          prev.map((f, idx) =>
            idx === i ? { ...f, status: 'done', progress: 100 } : f
          )
        )
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : 'Upload failed'
        setFiles((prev) =>
          prev.map((f, idx) =>
            idx === i ? { ...f, status: 'error', error: errorMessage } : f
          )
        )
      }
    }
  }

  const allDone = files.length > 0 && files.every((f) => f.status === 'done')
  const hasFiles = files.length > 0

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between p-4 border-b">
          <h2 className="text-lg font-semibold">파일 업로드</h2>
          <button onClick={onClose} className="p-1 hover:bg-gray-100 rounded">
            <X className="w-5 h-5" />
          </button>
        </div>

        <div className="p-4">
          <div
            onDrop={handleDrop}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            className={`border-2 border-dashed rounded-lg p-8 text-center transition-colors ${
              isDragging ? 'border-primary-500 bg-primary-50' : 'border-gray-300'
            }`}
          >
            <Upload className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <p className="text-gray-600 mb-2">파일을 여기에 드래그하거나</p>
            <label className="cursor-pointer text-primary-600 hover:text-primary-700 font-medium">
              파일 선택
              <input
                type="file"
                multiple
                onChange={handleFileSelect}
                className="hidden"
              />
            </label>
          </div>

          {hasFiles && (
            <div className="mt-4 space-y-2 max-h-60 overflow-y-auto">
              {files.map((f, index) => (
                <div
                  key={index}
                  className="flex items-center gap-3 p-2 bg-gray-50 rounded"
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{f.file.name}</p>
                    <div className="w-full bg-gray-200 rounded-full h-1.5 mt-1">
                      <div
                        className={`h-1.5 rounded-full transition-all ${
                          f.status === 'error' ? 'bg-red-500' : 'bg-primary-600'
                        }`}
                        style={{ width: `${f.progress}%` }}
                      />
                    </div>
                    {f.error && (
                      <p className="text-xs text-red-500 mt-1">{f.error}</p>
                    )}
                  </div>
                  <span className="text-xs text-gray-500">
                    {f.status === 'done' && '완료'}
                    {f.status === 'uploading' && `${f.progress}%`}
                    {f.status === 'error' && '오류'}
                    {f.status === 'pending' && '대기중'}
                  </span>
                  {f.status === 'pending' && (
                    <button
                      onClick={() => removeFile(index)}
                      className="p-1 hover:bg-gray-200 rounded"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2 p-4 border-t">
          <button
            onClick={onClose}
            className="px-4 py-2 border rounded-lg hover:bg-gray-100"
          >
            {allDone ? '닫기' : '취소'}
          </button>
          {!allDone && hasFiles && (
            <button
              onClick={startUpload}
              className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700"
            >
              업로드
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
