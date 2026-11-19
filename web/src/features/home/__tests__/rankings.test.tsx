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

import type { RankingsSnapshot } from '@/features/rankings/types'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { HomeRankings } from '../components/sections/rankings'
import { Home } from '../index'

let client: QueryClient
let snapshot: RankingsSnapshot
let homeContent: string

beforeEach(() => {
  window.localStorage.clear()
  useAuthStore.setState(useAuthStore.getInitialState(), true)
  useSystemConfigStore.setState(useSystemConfigStore.getInitialState(), true)
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], {})
  client.setQueryData(['notice'], { success: true, data: '' })
  snapshot = {
    models: [
      {
        rank: 1,
        model_name: 'claude-sonnet-4-6',
        vendor: 'Anthropic',
        category: 'all',
        total_tokens: 1500000,
        share: 0.6,
        growth_pct: 10,
      },
      {
        rank: 2,
        model_name: 'gpt-5.4',
        vendor: 'OpenAI',
        category: 'all',
        total_tokens: 1000000,
        share: 0.4,
        growth_pct: 0,
      },
    ],
    vendors: [
      {
        rank: 1,
        vendor: 'Anthropic',
        total_tokens: 1500000,
        share: 0.6,
        growth_pct: 10,
        models_count: 1,
        top_model: 'claude-sonnet-4-6',
      },
    ],
    top_movers: [],
    top_droppers: [],
    models_history: { points: [], models: [], buckets: 0 },
    vendor_share_history: { points: [], vendors: [], buckets: 0 },
  }
  homeContent = ''
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/rankings') {
      return { data: { success: true, data: snapshot } }
    }
    if (url === '/api/home_page_content') {
      return { data: { success: true, data: homeContent } }
    }
    throw new Error(`Unexpected home request: ${url}`)
  })
})

afterEach(() => {
  cleanup()
  client.clear()
  useAuthStore.setState(useAuthStore.getInitialState(), true)
  useSystemConfigStore.setState(useSystemConfigStore.getInitialState(), true)
  window.localStorage.clear()
})

