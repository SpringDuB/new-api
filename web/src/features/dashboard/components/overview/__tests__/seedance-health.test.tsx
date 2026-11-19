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
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import {
  act,
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { SeedanceHealthPanel } from '../seedance-health-panel'

let client: QueryClient
const windowEnd = 1794992400

beforeEach(() => {
  useAuthStore
    .getState()
    .auth.setUser({ id: 4, username: 'creator', role: ROLE.USER })
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/perf-metrics/summary') {
      return {
        data: {
          success: true,
          data: {
            window_end: windowEnd,
            window_start: windowEnd - 86400,
            models: [
              {
                model_name: 'seedance-2-mini',
                request_count: 1,
                success_rate: 100,
                avg_latency_ms: 276000,
                avg_tps: 0,
                recent_success_series: [
                  { ts: windowEnd - 3600, success_rate: 100 },
                ],
              },
              {
                model_name: 'gpt-test',
                request_count: 50,
                success_rate: 50,
                avg_latency_ms: 1000,
                avg_tps: 3,
              },
            ],
          },
        },
      }
    }
    if (url === '/api/channel/search') {
      return {
        data: {
          success: true,
          data: {
            total: 2,
            items: [
              {
                id: 1,
                name: 'Studio route',
                status: 1,
                models: 'seedance-2-mini',
                test_model: 'seedance-2-mini',
                response_time: 1200,
                test_time: windowEnd,
              },
              {
                id: 2,
                name: 'Paused route',
                status: 3,
                models: 'seedance-2-mini',
                test_model: 'gpt-test',
                response_time: 90,
                test_time: windowEnd,
              },
            ],
          },
        },
      }
    }
    throw new Error(`Unexpected health request: ${url}`)
  })
})

afterEach(() => {
  cleanup()
  client.clear()
  useAuthStore.getState().auth.reset()
})

