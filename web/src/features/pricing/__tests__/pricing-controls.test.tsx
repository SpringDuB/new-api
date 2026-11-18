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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { PricingSidebar } from '../components/pricing-sidebar'
import {
  PricingToolbar,
  type PricingToolbarProps,
} from '../components/pricing-toolbar'
import { extractAllTags, filterByQuotaType } from '../lib/filters'
import { applyModelSquareDisplay } from '../lib/model-square-display'
import type { PricingModel } from '../types'

function toolbarProps(): PricingToolbarProps {
  return {
    categoryFilter: 'all',
    categoryCounts: { all: 2, video: 1, image: 1, chat: 0, other: 0 },
    onCategoryChange: vi.fn(),
    familyFilter: 'all',
    onFamilyChange: vi.fn(),
    filteredCount: 2,
    sortBy: 'name',
    tokenUnit: 'M',
    showRechargePrice: false,
    viewMode: 'card',
    groupFilter: 'all',
    onSortChange: vi.fn(),
    onTokenUnitChange: vi.fn(),
    onRechargePriceChange: vi.fn(),
    onViewModeChange: vi.fn(),
    onGroupChange: vi.fn(),
    groups: ['default', 'premium'],
    models: [],
    hasActiveFilters: false,
    activeFilterCount: 0,
    onClearFilters: vi.fn(),
  }
}