async function renderHome(component = HomeRankings) {
  const router = createRouter({
    routeTree: createRootRoute({ component }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

describe('home rankings access and data', () => {
  it('shows real top models and vendor token shares when public rankings are enabled', async () => {
    await renderHome()

    const models = await screen.findByRole('list', { name: 'Top Models' })
    expect(within(models).getAllByRole('listitem')).toHaveLength(2)
    expect(
      within(models).getByRole('link', { name: 'claude-sonnet-4-6' })
    ).toHaveAttribute('href', '/pricing/claude-sonnet-4-6')
    expect(within(models).getByText('1.5M')).toBeVisible()
    const share = screen.getByRole('progressbar', { name: 'Anthropic' })
    expect(share).toHaveAttribute('aria-valuenow', '60')
    expect(screen.getByRole('button', { name: 'Week' })).toHaveAttribute(
      'aria-pressed',
      'true'
    )
    expect(api.get).toHaveBeenCalledWith('/api/rankings', {
      params: { period: 'week' },
    })
  })

  it('hides the section and makes no ranking request when rankings are disabled', async () => {
    client.setQueryData(['status'], {
      HeaderNavModules: JSON.stringify({ rankings: { enabled: false } }),
    })

    await renderHome()

    expect(
      screen.queryByRole('region', { name: 'Rankings' })
    ).not.toBeInTheDocument()
    expect(api.get).not.toHaveBeenCalled()
  })

  it('requires sign-in without exposing cached rankings or fetching data for guests', async () => {
    client.setQueryData(['status'], {
      HeaderNavModules: { rankings: { enabled: true, requireAuth: true } },
    })
    client.setQueryData(['rankings', 'week'], { success: true, data: snapshot })

    await renderHome()

    expect(await screen.findByText('Sign in to view rankings')).toBeVisible()
    expect(screen.getByRole('link', { name: 'Sign in' })).toHaveAttribute(
      'href',
      '/sign-in?redirect=%2F'
    )
    expect(
      screen.queryByRole('list', { name: 'Top Models' })
    ).not.toBeInTheDocument()
    expect(api.get).not.toHaveBeenCalled()
  })

  it('loads restricted rankings after the guest signs in', async () => {
    client.setQueryData(['status'], {
      HeaderNavModules: { rankings: { enabled: true, requireAuth: true } },
    })
    await renderHome()

    await act(() =>
      useAuthStore
        .getState()
        .auth.setUser({ id: 1, username: 'creator', role: 1 })
    )

    expect(
      await screen.findByRole('link', { name: 'claude-sonnet-4-6' })
    ).toBeVisible()
    expect(
      screen.queryByText('Sign in to view rankings')
    ).not.toBeInTheDocument()
  })

  it('waits for current access configuration even when a cached public snapshot exists', async () => {
    client.removeQueries({ queryKey: ['status'] })
    window.localStorage.setItem(
      'status',
      JSON.stringify({
        HeaderNavModules: { rankings: { enabled: true, requireAuth: false } },
      })
    )
    let resolveStatus: (value: unknown) => void = () => undefined
    vi.mocked(api.get).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveStatus = resolve
        })
    )
    await renderHome()
    expect(api.get).toHaveBeenCalledWith('/api/status')
    expect(api.get).not.toHaveBeenCalledWith('/api/rankings', expect.anything())

    await act(() =>
      resolveStatus({
        data: {
          data: {
            HeaderNavModules: {
              rankings: { enabled: true, requireAuth: true },
            },
          },
        },
      })
    )

    expect(await screen.findByText('Sign in to view rankings')).toBeVisible()
    expect(api.get).not.toHaveBeenCalledWith('/api/rankings', expect.anything())
  })

  it('changes the selected period and replaces the results with that period', async () => {
    const user = userEvent.setup()
    await renderHome()
    await screen.findByRole('link', { name: 'claude-sonnet-4-6' })
    snapshot = {
      ...snapshot,
      models: [{ ...snapshot.models[0], model_name: 'gemini-3-pro' }],
    }

    await user.click(screen.getByRole('button', { name: 'Month' }))

    expect(
      await screen.findByRole('link', { name: 'gemini-3-pro' })
    ).toBeVisible()
    expect(
      screen.queryByRole('link', { name: 'claude-sonnet-4-6' })
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Month' })).toHaveAttribute(
      'aria-pressed',
      'true'
    )
    expect(screen.getByRole('button', { name: 'Week' })).toHaveAttribute(
      'aria-pressed',
      'false'
    )
    expect(api.get).toHaveBeenLastCalledWith('/api/rankings', {
      params: { period: 'month' },
    })
  })

  it('lets keyboard users move between ranking periods', async () => {
    const user = userEvent.setup()
    await renderHome()
    await screen.findByRole('list', { name: 'Top Models' })

    await user.tab()
    expect(screen.getByRole('button', { name: 'Today' })).toHaveFocus()
    await user.keyboard('{ArrowRight}{ArrowRight} ')

    expect(screen.getByRole('button', { name: 'Month' })).toHaveAttribute(
      'aria-pressed',
      'true'
    )
    await waitFor(() =>
      expect(api.get).toHaveBeenLastCalledWith('/api/rankings', {
        params: { period: 'month' },
      })
    )
  })

  it('shows a loading state until the ranking response resolves', async () => {
    let resolveRankings: (value: unknown) => void = () => undefined
    vi.mocked(api.get).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRankings = resolve
        })
    )
    await renderHome()
    expect(await screen.findByRole('status')).toHaveTextContent(
      'Loading rankings...'
    )

    await act(() =>
      resolveRankings({ data: { success: true, data: snapshot } })
    )

    expect(
      await screen.findByRole('list', { name: 'Top Models' })
    ).toBeVisible()
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('shows an honest empty state when no traffic has been recorded', async () => {
    snapshot = { ...snapshot, models: [], vendors: [] }
    await renderHome()

    expect(await screen.findByText('No ranking data yet')).toBeVisible()
    expect(screen.queryByRole('list')).not.toBeInTheDocument()
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
  })

  it('keeps available models visible when vendor data is empty', async () => {
    snapshot = { ...snapshot, vendors: [] }
    await renderHome()

    expect(
      await screen.findByRole('link', { name: 'claude-sonnet-4-6' })
    ).toBeVisible()
    expect(screen.getByText('No vendor data available')).toBeVisible()
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
  })

  it('shows a failed response as an error and lets the user retry', async () => {
    const user = userEvent.setup()
    vi.mocked(api.get).mockResolvedValueOnce({
      data: { success: false, message: 'Rankings unavailable' },
    })
    await renderHome()
    expect(await screen.findByText('Unable to load rankings')).toBeVisible()
    expect(screen.queryByText('No ranking data yet')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Retry' }))

    expect(
      await screen.findByRole('link', { name: 'claude-sonnet-4-6' })
    ).toBeVisible()
  })
})

describe('default home rankings placement', () => {
  it('places the inspiration gallery after rankings and keeps the courtyard in the hero', async () => {
    await renderHome(Home)
    const rankings = await screen.findByRole('region', { name: 'Rankings' })
    const hero = screen.getByRole('main')
    const gallery = screen.getByRole('region', { name: 'Inspiration gallery' })
    const footer = screen.getByRole('contentinfo')

    expect(
      hero.compareDocumentPosition(rankings) & Node.DOCUMENT_POSITION_FOLLOWING
    ).not.toBe(0)
    expect(
      rankings.compareDocumentPosition(gallery) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).not.toBe(0)
    expect(
      gallery.compareDocumentPosition(footer) & Node.DOCUMENT_POSITION_FOLLOWING
    ).not.toBe(0)
    expect(
      within(hero).getByRole('img', {
        name: 'Sunlit courtyard with a blue door',
      })
    ).toHaveAttribute('fetchpriority', 'high')
    const galleryImages = within(gallery).getAllByRole('img')
    expect(galleryImages).toHaveLength(6)
    for (const image of galleryImages) {
      expect(image).toHaveAttribute('loading', 'lazy')
    }
  })

  it.each([
    ['iframe', 'https://example.com/home'],
    ['html', '<h1>Custom publication</h1>'],
    ['markdown', '# Custom publication'],
  ])(
    'preserves the custom %s homepage without fetching rankings',
    async (_mode, content) => {
      homeContent = content
      await renderHome(Home)

      await waitFor(() =>
        expect(screen.queryByText('Loading...')).not.toBeInTheDocument()
      )
      expect(
        screen.queryByRole('region', { name: 'Rankings' })
      ).not.toBeInTheDocument()
      expect(
        screen.queryByRole('region', { name: 'Inspiration gallery' })
      ).not.toBeInTheDocument()
      expect(api.get).not.toHaveBeenCalledWith(
        '/api/rankings',
        expect.anything()
      )
      if (content.startsWith('https://')) {
        expect(screen.getByTitle('Custom Home Page')).toHaveAttribute(
          'src',
          content
        )
      } else if (content.startsWith('#')) {
        expect(
          await screen.findByRole('heading', { name: 'Custom publication' })
        ).toBeVisible()
      } else {
        const heading = document
          .querySelector('.custom-home-content')
          ?.shadowRoot?.querySelector('h1')
        expect(heading).toHaveTextContent('Custom publication')
      }
    }
  )
})
