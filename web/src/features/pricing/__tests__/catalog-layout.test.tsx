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
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { Pricing } from '..'

vi.mock('@visactor/react-vchart', () => ({ VChart: () => null }))

vi.mock('@/components/layout', () => ({
  PublicLayout: (props: { children: ReactNode }) => props.children,
}))

vi.mock('../hooks/use-pricing-data', () => {
  const models = [
    {
      id: 1,
      model_name: 'seedance-2-5-studio',
      description: 'Create video from a prompt or reference image.',
      square_category: 'video',
      quota_type: 1,
      model_price: 0.4,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: ['default', 'premium'],
      vendor_name: 'Studio models',
    },
    {
      id: 2,
      model_name: 'seedance-2-0-fast',
      square_category: 'video',
      quota_type: 1,
      model_price: 0.2,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: ['default'],
    },
    {
      id: 3,
      model_name: 'image-2',
      square_category: 'image',
      quota_type: 1,
      model_price: 0.1,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: ['default'],
    },
  ]
  return {
    usePricingData: () => ({
      models,
      catalogModels: models,
      vendors: [{ id: 1, name: 'Studio models' }],
      groupRatio: { default: 1, premium: 2 },
      usableGroup: { default: 1, premium: 2 },
      endpointMap: {},
      autoGroups: [],
      isLoading: false,
      priceRate: 1,
      usdExchangeRate: 1,
    }),
  }
})

let queryClient: QueryClient

beforeEach(() => {
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { models: [] } },
  })
  useAuthStore.getState().auth.setUser(null)
})

afterEach(() => {
  queryClient.clear()
  useAuthStore.getState().auth.setUser(null)
})

