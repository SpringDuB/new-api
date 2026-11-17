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
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'

import type { ModelSquareDisplayResponse } from '../api'
import ModelSquareManager from '../components/model-square-manager'
import type {
  ModelCategory,
  ModelCategoryFilter,
} from '../lib/model-categories'
import type { ModelSquareDisplay } from '../lib/model-square-display'
import type { PricingModel } from '../types'

const models: PricingModel[] = ['alpha', 'beta', 'gamma'].map(
  (model_name, id) => ({
    id,
    model_name,
    quota_type: 1,
    model_price: 1,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    tags: 'catalog-tag',
  })
)

let client: QueryClient
beforeEach(() => {
  client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: { order: [], models: {} } },
  })
})
afterEach(() => {
  client.clear()
})

function renderManager(
  catalog = models,
  initialCategory: ModelCategoryFilter = 'all',
  initialFamily = 'all'
) {
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={client}>
      <ModelSquareManager
        models={catalog}
        initialCategory={initialCategory}
        initialFamily={initialFamily}
        onClose={onClose}
        priceRate={1}
        usdExchangeRate={1}
      />
    </QueryClientProvider>
  )
  return onClose
}

function deferCachedDisplayRefresh() {
  client.setQueryData(['model-square-display'], {
    success: true,
    data: { order: [], models: {} },
  })
  let completeRefresh:
    | ((response: { data: ModelSquareDisplayResponse }) => void)
    | undefined
  vi.mocked(api.get).mockImplementationOnce(
    () =>
      new Promise((resolve) => {
        completeRefresh = resolve
      })
  )
  return (display: ModelSquareDisplay) =>
    completeRefresh?.({ data: { success: true, data: display } })
}

