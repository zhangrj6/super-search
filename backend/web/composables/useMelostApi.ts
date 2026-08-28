import { parseApiResponse } from './useApi'
import { useApiFetch } from './useApiFetch'

export interface MelostSearchItem {
  doc_id: string
  disk_name: string
  disk_type: string
  link: string
  disk_pass: string
  files: string
  tags: string[]
  shared_time: string
  share_user: string
  size: number
  can_stage: boolean
  stage_message?: string
}

export interface MelostSearchResponse {
  total: number
  page: number
  page_size: number
  took: number
  items: MelostSearchItem[]
}

export interface MelostStageResponse {
  status: 'staged'
  existing: boolean
  resource_id: number
  resource_key: string
}

export const useMelostApi = () => {
  const search = (query: string, page = 1, size = 20, type = '') =>
    useApiFetch('/melost/search', {
      method: 'POST',
      body: { q: query, type, page, size }
    }).then(parseApiResponse<MelostSearchResponse>)

  const stageResource = (item: MelostSearchItem) =>
    useApiFetch('/melost/resources', {
      method: 'POST',
      body: {
        doc_id: item.doc_id,
        title: item.disk_name,
        link: item.link,
        disk_type: item.disk_type,
        disk_pass: item.disk_pass,
        files: item.files,
        tags: item.tags,
        shared_time: item.shared_time,
        share_user: item.share_user,
        size: item.size
      }
    }).then(parseApiResponse<MelostStageResponse>)

  return { search, stageResource }
}
