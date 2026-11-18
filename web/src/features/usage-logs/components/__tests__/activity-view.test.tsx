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
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { usageLogSchema } from '../../data/schema'
import type { LogCategory, TaskLog } from '../../types'
import { UsageLogsProvider, useUsageLogsContext } from '../usage-logs-provider'
import { UsageLogsTable } from '../usage-logs-table'

async function renderActivity(
  options: {
    mobile?: boolean
    category?: LogCategory
    admin?: boolean
    search?: string
  } = {}
) {
  const category = options.category ?? 'common'
  vi.spyOn(window, 'scrollTo').mockImplementation(() => {})
  const originalMatchMedia = window.matchMedia
  vi.spyOn(window, 'matchMedia').mockImplementation((query) => ({
    ...originalMatchMedia(query),
    matches: query === '(max-width: 640px)' && !!options.mobile,
  }))
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'viewer',
    role: options.admin ? 10 : 1,
  })
  const commonLog = usageLogSchema.parse({
    id: 1,
    user_id: 7,
    created_at: 1788840000,
    type: 2,
    content: '',
    model_name: 'story-model',
    username: 'studio-owner',
    token_name: 'production-key',
    channel: 42,
    channel_name: 'private-channel',
    group: 'premium',
    quota: 5000,
    other: '{}',
  })
  const task: TaskLog = {
    id: 2,
    user_id: 7,
    username: 'studio-owner',
    platform: 'video',
    task_id: 'story-task',
    action: 'video_generation',
    channel_id: 42,
    group: 'premium',
    quota: 5000,
    submit_time: 1788840000,
    finish_time: 1788840020,
    status: 'SUCCESS',
    properties: { origin_model_name: 'story-model' },
    image_artifacts_available: true,
  }
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    let data: unknown = { quota: 0, rpm: 0, tpm: 0 }
    if (/^\/api\/(log|task|image-log)(\/self)?\?/.test(url)) {
      data = {
        items: [category === 'common' ? commonLog : task],
        total: 1,
        page: 1,
        page_size: 100,
      }
    } else if (url === '/api/subscription/self') {
      data = { subscriptions: [] }
    } else if (
      url === '/api/subscription/admin/plans' ||
      url === '/api/group/'
    ) {
      data = []
    } else if (url === '/api/user/self/groups') {
      data = { premium: { desc: '', ratio: 1 } }
    } else if (url.includes('/client-request')) {
      data = { recorded: false, snapshot: null }
    } else if (url.includes('/response')) {
      data = { recorded: false, snapshot: null }
    }
    return { data: { success: true, data } }
  })
  const root = createRootRoute()
  const auth = createRoute({ getParentRoute: () => root, id: '_authenticated' })
  const logs = createRoute({
    getParentRoute: () => auth,
    path: '/usage-logs/$section',
    component: function ActivityFixture() {
      const context = useUsageLogsContext()
      return (
        <>
          <button type='button' onClick={() => context.setViewScope('self')}>
            Only my activity
          </button>
          <button
            type='button'
            onClick={() => context.setSensitiveVisible(false)}
          >
            Hide private details
          </button>
          <UsageLogsTable logCategory={category} />
        </>
      )
    },
    validateSearch: (search: Record<string, unknown>) => search,
  })
  const router = createRouter({
    routeTree: root.addChildren([auth.addChildren([logs])]),
    history: createMemoryHistory({
      initialEntries: [`/usage-logs/${category}${options.search ?? ''}`],
    }),
  })
  render(
    <QueryClientProvider
      client={
        new QueryClient({
          defaultOptions: { queries: { retry: false } },
        })
      }
    >
      <UsageLogsProvider>
        <RouterProvider router={router} />
      </UsageLogsProvider>
    </QueryClientProvider>
  )
  await screen.findByText('story-model')
  return router
}

afterEach(() => {
  vi.restoreAllMocks()
  useAuthStore.setState(useAuthStore.getInitialState(), true)
  localStorage.clear()
})

it.each([false, true])(
  'defaults to readable usage records and opens full details on mobile=%s',
  async (mobile) => {
    await renderActivity({ mobile })
    const record = screen.getByRole('article', { name: 'Activity record' })
    expect(within(record).getByText('story-model')).toBeVisible()
    expect(within(record).getByText('$0.01')).toBeVisible()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Card view' })).toHaveAttribute(
      'aria-pressed',
      'true'
    )
    await userEvent.click(
      within(record).getByTitle('Click to view full details')
    )
    expect(
      await screen.findByRole('dialog', { name: /Log Details/ })
    ).toBeVisible()
  }
)

