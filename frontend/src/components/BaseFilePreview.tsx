import { useEffect } from 'react'
import { X, Download, ChevronLeft, ChevronRight, Loader2 } from 'lucide-react'
import { formatBytes } from '../lib/utils'

export interface PreviewFile {
  id: string
  name: string
  size: number
  mime_type: string
  is_directory: boolean
}

export interface BaseFilePreviewProps<T extends PreviewFile> {
  file: T
  files?: T[]
  onClose: () => void
  onFileChange?: (file: T) => void
  onDownload: () => void
  onRetry: () => void
  previewUrl: string | null
  textContent: string | null
  isLoading: boolean
  error: string | null
}

export type PreviewCategory = 'image' | 'video' | 'audio' | 'pdf' | 'text' | 'none'

export const getPreviewCategory = (mimeType: string): PreviewCategory => {
  if (!mimeType) return 'none'
  if (mimeType.startsWith('image/')) return 'image'
  if (mimeType.startsWith('video/')) return 'video'
  if (mimeType.startsWith('audio/')) return 'audio'
  if (mimeType === 'application/pdf') return 'pdf'
  if (mimeType.startsWith('text/') ||
    mimeType === 'application/json' ||
    mimeType === 'application/javascript' ||
    mimeType === 'application/xml') return 'text'
  return 'none'
}

export default function BaseFilePreview<T extends PreviewFile>({
  file,
  files = [],
  onClose,
  onFileChange,
  onDownload,
  onRetry,
  previewUrl,
  textContent,
  isLoading,
  error,
}: BaseFilePreviewProps<T>) {
  const category = getPreviewCategory(file.mime_type)
  const canPreview = category !== 'none'

  // Filter previewable files
  const previewableFiles = files.filter(f => !f.is_directory && getPreviewCategory(f.mime_type) !== 'none')
  const currentIndex = previewableFiles.findIndex(f => f.id === file.id)
  const hasPrev = currentIndex > 0
  const hasNext = currentIndex < previewableFiles.length - 1

  const handleBackdropClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose()
    }
  }

  const handlePrev = () => {
    if (hasPrev && onFileChange) {
      onFileChange(previewableFiles[currentIndex - 1])
    }
  }

  const handleNext = () => {
    if (hasNext && onFileChange) {
      onFileChange(previewableFiles[currentIndex + 1])
    }
  }

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowLeft' && hasPrev) handlePrev()
      if (e.key === 'ArrowRight' && hasNext) handleNext()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [hasPrev, hasNext]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80"
      onClick={handleBackdropClick}
    >
      {/* Header */}
      <div className="absolute top-0 left-0 right-0 flex items-center justify-between px-4 py-3 bg-gradient-to-b from-black/50 to-transparent">
        <div className="flex items-center gap-3 text-white">
          <h3 className="text-lg font-medium truncate max-w-md">{file.name}</h3>
          <span className="text-sm text-gray-300">
            {formatBytes(file.size)}
          </span>
          {previewableFiles.length > 1 && (
            <span className="text-sm text-gray-400">
              {currentIndex + 1} / {previewableFiles.length}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onDownload}
            className="p-2 text-white hover:bg-white/20 rounded-full transition-colors"
            title="다운로드"
          >
            <Download className="h-5 w-5" />
          </button>
          <button
            type="button"
            onClick={onClose}
            className="p-2 text-white hover:bg-white/20 rounded-full transition-colors"
            title="닫기"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
      </div>

      {/* Prev button */}
      {hasPrev && (
        <button
          type="button"
          onClick={handlePrev}
          className="absolute left-4 top-1/2 -translate-y-1/2 p-3 text-white bg-black/50 hover:bg-black/70 rounded-full transition-colors"
          title="이전"
        >
          <ChevronLeft className="h-6 w-6" />
        </button>
      )}

      {/* Next button */}
      {hasNext && (
        <button
          type="button"
          onClick={handleNext}
          className="absolute right-4 top-1/2 -translate-y-1/2 p-3 text-white bg-black/50 hover:bg-black/70 rounded-full transition-colors"
          title="다음"
        >
          <ChevronRight className="h-6 w-6" />
        </button>
      )}

      {/* Preview content */}
      <div className="flex items-center justify-center w-full h-full p-16">
        {isLoading ? (
          <div className="flex flex-col items-center gap-4 text-white">
            <Loader2 className="h-12 w-12 animate-spin" />
            <p>로딩 중...</p>
          </div>
        ) : error ? (
          <div className="flex flex-col items-center gap-4 text-white">
            <p className="text-red-400">{error}</p>
            <button
              type="button"
              onClick={onRetry}
              className="px-4 py-2 bg-white/20 hover:bg-white/30 rounded-lg transition-colors"
            >
              다시 시도
            </button>
          </div>
        ) : !canPreview ? (
          <div className="flex flex-col items-center gap-4 text-white">
            <p>이 파일은 미리보기를 지원하지 않습니다</p>
            <button
              type="button"
              onClick={onDownload}
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
            >
              <Download className="h-4 w-4" />
              다운로드
            </button>
          </div>
        ) : (
          <>
            {category === 'image' && previewUrl && (
              <img
                src={previewUrl}
                alt={file.name}
                className="max-w-full max-h-full object-contain"
              />
            )}

            {category === 'video' && previewUrl && (
              <video
                src={previewUrl}
                controls
                autoPlay
                className="max-w-full max-h-full"
              >
                <track kind="captions" />
              </video>
            )}

            {category === 'audio' && previewUrl && (
              <div className="flex flex-col items-center gap-6 p-8 bg-white/10 rounded-xl">
                <div className="w-32 h-32 bg-gradient-to-br from-purple-500 to-pink-500 rounded-full flex items-center justify-center">
                  <span className="text-4xl">🎵</span>
                </div>
                <p className="text-white text-lg font-medium">{file.name}</p>
                <audio src={previewUrl} controls autoPlay className="w-80">
                  <track kind="captions" />
                </audio>
              </div>
            )}

            {category === 'pdf' && previewUrl && (
              <iframe
                src={previewUrl}
                title={file.name}
                className="w-full h-full bg-white rounded-lg"
              />
            )}

            {category === 'text' && textContent !== null && (
              <div className="w-full h-full bg-gray-900 rounded-lg overflow-auto">
                <pre className="p-4 text-gray-100 text-sm font-mono whitespace-pre-wrap">
                  {textContent}
                </pre>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
