/**
 * User Announcements API endpoints
 */

import { apiClient } from './client'
import type { BasePaginationResponse, FetchOptions, UserAnnouncement } from '@/types'

export async function list(unreadOnly: boolean = false): Promise<UserAnnouncement[]> {
  const { data } = await apiClient.get<UserAnnouncement[]>('/announcements', {
    params: unreadOnly ? { unread_only: 1 } : {}
  })
  return data
}

/**
 * Paginated archive of everything this user was eligible for, including archived
 * and expired announcements.
 *
 * The signature matches useTableLoader's fetchFn contract.
 */
export async function listArchive(
  page: number = 1,
  pageSize: number = 20,
  params?: { unread_only?: boolean; search?: string },
  options?: FetchOptions
): Promise<BasePaginationResponse<UserAnnouncement>> {
  const { data } = await apiClient.get<BasePaginationResponse<UserAnnouncement>>(
    '/announcements/archive',
    {
      params: {
        page,
        page_size: pageSize,
        ...(params?.unread_only ? { unread_only: 1 } : {}),
        ...(params?.search ? { search: params.search } : {}),
      },
      signal: options?.signal,
    }
  )
  return data
}

export async function markRead(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.post<{ message: string }>(`/announcements/${id}/read`)
  return data
}

const announcementsAPI = {
  list,
  listArchive,
  markRead
}

export default announcementsAPI

