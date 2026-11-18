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
    familyFilter: 'all',
    onFamilyChange: vi.fn(),
    filteredCount: 2,
    totalCount: 2,
    sortBy: 'name',
    tokenUnit: 'M',
    showRechargePrice: false,
    viewMode: 'card',
    quotaTypeFilter: 'all',
    endpointTypeFilter: 'all',
    vendorFilter: 'all',
    groupFilter: 'all',
    tagFilter: 'all',
    onSortChange: vi.fn(),
    onTokenUnitChange: vi.fn(),
    onRechargePriceChange: vi.fn(),
    onViewModeChange: vi.fn(),
    onQuotaTypeChange: vi.fn(),
    onEndpointTypeChange: vi.fn(),
    onVendorChange: vi.fn(),
    onGroupChange: vi.fn(),
    onTagChange: vi.fn(),
    vendors: [],
    groups: ['default', 'premium'],
    groupRatios: { default: 1, premium: 3 },
    tags: [],
    models: [],
    hasActiveFilters: false,
    activeFilterCount: 0,
    onClearFilters: vi.fn(),
  }
}

describe('pricing controls', () => {
  it('offers series sorting alongside the saved recommendation and name order', async () => {
    const props = toolbarProps()
    render(<PricingToolbar {...props} sortBy='family' />)
    const user = userEvent.setup()

    await user.click(screen.getByRole('combobox', { name: 'Sort' }))
    expect(
      screen.getByRole('option', { name: 'Recommended order' })
    ).toBeVisible()
    expect(screen.getByRole('option', { name: 'Name' })).toBeVisible()
    await user.click(screen.getByRole('option', { name: 'Model series' }))

    expect(props.onSortChange).toHaveBeenCalledWith('family')
  })

  it('counts a compound additional tag once and selects the complete filter label', async () => {
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
    const props = toolbarProps()
    render(
      <PricingSidebar
        {...props}
        models={models}
        tags={extractAllTags(models)}
      />
    )
    const user = userEvent.setup()

    const tags = screen.getByRole('combobox', { name: 'Model Tags' })
    await user.click(tags)
    await user.type(tags, 'text|image')
    expect(
      screen.getByRole('option', { name: /^high quality; text\|image · 1$/ })
    ).toBeVisible()
    expect(screen.getAllByRole('option')).toHaveLength(1)
    await user.keyboard('{ArrowDown}{Enter}')
    expect(props.onTagChange).toHaveBeenCalledWith(label.toLowerCase())
  })

  it('keeps an active series readable when the current search has no matching options and ignores an invalid typed selection', async () => {
    const props = toolbarProps()
    render(<PricingSidebar {...props} familyFilter='series:Seedance 2.5' />)
    const user = userEvent.setup()
    const series = screen.getByRole('combobox', { name: 'Model series' })
    expect(series).toHaveValue('Seedance 2.5')
    await user.click(series)
    await user.type(series, 'missing-series')
    expect(screen.queryByRole('option')).not.toBeInTheDocument()
    await user.keyboard('{Enter}')
    expect(props.onFamilyChange).not.toHaveBeenCalled()
    expect(series).toHaveValue('Seedance 2.5')
  })

  it('exposes each of the six filters as a labeled selector and retains their individual callbacks', async () => {
    const props = toolbarProps()
    const model: PricingModel = {
      id: 1,
      model_name: 'seedance-2-5',
      vendor_name: 'Studio',
      quota_type: 1,
      model_price: 0.4,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: ['premium'],
      supported_endpoint_types: ['openai-video'],
      tags: 'video',
    }
    render(
      <PricingSidebar
        {...props}
        models={[model]}
        vendors={[{ id: 1, name: 'Studio' }]}
        tags={['video']}
      />
    )
    const user = userEvent.setup()
    const choices = [
      {
        label: 'Model series',
        option: /^Seedance 2.5 · 1$/,
        callback: props.onFamilyChange,
        value: 'series:Seedance 2.5',
      },
      {
        label: 'Vendor',
        option: /^Studio · 1$/,
        callback: props.onVendorChange,
        value: 'Studio',
      },
      {
        label: 'Groups',
        option: /^premium · x3$/,
        callback: props.onGroupChange,
        value: 'premium',
      },
      {
        label: 'Model Tags',
        option: /^video · 1$/,
        callback: props.onTagChange,
        value: 'video',
      },
      {
        label: 'Pricing Type',
        option: /^Per Request · 1$/,
        callback: props.onQuotaTypeChange,
        value: 'request',
      },
      {
        label: 'Endpoint Type',
        option: /^Video · 1$/,
        callback: props.onEndpointTypeChange,
        value: 'openai-video',
      },
    ]
    for (const choice of choices) {
      await user.click(screen.getByRole('combobox', { name: choice.label }))
      await user.click(screen.getByRole('option', { name: choice.option }))
      expect(choice.callback).toHaveBeenCalledWith(choice.value)
    }
  })

  it('keeps the closed toolbar limited to a text sort and one settings entry', () => {
    render(<PricingToolbar {...toolbarProps()} />)
    expect(screen.getAllByRole('button')).toHaveLength(1)
    expect(
      screen.getByRole('button', { name: 'Filters and display' })
    ).toBeVisible()
    expect(screen.getByRole('combobox', { name: 'Sort' })).toBeVisible()
    expect(screen.queryByRole('radio')).not.toBeInTheDocument()
    expect(screen.queryByRole('tab')).not.toBeInTheDocument()
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
    'keeps filter results and sidebar counts consistent for $name',
    async ({ pricing, matches }) => {
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
      render(<PricingSidebar {...toolbarProps()} models={models} />)
      await userEvent
        .setup()
        .click(screen.getByRole('combobox', { name: 'Pricing Type' }))

      for (const [quotaType, label] of [
        ['token', 'Token-based'],
        ['request', 'Per Request'],
        ['task', 'Task billing'],
      ]) {
        const included = matches.includes(quotaType)
        expect(filterByQuotaType(models, quotaType)).toEqual(
          included ? [model] : []
        )
        expect(
          screen.getByRole('option', {
            name: new RegExp(`^${label} · ${included ? 1 : 0}$`),
          })
        ).toBeVisible()
      }
      expect(filterByQuotaType(models, 'all')).toEqual([model])
      expect(
        screen.getByRole('option', { name: /^All Models · 1$/ })
      ).toBeVisible()
    }
  )

  it('counts each model once in searchable filters and updates options when the catalog changes', async () => {
    const props = toolbarProps()
    const base: PricingModel = {
      id: 1,
      model_name: 'text-model',
      vendor_name: 'Vendor A',
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: ['default'],
      tags: 'Chat,chat',
      supported_endpoint_types: ['openai', 'openai'],
    }
    const models: PricingModel[] = [
      base,
      {
        ...base,
        id: 2,
        model_name: 'image-model',
        quota_type: 1,
        tags: 'chat,Image',
        supported_endpoint_types: ['image-generation'],
      },
      {
        ...base,
        id: 3,
        model_name: 'task-model',
        vendor_name: 'Vendor B',
        tags: 'Video',
        supported_endpoint_types: ['openai-video'],
        billing_usage_schema: { seconds: { type: 'number', unit: 'second' } },
      },
    ]
    const sidebarProps = {
      ...props,
      vendors: [
        { id: 1, name: 'Vendor A' },
        { id: 2, name: 'Vendor B' },
      ],
      tags: ['Chat', 'Image', 'Video'],
    }
    const { rerender } = render(
      <PricingSidebar {...sidebarProps} models={models} />
    )
    const user = userEvent.setup()
    await user.click(screen.getByRole('combobox', { name: 'Vendor' }))

    expect(
      screen.getByRole('option', { name: /^All Vendors · 3$/ })
    ).toBeVisible()
    expect(screen.getByRole('option', { name: /^Vendor A · 2$/ })).toBeVisible()
    expect(screen.getByRole('option', { name: /^Vendor B · 1$/ })).toBeVisible()
    await user.click(screen.getByRole('combobox', { name: 'Model Tags' }))
    expect(screen.getByRole('option', { name: /^Chat · 2$/ })).toBeVisible()
    await user.click(screen.getByRole('combobox', { name: 'Endpoint Type' }))
    expect(screen.getByRole('option', { name: /^Chat · 1$/ })).toBeVisible()
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('combobox', { name: 'Pricing Type' }))
    expect(
      screen.getByRole('option', { name: /^Token-based · 1$/ })
    ).toBeVisible()
    expect(
      screen.getByRole('option', { name: /^Per Request · 1$/ })
    ).toBeVisible()
    expect(
      screen.getByRole('option', { name: /^Task billing · 1$/ })
    ).toBeVisible()

    rerender(<PricingSidebar {...sidebarProps} models={[models[1]]} />)
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('combobox', { name: 'Vendor' }))
    expect(
      screen.getByRole('option', { name: /^All Vendors · 1$/ })
    ).toBeVisible()
    expect(screen.getByRole('option', { name: /^Vendor A · 1$/ })).toBeVisible()
    expect(screen.queryByRole('option', { name: /^Vendor B · 1$/ })).toBeNull()
    await user.click(screen.getByRole('combobox', { name: 'Model Tags' }))
    expect(screen.getByRole('option', { name: /^Chat · 1$/ })).toBeVisible()
    await user.click(screen.getByRole('combobox', { name: 'Endpoint Type' }))
    expect(screen.getByRole('option', { name: /^Chat · 0$/ })).toBeVisible()
    await user.keyboard('{Escape}')
    await user.click(screen.getByRole('combobox', { name: 'Pricing Type' }))
    expect(
      screen.getByRole('option', { name: /^Token-based · 0$/ })
    ).toBeVisible()
    expect(
      screen.getByRole('option', { name: /^Task billing · 0$/ })
    ).toBeVisible()
  })

  it('moves token units into the display tab and keeps the chosen radio selected when clicked again', async () => {
    const props = toolbarProps()
    const user = userEvent.setup()
    const { rerender } = render(<PricingToolbar {...props} />)
    expect(screen.queryByRole('radio', { name: '/1K' })).not.toBeInTheDocument()
    await user.click(
      screen.getByRole('button', { name: 'Filters and display' })
    )
    await user.click(screen.getByRole('tab', { name: 'Display' }))
    await user.click(screen.getByRole('radio', { name: '/1K' }))
    expect(props.onTokenUnitChange).toHaveBeenCalledWith('K')
    rerender(<PricingToolbar {...props} tokenUnit='K' />)
    const selected = screen.getByRole('radio', { name: '/1K' })
    expect(selected).toBeChecked()
    await user.click(selected)
    expect(selected).toBeChecked()
    expect(props.onTokenUnitChange).toHaveBeenCalledTimes(1)
  })

  it('changes the recharge display mode with an accessible selected state', async () => {
    const props = toolbarProps()
    const user = userEvent.setup()
    const { rerender } = render(<PricingToolbar {...props} />)
    await user.click(
      screen.getByRole('button', { name: 'Filters and display' })
    )
    await user.click(screen.getByRole('tab', { name: 'Display' }))
    await user.click(screen.getByRole('radio', { name: 'Recharge' }))
    expect(props.onRechargePriceChange).toHaveBeenCalledWith(true)
    rerender(<PricingToolbar {...props} showRechargePrice />)
    expect(screen.getByRole('radio', { name: 'Recharge' })).toBeChecked()
    expect(screen.getByRole('radio', { name: 'Standard' })).not.toBeChecked()
  })

  it('switches to table view with the keyboard and exposes the selected view', async () => {
    const props = toolbarProps()
    const user = userEvent.setup()
    const { rerender } = render(<PricingToolbar {...props} />)
    await user.click(
      screen.getByRole('button', { name: 'Filters and display' })
    )
    await user.click(screen.getByRole('tab', { name: 'Display' }))
    const tableButton = screen.getByRole('radio', { name: 'Table view' })
    tableButton.focus()
    await user.keyboard(' ')
    expect(props.onViewModeChange).toHaveBeenCalledWith('table')
    rerender(<PricingToolbar {...props} viewMode='table' />)
    expect(tableButton).toBeChecked()
    expect(screen.getByRole('radio', { name: 'List view' })).not.toBeChecked()
  })

  it('selects price sorting from the shared dropdown', async () => {
    const props = toolbarProps()
    const user = userEvent.setup()
    render(<PricingToolbar {...props} />)
    await user.click(screen.getByRole('combobox', { name: 'Sort' }))
    await user.click(
      screen.getByRole('option', { name: 'Input Price: Low to High' })
    )
    expect(props.onSortChange).toHaveBeenCalledWith('input-price-low')
  })

  it.each([390, 1440])(
    'opens filters from the right at %ipx, preserves selection, and restores focus on close',
    async (width) => {
      vi.stubGlobal('innerWidth', width)
      const props = toolbarProps()
      const user = userEvent.setup()
      const { rerender } = render(<PricingToolbar {...props} />)
      const trigger = screen.getByRole('button', {
        name: 'Filters and display',
      })
      expect(trigger).not.toHaveClass('xl:hidden')
      expect(trigger).toHaveAttribute('aria-expanded', 'false')
      await user.click(trigger)
      const dialog = await screen.findByRole('dialog', {
        name: 'Filters and display',
      })
      expect(dialog).toHaveAttribute('data-side', 'right')
      expect(trigger).toHaveAttribute('aria-expanded', 'true')
      expect(
        within(dialog).getByRole('button', { name: 'Reset' })
      ).toBeDisabled()
      await user.click(within(dialog).getByRole('combobox', { name: 'Groups' }))
      await user.click(screen.getByRole('option', { name: /premium/ }))
      expect(props.onGroupChange).toHaveBeenCalledWith('premium')
      rerender(
        <PricingToolbar
          {...props}
          groupFilter='premium'
          hasActiveFilters
          activeFilterCount={1}
        />
      )
      expect(
        within(dialog).getByRole('combobox', { name: 'Groups' })
      ).toHaveValue('premium · x3')
      await user.click(within(dialog).getByRole('button', { name: 'Reset' }))
      expect(props.onClearFilters).toHaveBeenCalledOnce()
      await user.keyboard('{Escape}')
      expect(
        await screen.findByRole('button', { name: /Filter/ })
      ).toHaveFocus()
      expect(trigger).toHaveAttribute('aria-expanded', 'false')
      vi.unstubAllGlobals()
    }
  )
})
