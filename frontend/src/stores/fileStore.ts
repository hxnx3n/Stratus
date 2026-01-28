import { create } from 'zustand'
import { api } from '../lib/api'
import { AxiosError } from 'axios'
import { useAuthStore } from './authStore'

interface ApiError {
  error?: string
}

export interface FileItem {
  id: string
  name: string
  path: string
  size: number
  mime_type: string
  is_directory: boolean
  created_at: string
  updated_at: string
}

interface FileState {
  files: FileItem[]
  currentPath: string
  selectedFiles: Set<string>
  loading: boolean
  error: string | null
  fetchFiles: (path?: string) => Promise<void>
  createFolder: (name: string, path?: string) => Promise<void>
  uploadFile: (file: File, path?: string, onProgress?: (progress: number) => void) => Promise<void>
  deleteFile: (id: string) => Promise<void>
  moveToTrash: (id: string) => Promise<void>
  restoreFromTrash: (id: string) => Promise<void>
  renameFile: (id: string, newName: string) => Promise<void>
  selectFile: (id: string) => void
  deselectFile: (id: string) => void
  clearSelection: () => void
  setCurrentPath: (path: string) => void
}

export const useFileStore = create<FileState>((set, get) => ({
  files: [],
  currentPath: '/',
  selectedFiles: new Set(),
  loading: false,
  error: null,

  fetchFiles: async (path = '/') => {
    set({ loading: true, error: null })
    try {
      const response = await api.get('/api/files', { params: { path } })
      set({ 
        files: response.data.files || [], 
        currentPath: path
      })
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      set({ error: axiosError.response?.data?.error || 'Failed to fetch files' })
    } finally {
      set({ loading: false })
    }
  },

  createFolder: async (name: string, path?: string) => {
    try {
      await api.post('/api/files/folder', { name, path: path || get().currentPath })
      await get().fetchFiles(get().currentPath)
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      throw new Error(axiosError.response?.data?.error || 'Failed to create folder')
    }
  },

  uploadFile: async (file: File, path?: string, onProgress?: (progress: number) => void) => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('path', path || get().currentPath)

    try {
      await api.post('/api/files/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (progressEvent) => {
          if (progressEvent.total && onProgress) {
            const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total)
            onProgress(progress)
          }
        },
      })
      await get().fetchFiles(get().currentPath)
      await useAuthStore.getState().fetchUser()
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      throw new Error(axiosError.response?.data?.error || 'Failed to upload file')
    }
  },

  deleteFile: async (id: string) => {
    try {
      await api.delete(`/api/files/${id}`, { params: { permanent: true } })
      await get().fetchFiles(get().currentPath)
      await useAuthStore.getState().fetchUser()
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      throw new Error(axiosError.response?.data?.error || 'Failed to delete file')
    }
  },

  moveToTrash: async (id: string) => {
    try {
      await api.delete(`/api/files/${id}/trash`)
      await get().fetchFiles(get().currentPath)
      await useAuthStore.getState().fetchUser()
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      throw new Error(axiosError.response?.data?.error || 'Failed to move to trash')
    }
  },

  restoreFromTrash: async (id: string) => {
    try {
      await api.post(`/api/files/${id}/restore`)
      await get().fetchFiles(get().currentPath)
      await useAuthStore.getState().fetchUser()
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      throw new Error(axiosError.response?.data?.error || 'Failed to restore file')
    }
  },

  renameFile: async (id: string, newName: string) => {
    try {
      await api.put(`/api/files/${id}/rename`, { name: newName })
      await get().fetchFiles(get().currentPath)
    } catch (error) {
      const axiosError = error as AxiosError<ApiError>
      throw new Error(axiosError.response?.data?.error || 'Failed to rename file')
    }
  },

  selectFile: (id: string) => {
    const newSelected = new Set(get().selectedFiles)
    newSelected.add(id)
    set({ selectedFiles: newSelected })
  },

  deselectFile: (id: string) => {
    const newSelected = new Set(get().selectedFiles)
    newSelected.delete(id)
    set({ selectedFiles: newSelected })
  },

  clearSelection: () => {
    set({ selectedFiles: new Set() })
  },

  setCurrentPath: (path: string) => {
    set({ currentPath: path })
  },
}))
