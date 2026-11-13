/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { TaskClientRequestDetails } from '../task-client-request'

function renderRequest(open = true) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <TaskClientRequestDetails taskId='task-public' open={open} />
    </QueryClientProvider>
  )
}

describe('task client request details', () => {
  test('a closed details view does not load the request body', () => {
    const get = vi.spyOn(api, 'get')
    renderRequest(false)
    expect(get).not.toHaveBeenCalled()
  })

  test('a historical task shows not recorded without using upstream results', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { recorded: false, snapshot: null } },
    })
    renderRequest()
    expect(await screen.findByText('Not recorded')).toBeVisible()
    expect(
      screen.queryByRole('button', { name: 'Copy to clipboard' })
    ).not.toBeInTheDocument()
  })

  test('an opened task displays and copies its original redacted request including zero values', async () => {
    const snapshot = {
      method: 'POST',
      path: '/v1/videos',
      content_type: 'application/json',
      body_kind: 'json',
      body: { prompt: 'original-client', seed: 0, enabled: false },
      redacted: true,
      files_omitted: false,
    }
    const get = vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { recorded: true, snapshot } },
    })
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    })
    renderRequest()
    const copy = await screen.findByRole('button', {
      name: 'Copy to clipboard',
    })
    fireEvent.click(copy)
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(JSON.stringify(snapshot, null, 2))
    )
    expect(get).toHaveBeenCalledWith('/api/task/task-public/request')
    expect(screen.getByText('Sensitive values redacted')).toBeVisible()
  })

  test('omitted bodies and multipart file contents are marked', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: {
          recorded: true,
          snapshot: {
            method: 'POST',
            path: '/v1/videos',
            content_type: 'multipart/form-data',
            body_kind: 'multipart',
            body_omitted: 'size_limit',
            files_omitted: true,
          },
        },
      },
    })
    renderRequest()
    expect(
      await screen.findByText('Request body omitted: size limit exceeded')
    ).toBeVisible()
    expect(screen.getByText('File contents omitted')).toBeVisible()
  })

  test('a failed request displays a retry action and recovers', async () => {
    vi.spyOn(api, 'get')
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce({
        data: { success: true, data: { recorded: false, snapshot: null } },
      })
    renderRequest()
    fireEvent.click(await screen.findByRole('button', { name: 'Retry' }))
    expect(await screen.findByText('Not recorded')).toBeVisible()
  })
})
