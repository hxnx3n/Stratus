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