describe('pricing controls', () => {
  it('shows every model type immediately and reveals additional choices inline with the keyboard', async () => {
    const props = toolbarProps()
    const user = userEvent.setup()
    render(<PricingToolbar {...props} />)
    await user.click(screen.getByRole('button', { name: /^Videos/ }))
    expect(props.onCategoryChange).toHaveBeenCalledWith('video')
    expect(screen.getByRole('button', { name: /^Conversations/ })).toBeVisible()
    expect(
      screen.queryByRole('group', { name: 'Groups' })
    ).not.toBeInTheDocument()
    const more = screen.getByRole('button', { name: 'More filters' })
    expect(more).toHaveAttribute('aria-expanded', 'false')
    more.focus()
    await user.keyboard('{Enter}')
    expect(more).toHaveAttribute('aria-expanded', 'true')
    await user.click(
      within(screen.getByRole('group', { name: 'Groups' })).getByRole(
        'button',
        { name: 'premium' }
      )
    )
    expect(props.onGroupChange).toHaveBeenCalledWith('premium')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    for (const name of [
      'Vendor',
      'Model Tags',
      'Pricing Type',
      'Endpoint Type',
      'Model series',
      'Groups',
    ]) {
      expect(screen.queryByRole('combobox', { name })).not.toBeInTheDocument()
    }
  })

  it('keeps an unavailable selected series and group readable so they can be cleared', () => {
    render(
      <PricingSidebar
        {...toolbarProps()}
        familyFilter='series:Seedance 2.5'
        groupFilter='retired-group'
        showGroups
      />
    )
    expect(
      screen.getByRole('button', { name: /Seedance 2.5/ })
    ).toHaveAttribute('aria-pressed', 'true')
    expect(
      screen.getByRole('button', { name: 'retired-group' })
    ).toHaveAttribute('aria-pressed', 'true')
  })

  it('deduplicates full display tags without adding removed tag controls', () => {
    const label = 'High quality; text|image'
    const models = applyModelSquareDisplay(
      [
        {
          id: 1,
          model_name: 'example-model',
          quota_type: 1,
          model_ratio: 1,
          completion_ratio: 1,
          enable_groups: ['default'],
        },
      ],
      { order: [], models: { 'example-model': { tags: [label, label] } } }
    )
    expect(extractAllTags(models)).toEqual([label.toLowerCase()])
  })

  it('offers series, recommendation and price sorting through the shared select', async () => {
    const props = toolbarProps()
    render(<PricingToolbar {...props} />)
    const user = userEvent.setup()
    await user.click(screen.getByRole('combobox', { name: 'Sort' }))
    expect(
      screen.getByRole('option', { name: 'Recommended order' })
    ).toBeVisible()
    expect(screen.getByRole('option', { name: 'Model series' })).toBeVisible()
    await user.click(
      screen.getByRole('option', { name: 'Input Price: Low to High' })
    )
    expect(props.onSortChange).toHaveBeenCalledWith('input-price-low')
  })

  it('opens display settings separately and preserves the selected token unit', async () => {
    const props = toolbarProps()
    const user = userEvent.setup()
    const { rerender } = render(<PricingToolbar {...props} />)
    await user.click(screen.getByRole('button', { name: 'Display' }))
    const dialog = screen.getByRole('dialog', { name: 'Display' })
    await user.click(within(dialog).getByRole('radio', { name: '/1K' }))
    expect(props.onTokenUnitChange).toHaveBeenCalledWith('K')
    rerender(<PricingToolbar {...props} tokenUnit='K' />)
    expect(screen.getByRole('radio', { name: '/1K' })).toBeChecked()
    await user.click(screen.getByRole('radio', { name: '/1K' }))
    expect(props.onTokenUnitChange).toHaveBeenCalledTimes(1)
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('switches display mode and currency with accessible radio controls', async () => {
    const props = toolbarProps()
    const user = userEvent.setup()
    render(<PricingToolbar {...props} />)
    await user.click(screen.getByRole('button', { name: 'Display' }))
    await user.click(screen.getByRole('radio', { name: 'Recharge' }))
    expect(props.onRechargePriceChange).toHaveBeenCalledWith(true)
    const table = screen.getByRole('radio', { name: 'Table view' })
    table.focus()
    await user.keyboard(' ')
    expect(props.onViewModeChange).toHaveBeenCalledWith('table')
  })

  it.each<{
    name: string
    pricing: Partial<PricingModel>
    matches: string[]
  }>([
    { name: 'legacy token pricing', pricing: {}, matches: ['token'] },
    {
      name: 'legacy per-request pricing',
      pricing: { quota_type: 1, model_price: 0.01 },
      matches: ['request'],
    },
    {
      name: 'dynamic fixed pricing with a legacy token quota type',
      pricing: {
        billing_mode: 'tiered_expr',
        billing_expr: 'tier("request", fixed(0.01))',
      },
      matches: ['request'],
    },
    {
      name: 'free dynamic per-request pricing',
      pricing: {
        billing_mode: 'tiered_expr',
        billing_expr: 'v1:tier("free", fixed(0))',
      },
      matches: ['request'],
    },
    {
      name: 'request-dependent fixed tiers with an extra multiplier',
      pricing: {
        billing_mode: 'tiered_expr',
        billing_expr:
          '(param("quality") == "hd" ? tier("hd", fixed(0.02)) : tier("base", fixed(0.01))) * (header("x-fast") == "true" ? 2 : 1)',
      },
      matches: ['request'],
    },
    {
      name: 'mixed pricing across nested time tiers classified only as token-based',
      pricing: {
        billing_mode: 'tiered_expr',
        billing_expr:
          'hour("UTC") < 12 ? (len < 1000 ? tier("short", fixed(0.01)) : tier("long", p * 2 + c * 8)) : tier("night", fixed(0.02))',
      },
      matches: ['token'],
    },
    {
      name: 'mixed pricing overriding a legacy request quota type',
      pricing: {
        quota_type: 1,
        billing_mode: 'tiered_expr',
        billing_expr:
          'len < 1000 ? tier("short", fixed(0.01)) : tier("long", p * 2 + c * 8)',
      },
      matches: ['token'],
    },
    {
      name: 'dynamic token pricing overriding a legacy request quota type',
      pricing: {
        quota_type: 1,
        billing_mode: 'tiered_expr',
        billing_expr: 'tier("base", p * 2 + c * 8)',
      },
      matches: ['token'],
    },
    {
      name: 'token pricing with fixed syntax only in its tier label',
      pricing: {
        billing_mode: 'tiered_expr',
        billing_expr: 'tier("fixed(0.01)", p * 2)',
      },
      matches: ['token'],
    },
    {
      name: 'task usage retaining its separate category',
      pricing: {
        billing_mode: 'tiered_expr',
        billing_expr: 'tier("base", u("seconds") * 0.4)',
        billing_usage_schema: { seconds: { type: 'number', unit: 'second' } },
      },
      matches: ['task'],
    },
    {
      name: 'unparseable dynamic pricing retaining its legacy category',
      pricing: {
        quota_type: 1,
        billing_mode: 'tiered_expr',
        billing_expr: 'tier("request", fixed(',
      },
      matches: ['request'],
    },
  ])(
    'classifies $name consistently when reused by catalog sorting',
    ({ pricing, matches }) => {
      const model: PricingModel = {
        id: 1,
        model_name: 'pricing-model',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: ['default'],
        ...pricing,
      }
      const models = [model]
      for (const [quotaType] of [
        ['token', 'Token-based'],
        ['request', 'Per Request'],
        ['task', 'Task billing'],
      ]) {
        const included = matches.includes(quotaType)
        expect(filterByQuotaType(models, quotaType)).toEqual(
          included ? [model] : []
        )
      }
      expect(filterByQuotaType(models, 'all')).toEqual([model])
    }
  )
})
