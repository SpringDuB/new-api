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
import { expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { TaskArtifactStoreSection } from '../general/task-artifact-store-section'

const config = {
  mode: 'oss',
  endpoint: 'https://oss-cn-hangzhou.aliyuncs.com',
  bucket: 'results',
  region: 'cn-hangzhou',
  prefix: 'tasks/v1',
  presign_ttl: 900,
  max_mb: 512,
  credentials_configured: true,
}

function renderSettings() {
  return render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <TaskArtifactStoreSection />
    </QueryClientProvider>
  )
}

test('OSS settings load without secrets, preserve existing credentials, save and test the connection', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: config },
  })
  const put = vi.spyOn(api, 'put').mockResolvedValue({
    data: { success: true, data: { ...config, bucket: 'new-results' } },
  })
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValue({ data: { success: true } })
  renderSettings()
  const bucket = await screen.findByRole('textbox', { name: 'Bucket' })
  expect(screen.getByLabelText('AccessKey Secret')).toHaveValue('')
  expect(
    screen.getByText('Credentials configured; leave blank to keep them')
  ).toBeVisible()
  fireEvent.change(bucket, { target: { value: 'new-results' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }))
  await waitFor(() =>
    expect(put).toHaveBeenCalledWith(
      '/api/option/task_artifact_store',
      expect.objectContaining({
        bucket: 'new-results',
        access_key: '',
        secret_key: '',
      })
    )
  )
  fireEvent.click(
    await screen.findByRole('button', { name: 'Test connection' })
  )
  await waitFor(() =>
    expect(post).toHaveBeenCalledWith('/api/option/task_artifact_store/test')
  )
  expect(await screen.findByText('OSS connection test passed')).toBeVisible()
})

test('disabled storage keeps credential inputs and the connection test disabled', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { ...config, mode: 'upstream' } },
  })
  renderSettings()
  expect(await screen.findByRole('textbox', { name: 'Bucket' })).toBeDisabled()
  expect(screen.getByLabelText('AccessKey Secret')).toBeDisabled()
  expect(screen.getByRole('button', { name: 'Test connection' })).toBeDisabled()
  fireEvent.click(
    screen.getByRole('switch', { name: 'Enable Aliyun OSS storage' })
  )
  expect(screen.getByRole('textbox', { name: 'Bucket' })).toBeEnabled()
  expect(screen.getByRole('button', { name: 'Test connection' })).toBeDisabled()
})

test('loading settings hide the form and a failed load can be retried', async () => {
  let rejectLoad!: (reason: Error) => void
  const get = vi.spyOn(api, 'get').mockImplementationOnce(
    () =>
      new Promise((_resolve, reject) => {
        rejectLoad = reject
      })
  )
  renderSettings()
  expect(screen.getByText('Loading...')).toBeVisible()
  expect(screen.queryByLabelText('AccessKey Secret')).not.toBeInTheDocument()
  rejectLoad(new Error('network unavailable'))
  get.mockResolvedValue({ data: { success: true, data: config } })
  fireEvent.click(await screen.findByRole('button', { name: 'Retry' }))
  expect(await screen.findByRole('textbox', { name: 'Bucket' })).toHaveValue(
    'results'
  )
})

test('an invalid enabled configuration shows validation feedback and is not saved', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: config },
  })
  const put = vi.spyOn(api, 'put')
  renderSettings()
  fireEvent.change(await screen.findByRole('textbox', { name: 'Bucket' }), {
    target: { value: '' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }))
  expect(
    await screen.findByText('Invalid task artifact storage configuration')
  ).toBeVisible()
  expect(put).not.toHaveBeenCalled()
})

test('a rejected connection test does not report success and can be retried', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: config },
  })
  const post = vi
    .spyOn(api, 'post')
    .mockResolvedValueOnce({
      data: {
        success: false,
        message:
          'OSS connection test failed; check endpoint, bucket and permissions',
        message_key:
          'OSS connection test failed at {{operation}} (HTTP {{status}}, code {{code}}, request ID {{request_id}})',
        message_params: {
          operation: 'PutObject',
          status: 403,
          code: 'AccessDenied',
          request_id: 'request-test-123',
        },
      },
    })
    .mockResolvedValueOnce({ data: { success: true } })
  renderSettings()
  fireEvent.click(
    await screen.findByRole('button', { name: 'Test connection' })
  )
  await waitFor(() => expect(post).toHaveBeenCalledTimes(1))
  const failure = await screen.findByRole('alert')
  expect(failure).toHaveTextContent('PutObject')
  expect(failure).toHaveTextContent('403')
  expect(failure).toHaveTextContent('AccessDenied')
  expect(failure).toHaveTextContent('request-test-123')
  await waitFor(() =>
    expect(
      screen.getByRole('button', { name: 'Test connection' })
    ).toBeEnabled()
  )
  expect(
    screen.queryByText('OSS connection test passed')
  ).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Test connection' }))
  expect(await screen.findByText('OSS connection test passed')).toBeVisible()
  expect(screen.queryByRole('alert')).not.toBeInTheDocument()
})