async function renderHealth(
  models = ['seedance-2-mini', 'seedance-2-pro', 'gpt-test'],
  options: { loading?: boolean; error?: boolean; onRetry?: () => void } = {}
) {
  const router = createRouter({
    routeTree: createRootRoute({
      component: () => <SeedanceHealthPanel models={models} {...options} />,
    }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  await act(async () =>
    render(
      <QueryClientProvider client={client}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    )
  )
}

describe('Seedance health', () => {
  it('ranks models by observed requests and then success rate, keeping unused models last', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        success: true,
        data: {
          models: [
            { model_name: 'seedance-z', request_count: 20, success_rate: 70 },
            { model_name: 'seedance-a', request_count: 10, success_rate: 50 },
            { model_name: 'seedance-b', request_count: 10, success_rate: 95 },
          ],
        },
      },
    })
    await renderHealth([
      'seedance-unused',
      'seedance-a',
      'seedance-b',
      'seedance-z',
    ])
    await screen.findByRole('region', { name: 'seedance-z' })
    expect(
      screen
        .getAllByRole('heading', { level: 3 })
        .map((heading) => heading.textContent)
    ).toEqual(['seedance-z', 'seedance-b', 'seedance-a', 'seedance-unused'])
  })
  it('shows only accessible Seedance models and separates a single observed request from unused models', async () => {
    await renderHealth()
    const observed = await screen.findByRole('region', {
      name: 'seedance-2-mini',
    })
    expect(within(observed).getByText('100.00%')).toBeVisible()
    expect(within(observed).getByText('276.00s')).toBeVisible()
    expect(within(observed).getByText('1')).toBeVisible()
    const unused = screen.getByRole('region', { name: 'seedance-2-pro' })
    expect(within(unused).getByText('No recent requests')).toBeVisible()
    expect(within(unused).queryByText('100.00%')).not.toBeInTheDocument()
    expect(screen.queryByText('gpt-test')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Next page' })
    ).not.toBeInTheDocument()
    expect(api.get).not.toHaveBeenCalledWith(
      '/api/channel/search',
      expect.anything()
    )
  })

  it('pages the ranked models six at a time and disables each boundary', async () => {
    const user = userEvent.setup()
    await renderHealth([
      'seedance-2-mini',
      'seedance-2-pro',
      'seedance-alpha',
      'seedance-beta',
      'seedance-delta',
      'seedance-gamma',
      'seedance-zeta',
    ])
    await screen.findByText('Page 1 of 2')
    const previous = screen.getByRole('button', { name: 'Previous page' })
    const next = screen.getByRole('button', { name: 'Next page' })
    expect(previous).toBeDisabled()
    expect(screen.getAllByRole('heading', { level: 3 })).toHaveLength(6)
    expect(screen.queryByText('seedance-zeta')).not.toBeInTheDocument()

    await user.click(next)
    expect(screen.getByText('Page 2 of 2')).toBeVisible()
    expect(screen.getByRole('heading', { level: 3 })).toHaveTextContent(
      'seedance-zeta'
    )
    expect(next).toBeDisabled()

    await user.click(previous)
    expect(
      screen.getByRole('heading', { name: 'seedance-2-mini' })
    ).toBeVisible()
    expect(screen.queryByText('seedance-zeta')).not.toBeInTheDocument()
  })

  it('leaves inactive hours unmeasured instead of making a continuous green history', async () => {
    await renderHealth()
    const observed = await screen.findByRole('region', {
      name: 'seedance-2-mini',
    })
    const activity = within(observed).getByRole('img', {
      name: '1 of 24 hours have recorded requests',
    })
    expect(activity.querySelectorAll('[data-observed=true]')).toHaveLength(1)
    expect(activity.querySelectorAll('[data-observed=false]')).toHaveLength(23)
  })

  it('allows administrators to inspect channel states and labels unrelated channel tests as unavailable', async () => {
    useAuthStore
      .getState()
      .auth.setUser({ id: 4, username: 'operator', role: ROLE.ADMIN })
    await renderHealth()
    await userEvent
      .setup()
      .click(screen.getByRole('tab', { name: 'Channel status' }))
    const active = await screen.findByRole('region', { name: 'Studio route' })
    expect(within(active).getByText('Enabled')).toBeVisible()
    expect(within(active).getByText('1.20s')).toBeVisible()
    const paused = screen.getByRole('region', { name: 'Paused route' })
    expect(within(paused).getByText('Auto Disabled')).toBeVisible()
    expect(within(paused).getByText('No Seedance test')).toBeVisible()
    expect(within(paused).queryByText('90ms')).not.toBeInTheDocument()
  })

  it('removes channel details after administrator permissions are revoked', async () => {
    useAuthStore
      .getState()
      .auth.setUser({ id: 4, username: 'operator', role: ROLE.ADMIN })
    await renderHealth()
    await userEvent
      .setup()
      .click(screen.getByRole('tab', { name: 'Channel status' }))
    await screen.findByText('Studio route')
    act(() =>
      useAuthStore
        .getState()
        .auth.setUser({ id: 4, username: 'operator', role: ROLE.USER })
    )
    expect(
      screen.queryByRole('tab', { name: 'Channel status' })
    ).not.toBeInTheDocument()
    expect(screen.queryByText('Studio route')).not.toBeInTheDocument()
    expect(
      screen.getByRole('tab', { name: 'Model performance' })
    ).toHaveAttribute('aria-selected', 'true')
  })

  it('loads six channels per request and shows the final channel on the second page', async () => {
    useAuthStore
      .getState()
      .auth.setUser({ id: 4, username: 'operator', role: ROLE.ADMIN })
    const channelNames = [
      'Studio',
      'Cinema',
      'Editorial',
      'Draft',
      'Video',
      'Motion',
      'Backup',
    ]
    vi.mocked(api.get).mockImplementation(async (_url, config) => {
      const params = config?.params as { p?: number } | undefined
      const start = params?.p === 2 ? 6 : 0
      return {
        data: {
          success: true,
          data: {
            total: channelNames.length,
            items: channelNames.slice(start, start + 6).map((name, index) => ({
              id: start + index + 1,
              name,
              models: 'seedance-2-mini',
              status: 1,
            })),
          },
        },
      }
    })
    const user = userEvent.setup()
    await renderHealth([])
    await user.click(screen.getByRole('tab', { name: 'Channel status' }))
    await screen.findByText('Page 1 of 2')
    expect(screen.getAllByRole('heading', { level: 3 })).toHaveLength(6)
    await user.click(screen.getByRole('button', { name: 'Next page' }))

    expect(await screen.findByRole('region', { name: 'Backup' })).toBeVisible()
    expect(screen.getByText('Page 2 of 2')).toBeVisible()
    expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled()
    expect(api.get).toHaveBeenLastCalledWith('/api/channel/search', {
      params: { model: 'seedance', p: 2, page_size: 6 },
    })
  })

  it('shows an empty inventory without fetching performance data when no Seedance models are available', async () => {
    await renderHealth(['gpt-test'])
    expect(
      await screen.findByText('No Seedance models available')
    ).toBeVisible()
    expect(api.get).not.toHaveBeenCalled()
  })

  it('shows a request failure and retries with the original inventory', async () => {
    vi.mocked(api.get).mockRejectedValueOnce(new Error('network offline'))
    await renderHealth()
    expect(
      await screen.findByText('Unable to load Seedance activity')
    ).toBeVisible()
    await userEvent.setup().click(screen.getByRole('button', { name: 'Retry' }))
    await waitFor(() =>
      expect(
        screen.getByRole('region', { name: 'seedance-2-mini' })
      ).toBeVisible()
    )
  })

  it('shows the inventory error and retries the inventory before issuing performance requests', async () => {
    const retry = vi.fn()
    await renderHealth(['seedance-2-mini'], { error: true, onRetry: retry })
    expect(
      await screen.findByText('Unable to load Seedance activity')
    ).toBeVisible()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    await userEvent.setup().click(screen.getByRole('button', { name: 'Retry' }))
    expect(retry).toHaveBeenCalledOnce()
    expect(api.get).not.toHaveBeenCalled()
  })

  it('keeps the inventory loading state distinct from an empty Seedance catalog', async () => {
    await renderHealth([], { loading: true })
    expect(
      screen.getByRole('status', { name: 'Loading Seedance activity' })
    ).toBeVisible()
    expect(
      screen.queryByText('No Seedance models available')
    ).not.toBeInTheDocument()
    expect(api.get).not.toHaveBeenCalled()
  })
})