function renderCatalog(path = '/pricing/') {
  const root = createRootRoute()
  const route = createRoute({
    getParentRoute: () => root,
    path: '/pricing/',
    component: Pricing,
  })
  const router = createRouter({
    routeTree: root.addChildren([route]),
    history: createMemoryHistory({ initialEntries: [path] }),
  })
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

describe('model library layout', () => {
  it('presents the actual catalog as a directory with category navigation and no permanent filter sidebar', async () => {
    renderCatalog()
    expect(
      await screen.findByRole('heading', { name: 'Model library', level: 1 })
    ).toBeVisible()
    expect(screen.getByRole('textbox', { name: 'Search models' })).toBeVisible()
    expect(
      screen.getByRole('navigation', { name: 'Model categories' })
    ).toBeVisible()
    expect(screen.queryByRole('complementary')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Manage display' })
    ).not.toBeInTheDocument()
    const directory = screen.getByRole('list', { name: 'Models' })
    expect(within(directory).getAllByRole('listitem')).toHaveLength(3)
    const model = within(directory).getByRole('article', {
      name: 'seedance-2-5-studio',
    })
    expect(
      within(model).getByText('Create video from a prompt or reference image.')
    ).toBeVisible()
    expect(
      within(model).getByRole('group', { name: 'Pricing' })
    ).toHaveTextContent('/ request')
    expect(within(model).getByRole('button', { name: 'Details' })).toBeVisible()
    expect(
      within(model).queryByLabelText(
        'Performance metrics for the last 24 hours'
      )
    ).not.toBeInTheDocument()
  })

  it('keeps the selected category and filter state when closing and reopening the right panel, and resets within that category', async () => {
    renderCatalog('/pricing/?category=video')
    const user = userEvent.setup()
    const trigger = await screen.findByRole('button', {
      name: 'Filters and display',
    })
    await user.click(trigger)
    const panel = await screen.findByRole('dialog', {
      name: 'Filters and display',
    })
    await user.click(within(panel).getByRole('combobox', { name: 'Groups' }))
    await user.click(screen.getByRole('option', { name: /premium/ }))
    await user.keyboard('{Escape}')
    const directory = await screen.findByRole('list', { name: 'Models' })
    expect(within(directory).getAllByRole('article')).toHaveLength(1)
    expect(
      within(directory).getByRole('article', { name: 'seedance-2-5-studio' })
    ).toBeVisible()
    const activeTrigger = screen.getByRole('button', {
      name: /Filters and display\s*\(1\)/,
    })
    expect(activeTrigger).toHaveFocus()
    await user.click(activeTrigger)
    const reopened = await screen.findByRole('dialog', {
      name: 'Filters and display',
    })
    expect(reopened).toHaveAttribute('data-side', 'right')
    expect(
      within(reopened).getByRole('combobox', { name: 'Groups' })
    ).toHaveValue('premium · x2')
    await user.click(within(reopened).getByRole('button', { name: 'Reset' }))
    await user.keyboard('{Escape}')
    expect(
      screen.getByRole('button', { name: /^Videos\s*2$/ })
    ).toHaveAttribute('aria-pressed', 'true')
    expect(
      within(screen.getByRole('list', { name: 'Models' })).getAllByRole(
        'article'
      )
    ).toHaveLength(2)
  })

  it('lets keyboard category navigation and search recover from an empty directory', async () => {
    renderCatalog()
    const user = userEvent.setup()
    const images = await screen.findByRole('button', { name: /^Images\s*1$/ })
    images.focus()
    await user.keyboard('{Enter}')
    expect(images).toHaveAttribute('aria-pressed', 'true')
    expect(
      within(screen.getByRole('list', { name: 'Models' })).getAllByRole(
        'article'
      )
    ).toHaveLength(1)
    await user.type(
      screen.getByRole('textbox', { name: 'Search models' }),
      'unavailable-model'
    )
    expect(
      await screen.findByRole('heading', { name: 'No models found' })
    ).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Clear search' }))
    await waitFor(() =>
      expect(screen.getByRole('article', { name: 'image-2' })).toBeVisible()
    )
    expect(images).toHaveAttribute('aria-pressed', 'true')
  })

  it('loads display choices from the URL and preserves the active group when changing layout across tabs', async () => {
    renderCatalog(
      '/pricing/?category=video&group=premium&tokenUnit=K&rechargePrice=true'
    )
    const user = userEvent.setup()
    await user.click(
      await screen.findByRole('button', { name: /Filters and display/ })
    )
    expect(screen.getByRole('combobox', { name: 'Groups' })).toHaveValue(
      'premium · x2'
    )
    await user.click(screen.getByRole('tab', { name: 'Display' }))
    expect(screen.getByRole('radio', { name: 'Recharge' })).toBeChecked()
    expect(screen.getByRole('radio', { name: '/1K' })).toBeChecked()
    expect(screen.getByRole('radio', { name: 'List view' })).toBeChecked()
    await user.click(screen.getByRole('tab', { name: 'Filter' }))
    expect(screen.getByRole('combobox', { name: 'Groups' })).toHaveValue(
      'premium · x2'
    )
    await user.click(screen.getByRole('tab', { name: 'Display' }))
    await user.click(screen.getByRole('radio', { name: 'Table view' }))
    await user.keyboard('{Escape}')
    expect(await screen.findByRole('table')).toBeVisible()
    await user.click(
      screen.getByRole('button', { name: /Filters and display/ })
    )
    await user.click(screen.getByRole('tab', { name: 'Display' }))
    expect(screen.getByRole('radio', { name: 'Table view' })).toBeChecked()
    expect(screen.getByRole('radio', { name: '/1K' })).toBeChecked()
    expect(screen.getByRole('radio', { name: 'Recharge' })).toBeChecked()
    await user.click(screen.getByRole('tab', { name: 'Filter' }))
    expect(screen.getByRole('combobox', { name: 'Groups' })).toHaveValue(
      'premium · x2'
    )
  })

  it('retains display management access for an administrator in the new heading', async () => {
    useAuthStore.getState().auth.setUser({ id: 1, username: 'admin', role: 10 })
    renderCatalog()
    expect(
      await screen.findByRole('button', { name: 'Manage display' })
    ).toBeVisible()
  })
})