describe('model square manager', () => {
  it.each<{
    category: ModelCategory
    names: string[]
    expected: string[]
  }>([
    {
      category: 'video',
      names: [
        'a-minimax-h3',
        'b-seedance-2-0',
        'c-wan-3-0',
        'd-seedance-2-5',
        'e-minimax-h3',
        'f-seedance-2-5',
      ],
      expected: [
        'd-seedance-2-5',
        'f-seedance-2-5',
        'b-seedance-2-0',
        'c-wan-3-0',
        'a-minimax-h3',
        'e-minimax-h3',
      ],
    },
    {
      category: 'image',
      names: ['a-seedream-5-0', 'b-image-2', 'c-seedream-5-0'],
      expected: ['b-image-2', 'a-seedream-5-0', 'c-seedream-5-0'],
    },
    {
      category: 'chat',
      names: ['a-gpt-4o', 'b-claude-sonnet-4', 'c-gpt-4o-mini'],
      expected: ['b-claude-sonnet-4', 'a-gpt-4o', 'c-gpt-4o-mini'],
    },
  ])(
    'opens interleaved $category recommendations in series order without changing the saved settings',
    async ({ category, names, expected }) => {
      const catalog = names.map((model_name, id) => ({
        ...models[0],
        id,
        model_name,
        square_category: category,
      }))
      vi.mocked(api.get).mockResolvedValue({
        data: { success: true, data: { order: names, models: {} } },
      })
      renderManager(catalog, category)
      await screen.findByRole('button', { name: `Edit ${expected[0]}` })

      const ordering = screen.getByRole('region', { name: 'Model order' })
      expect(ordering).toHaveClass('max-h-[48dvh]', 'overflow-y-auto')
      expect(
        within(ordering)
          .getAllByRole('button', { name: /^Edit / })
          .map((button) => button.getAttribute('aria-label')?.slice(5))
      ).toEqual(expected)
      expect(screen.getByRole('combobox', { name: 'Sort' })).toHaveTextContent(
        'Model series'
      )
      expect(
        screen.getByRole('button', { name: 'Save changes' })
      ).toBeDisabled()
    }
  )

  it.each(['arrow', 'top', 'position', 'keyboard'])(
    'keeps series grouped when reordering with %s and preserves other saved positions',
    async (method) => {
      const names = [
        'a-seedance-2-0',
        'b-seedance-2-5',
        'c-wan-3-0',
        'd-seedance-2-5',
        'e-minimax',
      ]
      const catalog = names.map((model_name, id) => ({
        ...models[0],
        id,
        model_name,
        square_category: 'video' as const,
      }))
      vi.mocked(api.get).mockResolvedValue({
        data: {
          success: true,
          data: {
            order: [names[0], 'hidden', ...names.slice(1)],
            models: { hidden: { note: 'Preserve' } },
          },
        },
      })
      const save = vi
        .spyOn(api, 'put')
        .mockImplementation(async (_url, data) => ({
          data: { success: true, data },
        }))
      renderManager(catalog, 'video')
      const user = userEvent.setup()
      const series = await screen.findByRole('list', {
        name: 'Model order: Seedance 2.5',
      })
      expect(
        within(series).getByRole('button', { name: 'Move b-seedance-2-5 up' })
      ).toBeDisabled()
      expect(
        within(series).getByRole('button', { name: 'Move d-seedance-2-5 down' })
      ).toBeDisabled()

      if (method === 'arrow') {
        await user.click(
          within(series).getByRole('button', { name: 'Move d-seedance-2-5 up' })
        )
      } else if (method === 'keyboard') {
        within(series)
          .getByRole('button', {
            name: 'Drag d-seedance-2-5 to reorder',
          })
          .focus()
        await user.keyboard('{ArrowUp}')
      } else {
        await user.click(
          within(series).getByRole('button', {
            name: 'Edit d-seedance-2-5',
          })
        )
        if (method === 'top') {
          await user.click(screen.getByRole('button', { name: 'Move to top' }))
        } else {
          const position = screen.getByLabelText('Move to position')
          expect(position).toHaveAttribute('max', '2')
          await user.type(position, '1')
          await user.click(screen.getByRole('button', { name: /^Move$/ }))
        }
      }
      expect(
        screen
          .getAllByRole('button', { name: /^Edit / })
          .map((button) => button.getAttribute('aria-label')?.slice(5))
      ).toEqual([
        'd-seedance-2-5',
        'b-seedance-2-5',
        'a-seedance-2-0',
        'c-wan-3-0',
        'e-minimax',
      ])
      await user.click(screen.getByRole('button', { name: 'Save changes' }))
      await waitFor(() => expect(save).toHaveBeenCalled())
      expect(save.mock.calls[0][1]).toEqual({
        order: [
          'a-seedance-2-0',
          'hidden',
          'd-seedance-2-5',
          'c-wan-3-0',
          'b-seedance-2-5',
          'e-minimax',
        ],
        models: { hidden: { note: 'Preserve' } },
      })
    }
  )

  it('allows switching to recommendation order for moves across series without saving the viewing preference', async () => {
    const catalog = ['a-seedance-2-0', 'b-minimax', 'c-seedance-2-5'].map(
      (model_name, id) => ({
        ...models[0],
        id,
        model_name,
        square_category: 'video' as const,
      })
    )
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    renderManager(catalog, 'video')
    const user = userEvent.setup()
    const sort = await screen.findByRole('combobox', { name: 'Sort' })
    await user.click(sort)
    await user.click(screen.getByRole('option', { name: 'Recommended order' }))
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled()
    expect(
      screen
        .getAllByRole('button', { name: /^Edit / })
        .map((button) => button.getAttribute('aria-label')?.slice(5))
    ).toEqual(['a-seedance-2-0', 'b-minimax', 'c-seedance-2-5'])

    await user.click(sort)
    await user.click(screen.getByRole('option', { name: 'Model series' }))
    expect(
      screen
        .getAllByRole('button', { name: /^Edit / })
        .map((button) => button.getAttribute('aria-label')?.slice(5))
    ).toEqual(['c-seedance-2-5', 'a-seedance-2-0', 'b-minimax'])
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled()

    await user.click(sort)
    await user.click(screen.getByRole('option', { name: 'Recommended order' }))
    await user.click(screen.getByRole('button', { name: 'Edit b-minimax' }))
    await user.click(screen.getByRole('button', { name: 'Move to top' }))
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(save).toHaveBeenCalled())
    expect((save.mock.calls[0][1] as ModelSquareDisplay).order).toEqual([
      'b-minimax',
      'a-seedance-2-0',
      'c-seedance-2-5',
    ])
  })

  it('keeps editing the same model after assigning it to another series and disables moves outside the scope', async () => {
    const catalog = [
      {
        ...models[0],
        model_name: 'seedance-2-5',
        square_category: 'video' as const,
      },
    ]
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    renderManager(catalog, 'video', 'series:Seedance 2.5')
    const user = userEvent.setup()
    const family = await screen.findByRole('combobox', { name: 'Model series' })
    await user.clear(family)
    await user.type(family, 'Editorial collection')
    expect(family).toHaveValue('Editorial collection')
    expect(
      screen.getByRole('combobox', { name: 'Filter by model series' })
    ).toHaveValue('Seedance 2.5 · 0')
    expect(screen.getByLabelText('Move to position')).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Move to top' })).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(save).toHaveBeenCalled())
    expect((save.mock.calls[0][1] as ModelSquareDisplay).models).toEqual({
      'seedance-2-5': { family: 'Editorial collection' },
    })
  })

  it.each(['arrow', 'top', 'position'])(
    'reorders only the current category and series with %s, preserving other series and hidden slots',
    async (method) => {
      const names = [
        'a-seedance-2-5',
        'b-image-2',
        'c-seedance-2-0',
        'd-seedance-2-5',
      ]
      const catalog = names.map((model_name, index) => ({
        ...models[0],
        id: index,
        model_name,
        square_category: index === 1 ? ('image' as const) : ('video' as const),
      }))
      vi.mocked(api.get).mockResolvedValue({
        data: {
          success: true,
          data: {
            order: [names[0], 'hidden', ...names.slice(1)],
            models: { hidden: { note: 'Preserve' } },
          },
        },
      })
      const save = vi
        .spyOn(api, 'put')
        .mockImplementation(async (_url, data) => ({
          data: { success: true, data },
        }))
      renderManager(catalog, 'video', 'series:Seedance 2.5')
      const user = userEvent.setup()
      const edit = await screen.findByRole('button', {
        name: 'Edit d-seedance-2-5',
      })
      expect(
        screen.queryByRole('button', { name: 'Edit b-image-2' })
      ).not.toBeInTheDocument()
      expect(
        screen.queryByRole('button', { name: 'Edit c-seedance-2-0' })
      ).not.toBeInTheDocument()
      if (method === 'arrow') {
        await user.click(
          screen.getByRole('button', { name: 'Move d-seedance-2-5 up' })
        )
      } else {
        await user.click(edit)
        if (method === 'top') {
          await user.click(screen.getByRole('button', { name: 'Move to top' }))
        } else {
          const position = screen.getByLabelText('Move to position')
          expect(position).toHaveAttribute('max', '2')
          await user.type(position, '1')
          await user.click(screen.getByRole('button', { name: /^Move$/ }))
        }
      }
      await user.click(screen.getByRole('button', { name: 'Save changes' }))
      await waitFor(() => expect(save).toHaveBeenCalled())
      expect(save.mock.calls[0][1]).toEqual({
        order: [
          'd-seedance-2-5',
          'hidden',
          'b-image-2',
          'c-seedance-2-0',
          'a-seedance-2-5',
        ],
        models: { hidden: { note: 'Preserve' } },
      })
    }
  )

  it('changes the ordering scope without dirtying the form and shows an empty scope safely', async () => {
    renderManager()
    const user = userEvent.setup()
    await user.click(
      await screen.findByRole('combobox', { name: 'Ordering scope' })
    )
    await user.click(screen.getByRole('option', { name: /^Videos · 0$/ }))
    expect(screen.getByText('No models found')).toBeVisible()
    expect(
      screen.queryByLabelText('Recommendation badge')
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled()
    await user.click(screen.getByRole('combobox', { name: 'Ordering scope' }))
    await user.click(screen.getByRole('option', { name: /^All · 3$/ }))
    expect(screen.getByLabelText('Recommendation badge')).toBeVisible()
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled()
  })

  it('saves a custom series then restores automatic grouping when its value is cleared', async () => {
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    renderManager()
    const user = userEvent.setup()
    const family = await screen.findByRole('combobox', { name: 'Model series' })
    await user.clear(family)
    await user.type(family, '  Editorial  ')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(save).toHaveBeenCalled())
    expect(
      (save.mock.calls[0][1] as ModelSquareDisplay).models.alpha.family
    ).toBe('Editorial')
    await user.clear(family)
    await user.type(screen.getByLabelText('Card notice'), 'Keep')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(save).toHaveBeenCalledTimes(2))
    expect((save.mock.calls[1][1] as ModelSquareDisplay).models.alpha).toEqual({
      note: 'Keep',
    })
  })

  it('assigns a custom series in bulk while retaining badges and hidden settings', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        success: true,
        data: {
          order: [],
          models: { alpha: { badge: 'Keep' }, hidden: { family: 'Private' } },
        },
      },
    })
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    renderManager()
    const user = userEvent.setup()
    await user.click(
      await screen.findByRole('button', { name: 'Batch classification' })
    )
    const selection = await screen.findByRole('combobox', {
      name: 'Search and select models...',
    })
    await user.click(selection)
    await user.click(await screen.findByRole('option', { name: 'alpha' }))
    await user.click(selection)
    await user.click(await screen.findByRole('option', { name: 'beta' }))
    await user.keyboard('{Escape}')
    const family = screen.getByRole('combobox', { name: 'Assign series' })
    await user.clear(family)
    await user.type(family, 'Custom collection')
    await user.click(
      screen.getByRole('button', { name: 'Apply series to selected' })
    )
    expect(screen.getByRole('combobox', { name: 'Model series' })).toHaveValue(
      'Custom collection'
    )
    expect(save).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(save).toHaveBeenCalled())
    expect((save.mock.calls[0][1] as ModelSquareDisplay).models).toEqual({
      alpha: { badge: 'Keep', family: 'Custom collection' },
      beta: { family: 'Custom collection' },
      hidden: { family: 'Private' },
    })
  })

  it('saves a manual category and returns to automatic classification without losing the badge', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        success: true,
        data: {
          order: [],
          models: {
            alpha: { badge: 'Keep', category: 'video', family: 'Editorial' },
          },
        },
      },
    })
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    const onClose = renderManager()
    const user = userEvent.setup()
    const category = await screen.findByRole('combobox', {
      name: 'Model category',
    })
    await user.click(category)
    await user.click(screen.getByRole('option', { name: 'Images' }))
    await user.click(
      screen.getByRole('button', { name: 'Clear card decoration' })
    )
    expect(category).toHaveTextContent('Images')
    expect(screen.getByRole('combobox', { name: 'Model series' })).toHaveValue(
      'Editorial'
    )
    expect(screen.getByLabelText('Recommendation badge')).toHaveValue('')
    await user.type(screen.getByLabelText('Recommendation badge'), 'Keep')
    await user.click(category)
    await user.click(
      screen.getByRole('option', {
        name: 'Automatic classification',
      })
    )
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(onClose).toHaveBeenCalledOnce())
    expect(save).toHaveBeenCalledWith('/api/models/display', {
      order: [],
      models: { alpha: { badge: 'Keep', family: 'Editorial' } },
    })
  })

  it('applies a category to selected models as a draft and preserves unselected hidden settings', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        success: true,
        data: {
          order: [],
          models: {
            hidden: { category: 'video', note: 'Preserve' },
            alpha: { badge: 'Keep' },
          },
        },
      },
    })
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    const onClose = renderManager()
    const user = userEvent.setup()
    await user.click(
      await screen.findByRole('button', { name: 'Batch classification' })
    )
    expect(
      await screen.findByRole('button', { name: 'Apply to selected' })
    ).toBeDisabled()
    const models = screen.getByRole('combobox', {
      name: 'Search and select models...',
    })
    await user.click(models)
    await user.click(await screen.findByRole('option', { name: /^alpha/ }))
    await user.click(models)
    await user.click(await screen.findByRole('option', { name: /^beta/ }))
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('combobox', { name: 'Assign category' }))
    await user.click(screen.getByRole('option', { name: 'Images' }))
    await user.click(screen.getByRole('button', { name: 'Apply to selected' }))
    expect(
      screen.getByRole('combobox', { name: 'Model category' })
    ).toHaveTextContent('Images')
    expect(save).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(onClose).toHaveBeenCalledOnce())
    expect(save).toHaveBeenCalledWith('/api/models/display', {
      order: [],
      models: {
        hidden: { category: 'video', note: 'Preserve' },
        alpha: { badge: 'Keep', category: 'image' },
        beta: { category: 'image' },
      },
    })
  })

  it('previews badge, notice, border and additional tags before saving them together', async () => {
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    const onClose = renderManager()
    const user = userEvent.setup()
    await user.type(
      await screen.findByLabelText('Recommendation badge'),
      'Site choice'
    )
    await user.type(screen.getByLabelText('Card notice'), 'Available all day')
    await user.type(
      screen.getByLabelText('Additional tags'),
      'High quality{Enter}'
    )
    await user.click(
      screen.getByRole('switch', { name: 'Highlight card border' })
    )
    await user.click(
      screen.getByRole('button', { name: 'Recommendation sticker' })
    )
    expect(screen.getByText('Site choice')).toBeVisible()
    expect(screen.getByText('Available all day')).toBeVisible()
    expect(screen.getByRole('group', { name: 'Tags' })).toHaveTextContent(
      'catalog-tag, High quality'
    )
    expect(save).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(onClose).toHaveBeenCalled())
    expect(save).toHaveBeenCalledWith('/api/models/display', {
      order: [],
      models: {
        alpha: {
          badge: 'Site choice',
          note: 'Available all day',
          tags: ['High quality'],
          highlight: true,
          badge_style: 'burst',
        },
      },
    })
  })

  it('keeps an edited badge when cached settings refresh and saves it with the latest untouched order', async () => {
    const refresh = deferCachedDisplayRefresh()
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    const onClose = renderManager()
    const user = userEvent.setup()
    const badge = await screen.findByLabelText('Recommendation badge')
    await user.type(badge, 'Local draft')

    await act(async () => {
      refresh({
        order: ['gamma', 'beta', 'alpha'],
        models: { alpha: { badge: 'Server badge' } },
      })
    })
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: 'Move gamma up' })
      ).toBeDisabled()
    )

    expect(badge).toHaveValue('Local draft')
    expect(onClose).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeEnabled()
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(onClose).toHaveBeenCalledOnce())
    expect(save).toHaveBeenCalledWith('/api/models/display', {
      order: ['gamma', 'beta', 'alpha'],
      models: { alpha: { badge: 'Local draft' } },
    })
  })

  it('keeps the edited order when cached settings refresh and saves it with untouched server decorations', async () => {
    const refresh = deferCachedDisplayRefresh()
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    const onClose = renderManager()
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Edit gamma' }))
    await user.click(screen.getByRole('button', { name: 'Move to top' }))

    await act(async () => {
      refresh({
        order: ['beta', 'alpha', 'gamma'],
        models: { gamma: { badge: 'Server badge' } },
      })
    })
    expect(await screen.findByDisplayValue('Server badge')).toBeVisible()

    expect(screen.getByRole('button', { name: 'Move gamma up' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeEnabled()
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(onClose).toHaveBeenCalledOnce())
    expect(save).toHaveBeenCalledWith('/api/models/display', {
      order: ['gamma', 'alpha', 'beta'],
      models: { gamma: { badge: 'Server badge' } },
    })
  })

  it('moves a model using keyboard arrows and preserves hidden model settings when saving', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        success: true,
        data: {
          order: ['alpha', 'hidden', 'beta', 'gamma'],
          models: { hidden: { note: 'Preserve' } },
        },
      },
    })
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    renderManager()
    const user = userEvent.setup()
    const drag = await screen.findByRole('button', {
      name: 'Drag beta to reorder',
    })
    drag.focus()
    await user.keyboard('{ArrowUp}')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(save).toHaveBeenCalled())
    expect(save.mock.calls[0][1]).toEqual({
      order: ['beta', 'hidden', 'alpha', 'gamma'],
      models: { hidden: { note: 'Preserve' } },
    })
  })

  it('rejects out-of-range positions and moves the selected model to a valid position', async () => {
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    renderManager()
    const user = userEvent.setup()
    const position = await screen.findByLabelText('Move to position')
    await user.type(position, '9')
    expect(screen.getByRole('button', { name: /^Move$/ })).toBeDisabled()
    await user.clear(position)
    await user.type(position, '3')
    await user.click(screen.getByRole('button', { name: /^Move$/ }))
    expect(screen.getByRole('button', { name: 'Edit alpha' })).toHaveAttribute(
      'aria-pressed',
      'true'
    )
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(save).toHaveBeenCalled())
    expect((save.mock.calls[0][1] as ModelSquareDisplay).order).toEqual([
      'beta',
      'gamma',
      'alpha',
    ])
  })

  it('keeps an unsaved edit when cancellation is declined and closes only after discard', async () => {
    const onClose = renderManager()
    const user = userEvent.setup()
    await user.type(
      await screen.findByLabelText('Recommendation badge'),
      'Draft'
    )
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    const confirm = await screen.findByRole('alertdialog')
    expect(onClose).not.toHaveBeenCalled()
    await user.click(within(confirm).getByRole('button', { name: 'Cancel' }))
    expect(screen.getByLabelText('Recommendation badge')).toHaveValue('Draft')
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    await user.click(
      within(await screen.findByRole('alertdialog')).getByRole('button', {
        name: 'Discard changes',
      })
    )
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('keeps the dialog and draft available after a failed save', async () => {
    vi.spyOn(api, 'put').mockRejectedValue(new Error('network unavailable'))
    const onClose = renderManager()
    const user = userEvent.setup()
    await user.type(await screen.findByLabelText('Card notice'), 'Draft notice')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Save changes' })).toBeEnabled()
    )
    expect(onClose).not.toHaveBeenCalled()
    expect(screen.getByLabelText('Card notice')).toHaveValue('Draft notice')
    expect(screen.getByRole('alert')).toHaveTextContent('network unavailable')
  })

  it('blocks saving an oversized tag and allows correction before saving', async () => {
    renderManager()
    const user = userEvent.setup()
    const tags = await screen.findByLabelText('Additional tags')
    await user.type(tags, `${'a'.repeat(33)}{Enter}`)
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled()
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Use up to 64 characters for series'
    )
    await user.click(screen.getByRole('button', { name: 'Remove tag' }))
    await user.type(tags, 'valid{Enter}')
    await waitFor(() =>
      expect(screen.getByRole('button', { name: 'Save changes' })).toBeEnabled()
    )
  })

  it('keeps other models in place when moving models within search results', async () => {
    const save = vi
      .spyOn(api, 'put')
      .mockImplementation(async (_url, data) => ({
        data: { success: true, data },
      }))
    const catalog = ['a-image', 'b-text', 'c-image'].map(
      (model_name, index) => ({ ...models[index], model_name })
    )
    renderManager(catalog)
    const user = userEvent.setup()
    await user.type(
      await screen.findByRole('textbox', { name: 'Model order' }),
      'image'
    )
    await user.click(screen.getByRole('button', { name: 'Move c-image up' }))
    await user.click(screen.getByRole('button', { name: 'Save changes' }))
    await waitFor(() => expect(save).toHaveBeenCalled())
    expect((save.mock.calls[0][1] as ModelSquareDisplay).order).toEqual([
      'c-image',
      'b-text',
      'a-image',
    ])
  })

  it('lets an administrator retry loading settings after a network failure', async () => {
    vi.mocked(api.get).mockRejectedValueOnce(new Error('offline'))
    renderManager()
    const user = userEvent.setup()
    await user.click(await screen.findByRole('button', { name: 'Retry' }))
    expect(await screen.findByLabelText('Recommendation badge')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled()
  })

  it('shows an empty state and disables save when no models are available', async () => {
    renderManager([])
    expect(await screen.findByText('No models found')).toBeVisible()
    expect(screen.getByRole('button', { name: 'Save changes' })).toBeDisabled()
  })
})
