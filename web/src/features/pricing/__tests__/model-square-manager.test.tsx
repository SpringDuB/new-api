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

function renderManager(catalog = models) {
  const onClose = vi.fn()
  render(
    <QueryClientProvider client={client}>
      <ModelSquareManager
        models={catalog}
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
      'Use up to 24 characters'
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
