import { useState, useEffect, useCallback } from 'react'
import { FileItem } from '../stores/fileStore'
import { api } from '../lib/api'
import BaseFilePreview, { getPreviewCategory } from './BaseFilePreview'

interface FilePreviewProps {
  file: FileItem
  files?: FileItem[]
  onClose: () => void
  onFileChange?: (file: FileItem) => void
}

export default function FilePreview({
  file,
  files = [],
  onClose,
  onFileChange,
}: FilePreviewProps) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const [textContent, setTextContent] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const category = getPreviewCategory(file.mime_type)
  const canPreview = category !== 'none'

  const loadPreview = useCallback(async () => {
    if (!canPreview) {
      setIsLoading(false)
      return
    }

    setIsLoading(true)
    setError(null)
    setPreviewUrl(null)
    setTextContent(null)

    try {
      const response = await api.get(`/api/files/${file.id}/download`, {
        responseType: category === 'text' ? 'text' : 'blob',
      })

      if (category === 'text') {
        setTextContent(response.data)
      } else {
        const url = URL.createObjectURL(response.data)
        setPreviewUrl(url)
      }
    } catch (err) {
      setError('파일을 불러올 수 없습니다.')
      console.error('Failed to load preview:', err)
    } finally {
      setIsLoading(false)
    }
  }, [file.id, canPreview, category])

  useEffect(() => {
    loadPreview()
    return () => {
      if (previewUrl) {
        URL.revokeObjectURL(previewUrl)
      }
    }
  }, [file.id]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleDownload = async () => {
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

  return (
    <BaseFilePreview
      file={file}
      files={files}
      onClose={onClose}
      onFileChange={onFileChange}
      onDownload={handleDownload}
      onRetry={loadPreview}
      previewUrl={previewUrl}
      textContent={textContent}
      isLoading={isLoading}
      error={error}
    />
  )
}