it('masks administrator task amounts and removes privileged fields after switching to personal scope', async () => {
  await renderActivity({ category: 'task', admin: true })
  const record = screen.getByRole('article', { name: 'Activity record' })
  expect(within(record).getByText('studio-owner')).toBeVisible()
  expect(within(record).getByText('$0.01')).toBeVisible()
  await userEvent.click(
    screen.getByRole('button', { name: 'Hide private details' })
  )
  expect(within(record).queryByText('studio-owner')).not.toBeInTheDocument()
  expect(within(record).queryByText('$0.01')).not.toBeInTheDocument()
  await userEvent.click(
    screen.getByRole('button', { name: 'Only my activity' })
  )
  const personalRecord = await screen.findByRole('article', {
    name: 'Activity record',
  })
  expect(within(personalRecord).queryByText('User')).not.toBeInTheDocument()
  expect(within(personalRecord).queryByText('Channel')).not.toBeInTheDocument()
  expect(within(personalRecord).queryByText('Cost')).not.toBeInTheDocument()
})

it('keeps applied URL filters and unapplied model input when switching to the detailed table', async () => {
  const router = await renderActivity({ search: '?group=premium' })
  await userEvent.type(screen.getByPlaceholderText('Model Name'), 'draft-model')
  await userEvent.click(screen.getByRole('button', { name: 'Table view' }))
  expect(await screen.findByRole('table')).toBeVisible()
  expect(screen.getByPlaceholderText('Model Name')).toHaveValue('draft-model')
  expect(router.state.location.search).toMatchObject({ group: 'premium' })
  expect(router.state.location.search).not.toHaveProperty('model')
  await userEvent.click(screen.getByRole('button', { name: 'Search' }))
  await waitFor(() =>
    expect(router.state.location.search).toMatchObject({
      group: 'premium',
      model: 'draft-model',
      page: 1,
    })
  )
  await userEvent.click(screen.getByRole('button', { name: 'Card view' }))
  expect(screen.getByRole('article', { name: 'Activity record' })).toBeVisible()
  expect(screen.queryByRole('table')).not.toBeInTheDocument()
})

it('keeps secondary filters collapsed until requested and preserves drafts across collapse', async () => {
  const router = await renderActivity()
  expect(
    screen.queryByRole('combobox', { name: 'Group' })
  ).not.toBeInTheDocument()
  const toggle = screen.getByRole('button', { name: /^Filter/ })
  expect(toggle).toHaveAttribute('aria-expanded', 'false')
  await userEvent.click(toggle)
  await userEvent.type(screen.getByPlaceholderText('Token Name'), 'draft-key')
  await userEvent.click(toggle)
  expect(screen.queryByPlaceholderText('Token Name')).not.toBeInTheDocument()
  await userEvent.click(toggle)
  expect(screen.getByPlaceholderText('Token Name')).toHaveValue('draft-key')
  expect(router.state.location.search).not.toHaveProperty('token')
  await userEvent.click(screen.getByRole('button', { name: 'Reset' }))
  await waitFor(() =>
    expect(screen.getByPlaceholderText('Token Name')).toHaveValue('')
  )
})

it.each([false, true])(
  'keeps task actions available while omitting admin fields in personal records on mobile=%s',
  async (mobile) => {
    await renderActivity({ mobile, category: 'task' })
    const record = screen.getByRole('article', { name: 'Activity record' })
    expect(within(record).getByText('story-task')).toBeVisible()
    expect(
      within(record).getByRole('button', { name: 'Artifacts' })
    ).toBeVisible()
    expect(within(record).queryByText('studio-owner')).not.toBeInTheDocument()
    expect(within(record).queryByText('$0.01')).not.toBeInTheDocument()
    await userEvent.click(
      within(record).getByRole('button', { name: 'View details' })
    )
    const dialog = await screen.findByRole('dialog', { name: /Task Details/ })
    expect(within(dialog).queryByText('Admin Only')).not.toBeInTheDocument()
  }
)
