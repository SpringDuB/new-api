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
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createJSONStorage } from 'zustand/middleware'

import { api } from '@/lib/api'
import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { CachedPriceCell } from '../components/cached-price-cell'
import { ModelCard } from '../components/model-card'
import { ModelCardGrid } from '../components/model-card-grid'
import { assignCatalogCovers } from '../lib/catalog-covers'
import { filterBySearch } from '../lib/filters'
import { applyModelSquareDisplay } from '../lib/model-square-display'
import type { PricingModel } from '../types'

function pricingModel(overrides: Partial<PricingModel> = {}): PricingModel {
  return {
    id: 1,
    model_name: 'example-model',
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 3,
    enable_groups: ['default', 'premium'],
    group_ratio: { default: 1, premium: 3 },
    ...overrides,
  }
}

let queryClient: QueryClient
const originalStorage = useSystemConfigStore.persist.getOptions().storage
beforeEach(() => {
  const storage = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (key: string) => storage.get(key) ?? null,
    setItem: (key: string, value: string) => storage.set(key, value),
    removeItem: (key: string) => storage.delete(key),
    clear: () => storage.clear(),
    key: (index: number) => [...storage.keys()][index] ?? null,
    get length() {
      return storage.size
    },
  })
  useSystemConfigStore.persist.setOptions({
    storage: createJSONStorage(() => localStorage),
  })
  useSystemConfigStore.getState().setConfig({
    currency: { ...DEFAULT_CURRENCY_CONFIG, quotaDisplayType: 'USD' },
  })
  queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
})
afterEach(() => {
  queryClient.clear()
  useSystemConfigStore
    .getState()
    .setConfig({ currency: { ...DEFAULT_CURRENCY_CONFIG } })
  vi.useRealTimers()
  vi.unstubAllGlobals()
  useSystemConfigStore.persist.setOptions({ storage: originalStorage })
})

describe('model cards', () => {
  it('quotes the selected image tier, with the lowest first and no invented reference cap', async () => {
    const onClick = vi.fn()
    const user = userEvent.setup()
    render(
      <ModelCard
        onClick={onClick}
        presentation='directory'
        selectedGroup='premium'
        model={pricingModel({
          image_specs: [
            {
              resolution: '4K',
              rate: { unit: 'image', amount: 0.06 },
              maxImages: 9,
            },
            { resolution: '1K', rate: { unit: 'image', amount: 0.032 } },
          ],
        })}
      />
    )
    const prices = screen.getByRole('group', { name: 'Pricing' })
    expect(prices).toHaveTextContent('0.096')
    expect(prices).toHaveTextContent('/ image')
    expect(screen.getByText('Reference images')).toBeVisible()
    await user.click(screen.getByRole('button', { name: '4K' }))
    expect(prices).toHaveTextContent('0.18')
    expect(screen.getByText('Reference images ≤9')).toBeVisible()
    expect(onClick).not.toHaveBeenCalled()
  })
  it('switches resolution, its quote and its actual input combinations without opening Details', async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    useSystemConfigStore.getState().setConfig({
      currency: {
        ...DEFAULT_CURRENCY_CONFIG,
        quotaDisplayType: 'CNY',
        usdExchangeRate: 1,
      },
    })
    const combo = {
      minima: [0, 0, 0],
      maxima: [9, 3, 3],
      total: 12,
      nativeFace: true,
    } as const
    const model = pricingModel({
      video_specs: [
        {
          resolution: '1080p',
          preferred: true,
          combinations: [
            { minima: [0, 0, 0], maxima: [30, 0, 0], total: 30 },
            { minima: [0, 1, 0], maxima: [0, 1, 0], total: 1 },
          ],
          without_reference: [{ unit: 'second', amount: 0.65 }],
          with_reference: [{ unit: 'second', amount: 0.65 }],
        },
        {
          resolution: '720p',
          combinations: [{ ...combo, nativeFace: false }, combo],
          without_reference: [{ unit: 'second', amount: 0.39 }],
          with_reference: [
            { unit: 'second', amount: 0.39 },
            { unit: 'reference_second', amount: 0.31 },
          ],
        },
      ],
    })
    const { rerender } = render(
      <ModelCard presentation='directory' model={model} onClick={onClick} />
    )
    const prices = () => screen.getByRole('group', { name: 'Pricing' })
    expect(prices()).toHaveTextContent('¥0.39')
    expect(prices()).toHaveTextContent('¥0.31')
    expect(screen.getByText('Native face support')).toBeVisible()
    expect(screen.getAllByText('Images ≤9')).toHaveLength(1)
    expect(screen.queryByText('or')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: '1080p' }))
    expect(prices()).toHaveTextContent('¥0.65')
    expect(prices()).not.toHaveTextContent('¥0.31')
    const limits = screen.getByRole('group', {
      name: 'Reference material limits',
    })
    expect(limits).toHaveTextContent('Images ≤30')
    expect(limits).not.toHaveTextContent('Videos')
    expect(screen.queryByText('Native face support')).toBeNull()
    expect(onClick).not.toHaveBeenCalled()
    rerender(
      <ModelCard
        presentation='directory'
        model={model}
        selectedGroup='premium'
        onClick={onClick}
      />
    )
    expect(prices()).toHaveTextContent('¥1.95')
    await user.click(screen.getByRole('button', { name: 'Details' }))
    expect(onClick).toHaveBeenCalledWith(model.model_name)
  })

  it('keeps material limits and duration-specific surcharges tied to the selected provider', async () => {
    const user = userEvent.setup()
    const base = {
      resolution: '768P',
      combinations: [{ minima: [0, 0, 0], maxima: [9, 0, 0], total: 9 }],
    } as const
    const rows: NonNullable<PricingModel['video_specs']> = [
      ...['short', 'long'].map((band, i) => ({
        ...base,
        combinations: [...base.combinations],
        provider: 'MS',
        duration_tier: band,
        duration_label: { en: i ? 'Over 15 seconds' : 'Up to 15 seconds' },
        includedImages: 5,
        without_reference: [
          { unit: 'second' as const, amount: i ? 0.24 : 0.14 },
          { unit: 'reference_image' as const, amount: 0.24 },
        ],
      })),
      {
        ...base,
        provider: 'Other',
        combinations: [{ minima: [0, 0, 0], maxima: [1, 0, 0], total: 1 }],
        without_reference: [{ unit: 'request', amount: 6 }],
      },
    ]
    render(
      <ModelCard
        model={pricingModel({ video_specs: rows })}
        onClick={vi.fn()}
      />
    )
    const prices = screen.getByRole('group', { name: 'Pricing' })
    expect(prices).toHaveTextContent('$0.14')
    await user.click(screen.getByRole('button', { name: 'Over 15 seconds' }))
    expect(prices).not.toHaveTextContent('$0.14')
    expect(prices).toHaveTextContent('After 5 images')
    await user.click(screen.getByRole('button', { name: 'Other' }))
    expect(prices).toHaveTextContent('$6')
    expect(prices).not.toHaveTextContent('After 5 images')
    expect(
      screen.getByRole('group', { name: 'Reference material limits' })
    ).toHaveTextContent('Images ≤1')
  })

  it.each([
    ['request', 5.5, 'call'],
    ['video_tokens', 48.7872, '1M tokens'],
  ] as const)(
    'shows a static resolution and the %s billing denominator',
    (unit, amount, label) => {
      render(
        <ModelCard
          onClick={vi.fn()}
          model={pricingModel({
            video_specs: [
              {
                resolution: '720p',
                preferred: true,
                combinations: [
                  { minima: [0, 0, 0], maxima: [1, 0, 0], total: 1 },
                ],
                without_reference: [{ unit, amount }],
              },
            ],
          })}
        />
      )
      expect(screen.getByText('720p')).toBeVisible()
      expect(screen.queryByRole('button', { name: '720p' })).toBeNull()
      expect(screen.getByRole('group', { name: 'Pricing' })).toHaveTextContent(
        label
      )
    }
  )

  it('shows the display name and API model ID without a vendor label', async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    const model = pricingModel({
      model_name: 'seedance-2-5-01',
      display_name: 'Seedance 2.5 · 多参考',
      vendor_name: 'ByteDance',
    })
    render(
      <ModelCard model={model} onClick={onClick} presentation='directory' />
    )
    expect(
      screen.getByRole('heading', { name: model.display_name })
    ).toBeVisible()
    expect(screen.getByText(model.model_name)).toBeVisible()
    expect(screen.queryByText('ByteDance')).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Copy model name' }))
    expect(await navigator.clipboard.readText()).toBe(model.model_name)
    await user.click(screen.getByRole('button', { name: 'Details' }))
    expect(onClick).toHaveBeenCalledWith(model.model_name)
    expect(filterBySearch([model], '多参考')).toEqual([model])
    expect(filterBySearch([model], model.model_name)).toEqual([model])
  })
  it.each([
    undefined,
    { avg_latency_ms: Number.NaN, avg_tps: 0, success_rate: Number.NaN },
  ])(
    'shows 24 unmeasured hours and keeps the details action: %j',
    async (perf) => {
      const onClick = vi.fn()
      render(
        <ModelCard
          model={pricingModel()}
          onClick={onClick}
          presentation='directory'
          perf={perf}
        />
      )
      const entry = screen.getByRole('article', { name: 'example-model' })
      expect(
        within(entry).getByRole('group', { name: 'Pricing' })
      ).toBeVisible()
      const strip = within(entry).getByRole('img', {
        name: 'Latest active hours',
      })
      expect(strip.children).toHaveLength(24)
      expect(within(entry).getByText('—')).toBeVisible()
      await userEvent
        .setup()
        .click(within(entry).getByRole('button', { name: 'Details' }))
      expect(onClick).toHaveBeenCalledWith('example-model')
    }
  )

  it('retains a reported zero success rate in the directory without treating it as missing data', () => {
    render(
      <ModelCard
        model={pricingModel()}
        onClick={vi.fn()}
        presentation='directory'
        perf={{
          avg_latency_ms: Number.NaN,
          avg_tps: Number.NaN,
          success_rate: 0,
        }}
      />
    )
    const metrics = screen.getByLabelText(
      'Success rate over the latest active hours'
    )
    expect(within(metrics).getByText('0.00%')).toBeVisible()
  })

  it('shows the latest active hours across seven days and the grading notes without opening model details', async () => {
    const end = 200 * 3600 + 135
    const onClick = vi.fn()
    render(
      <ModelCard
        model={pricingModel()}
        presentation='directory'
        onClick={onClick}
        perf={{
          window_end: end,
          success_rate: 72.35,
          avg_latency_ms: 31020,
          avg_tps: 0,
          recent_success_series: [
            [36, 100],
            [77, 50],
            [79, 0],
            [98, 89.99],
            [99, 90],
            [100, 49.99],
          ].map(([hour, success_rate]) => ({ ts: hour * 3600, success_rate })),
        }}
      />
    )
    expect(screen.getByText('72.35%')).toBeVisible()
    const cells = screen.getByRole('img', {
      name: 'Latest active hours',
    }).children
    expect(cells).toHaveLength(24)
    expect(cells[17]).toHaveAttribute('data-observed', 'false')
    expect(cells[18]).toHaveClass('bg-emerald-500')
    expect(cells[19]).toHaveClass('bg-amber-500')
    expect(cells[20]).toHaveClass('bg-red-500')
    expect(cells[21]).toHaveClass('bg-amber-500')
    expect(cells[22]).toHaveClass('bg-emerald-500')
    expect(cells[23]).toHaveClass('bg-red-500')
    await userEvent
      .setup()
      .click(screen.getByRole('button', { name: 'How this is calculated' }))
    expect(await screen.findByRole('dialog')).toHaveTextContent('90%')
    expect(onClick).not.toHaveBeenCalled()
  })

  it.each(['seedance-2-0-per-request', 'seedance-2-5-per-request'])(
    'shows material limits and description alongside anonymous call prices for %s',
    (name) => {
      const sd2 = name.includes('2-0')
      render(
        <ModelCard
          model={pricingModel({
            model_name: name,
            description: `Actual description for ${name}`,
            video_specs: [
              {
                provider: 'LecAPI',
                resolution: '720p',
                combinations: [
                  {
                    minima: [0, 0, 0],
                    maxima: sd2 ? [9, 3, 3] : [30, 10, 10],
                    total: sd2 ? 15 : 50,
                  },
                ],
                without_reference: [
                  { unit: 'request', amount: sd2 ? 2.9 : 5.5 },
                ],
                with_reference: [{ unit: 'request', amount: sd2 ? 3.3 : 5.5 }],
              },
            ],
          })}
          onClick={vi.fn()}
          presentation='directory'
        />
      )
      expect(screen.getByText('720p')).toBeVisible()
      const price = screen.getByRole('group', { name: 'Pricing' })
      expect(price).toHaveTextContent(sd2 ? '2.9' : '5.5')
      if (sd2) expect(price).toHaveTextContent('3.3')
      const materials = screen.getByRole('group', {
        name: 'Reference material limits',
      })
      expect(materials).toHaveTextContent(sd2 ? 'Images ≤9' : 'Images ≤30')
      expect(materials).toHaveTextContent(sd2 ? 'Videos ≤3' : 'Videos ≤10')
      expect(materials).toHaveTextContent(sd2 ? 'Audio ≤3' : 'Audio ≤10')
      expect(screen.getByText(`Actual description for ${name}`)).toHaveClass(
        'line-clamp-2'
      )
      expect(screen.queryByText('LecAPI')).not.toBeInTheDocument()
    }
  )

  it('shows all routable call prices without a provider switch or losing the video-only quote', () => {
    render(
      <ModelCard
        onClick={vi.fn()}
        presentation='directory'
        model={pricingModel({
          model_name: 'seedance-2-0-per-request',
          video_specs: [
            {
              provider: 'A',
              resolution: '720p',
              combinations: [
                { minima: [0, 0, 0], maxima: [1, 0, 0], total: 1 },
              ],
              without_reference: [{ unit: 'request', amount: 3 }],
            },
            {
              provider: 'B',
              resolution: '720p',
              combinations: [
                { minima: [0, 0, 0], maxima: [3, 0, 0], total: 3 },
                { minima: [0, 1, 0], maxima: [1, 1, 0], total: 2 },
              ],
              without_reference: [{ unit: 'request', amount: 2.9 }],
              with_reference: [{ unit: 'request', amount: 3.3 }],
            },
          ],
        })}
      />
    )
    const price = screen.getByRole('group', { name: 'Pricing' })
    expect(price).toHaveTextContent(/2\.9.*–.*3/)
    expect(price).toHaveTextContent(/With reference video.*3\.3/)
    expect(screen.queryByLabelText('Provider')).not.toBeInTheDocument()
    const limits = screen.getByRole('group', {
      name: 'Reference material limits',
    })
    expect(limits).toHaveTextContent('Images ≤3')
    expect(limits).not.toHaveTextContent('Videos')
    expect(limits).not.toHaveTextContent('Images ≤1')
    expect(within(limits).queryByText('or')).not.toBeInTheDocument()
  })

  it('uses a paginated tile gallery while preserving promotions, group metadata, and task price units', async () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { models: [] } },
    })
    render(
      <QueryClientProvider client={queryClient}>
        <ModelCardGrid
          models={[
            pricingModel({
              description: 'A video model with an actual duration price.',
              tags: 'video,production',
              supported_endpoint_types: ['openai-video'],
              square_display: {
                badge: 'Studio choice',
                note: 'Available today',
                highlight: true,
                color: 'red',
              },
              billing_mode: 'tiered_expr',
              billing_expr: 'tier("base", u("seconds") * 0.4)',
              billing_usage_schema: {
                seconds: { type: 'number', unit: 'second' },
              },
            }),
          ]}
          onModelClick={vi.fn()}
          presentation='directory'
          tokenUnit='K'
        />
      </QueryClientProvider>
    )
    const directory = screen.getByRole('list', { name: 'Models' })
    expect(directory).toHaveClass('stolas-model-gallery')
    const entry = within(directory).getByRole('article', {
      name: 'example-model',
    })
    expect(within(entry).getByText('Studio choice')).toBeVisible()
    expect(within(entry).getByText('Available today')).toBeVisible()
    expect(
      within(entry).getByText('A video model with an actual duration price.')
    ).toHaveClass('line-clamp-3')
    expect(within(entry).getByText('default +1')).toBeVisible()
    expect(within(entry).queryByText('openai-video')).not.toBeInTheDocument()
    expect(
      within(entry).queryByRole('group', { name: 'Tags' })
    ).not.toBeInTheDocument()
    const price = within(entry).getByRole('group', { name: 'Pricing' })
    expect(price).toHaveTextContent('$0.4 / s')
    expect(price).not.toHaveTextContent(/1K|1M/)
  })

  it('shows a compound additional tag as one label alongside the original catalog tag', () => {
    const label = 'High quality; text|image'
    const [model] = applyModelSquareDisplay([pricingModel({ tags: 'chat' })], {
      order: [],
      models: { 'example-model': { tags: [label] } },
    })
    render(<ModelCard model={model} onClick={vi.fn()} />)

    const tags = screen.getByRole('group', { name: 'Tags' })
    expect(within(tags).getByText(`chat, ${label}`)).toBeVisible()
    expect(within(tags).queryByText(/^\+/)).not.toBeInTheDocument()
  })

  it('shows a configured promotion as text with its border and notice while retaining long badge wrapping', () => {
    render(
      <ModelCard
        model={pricingModel({
          square_display: {
            badge: '<strong>Site choice</strong>',
            note: 'Available all day',
            highlight: true,
            color: 'red',
            badge_style: 'burst',
          },
        })}
        onClick={vi.fn()}
      />
    )
    const badge = screen.getByText('<strong>Site choice</strong>')
    expect(badge).toBeVisible()
    expect(badge.querySelector('strong')).toBeNull()
    expect(badge).toHaveClass('whitespace-normal')
    expect(badge.closest('[data-slot="card"]')).toHaveClass('ring-2')
    expect(screen.getByText('Available all day')).toBeVisible()
  })
  it('shows separate generic and image cache prices including a free image cache', () => {
    render(
      <CachedPriceCell
        model={pricingModel({
          billing_mode: 'tiered_expr',
          billing_expr:
            'tier("standard", p * 5 + cr * 1.25 + img * 8 + img_cr * 0 + c * 30)',
        })}
        options={{ tokenUnit: 'M' }}
      />
    )
    expect(screen.getByText('Cache Read').parentElement).toHaveTextContent(
      '$1.25'
    )
    expect(screen.getByText('Image Cache').parentElement).toHaveTextContent(
      '$0'
    )
  })
  it('shows fixed prices per request in both token display units', () => {
    const model = pricingModel({
      billing_mode: 'tiered_expr',
      billing_expr: 'tier("request", fixed(0.01))',
    })
    const { rerender } = render(
      <ModelCard model={model} onClick={vi.fn()} tokenUnit='K' />
    )
    expect(screen.getByText('$0.01')).toBeVisible()
    expect(screen.getByText('/ request')).toBeVisible()
    rerender(<ModelCard model={model} onClick={vi.fn()} tokenUnit='M' />)
    expect(screen.getByText('$0.01')).toBeVisible()
    expect(screen.queryByText('/ 1M')).not.toBeInTheDocument()
  })
  it('updates the current time tier at a minute boundary and after returning to the page', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-07T08:59:59+08:00'))
    const model = pricingModel({
      billing_mode: 'tiered_expr',
      billing_expr:
        'hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12 ? tier("peak", p * 3 + c * 9) : tier("off_peak", p * 1.5 + c * 4.5)',
    })
    render(<ModelCard model={model} onClick={vi.fn()} tokenUnit='M' />)
    expect(screen.getByText('Current period price')).toBeVisible()
    expect(screen.getByText('$1.5')).toBeVisible()
    act(() => vi.advanceTimersByTime(1000))
    expect(screen.getByText('$3')).toBeVisible()
    act(() => {
      vi.setSystemTime(new Date('2026-09-07T12:00:00+08:00'))
      fireEvent(document, new Event('visibilitychange'))
    })
    expect(screen.getByText('$1.5')).toBeVisible()
  })

  it('copies the complete long model name without opening details', async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()
    const name = 'provider/model-with-a-long-name-and-a-version-suffix-20260906'
    render(
      <ModelCard
        model={pricingModel({ model_name: name })}
        onClick={onClick}
        presentation='directory'
      />
    )
    const heading = screen.getByRole('heading', { name })
    expect(heading).toHaveClass('truncate')
    expect(heading).toHaveAttribute('title', name)
    await user.click(screen.getByRole('button', { name: 'Copy model name' }))
    expect(await navigator.clipboard.readText()).toBe(name)
    expect(onClick).not.toHaveBeenCalled()
    await user.click(screen.getByRole('button', { name: 'Details' }))
    expect(onClick).toHaveBeenCalledOnce()
    expect(onClick).toHaveBeenCalledWith(name)
  })

  it('shows missing values without placeholder bars when metrics are unavailable', () => {
    render(<ModelCard model={pricingModel()} onClick={vi.fn()} />)
    const metrics = screen.getByLabelText(
      'Performance metrics for the last 24 hours'
    )
    expect(within(metrics).getAllByText('—')).toHaveLength(2)
    expect(within(metrics).getByText('—s')).toBeVisible()
    expect(within(metrics).getByText('—t/s')).toBeVisible()
    expect(within(metrics).queryByText(/100/)).not.toBeInTheDocument()
    const statusStrip = within(metrics).getByRole('img', {
      name: 'No performance data available',
    })
    expect(statusStrip).toHaveTextContent('—')
    expect(statusStrip.querySelectorAll('[aria-hidden]')).toHaveLength(0)
    expect(screen.getByText('No description available.')).toBeVisible()
    expect(screen.getByRole('button', { name: 'Details' })).toBeEnabled()
  })

  it('uses fixed spacing between available status samples', () => {
    render(
      <ModelCard
        model={pricingModel()}
        onClick={vi.fn()}
        perf={{
          avg_latency_ms: 1200,
          avg_tps: 42,
          success_rate: 50,
          recent_success_series: [
            { ts: 3600, success_rate: 100 },
            { ts: 18000, success_rate: 0 },
          ],
        }}
      />
    )
    const statusStrip = screen.getByRole('img', {
      name: 'Success rate',
    })
    expect(statusStrip).toHaveClass('gap-px')
    expect(statusStrip).not.toHaveClass('justify-between')
  })

  it('shows server history for idle models while preserving the current window metrics', async () => {
    const request = vi.spyOn(api, 'get').mockResolvedValue({
      data: {
        success: true,
        data: {
          models: [
            {
              model_name: 'active-model',
              avg_latency_ms: 1200,
              avg_tps: 42,
              success_rate: 99.8,
              recent_success_series: [{ ts: 7200, success_rate: 0 }],
            },
          ],
          recent_success_series: {
            'active-model': [{ ts: 3600, success_rate: 100 }],
            'idle-model': [
              { ts: 3600, success_rate: 0 },
              { ts: 7200, success_rate: 100 },
            ],
          },
        },
      },
    })
    render(
      <QueryClientProvider client={queryClient}>
        <ModelCardGrid
          models={[
            pricingModel({ model_name: 'active-model' }),
            pricingModel({ id: 2, model_name: 'idle-model' }),
          ]}
          onModelClick={vi.fn()}
        />
      </QueryClientProvider>
    )
    await screen.findByText('99.80%')
    expect(request).toHaveBeenCalledWith('/api/perf-metrics/summary', {
      params: { hours: 24, recent_model: ['active-model', 'idle-model'] },
      paramsSerializer: { indexes: null },
    })
    const [active, idle] = screen.getAllByLabelText(
      'Performance metrics for the last 24 hours'
    )
    expect(within(active).getByText('1.20s')).toBeVisible()
    expect(
      within(active).getByRole('img', { name: 'Success rate' }).children[0]
    ).toHaveClass('bg-emerald-500')
    const samples = within(idle).getByRole('img', {
      name: 'Success rate',
    }).children
    expect(samples).toHaveLength(2)
    expect(samples[0]).toHaveClass('bg-red-500')
    expect(samples[1]).toHaveClass('bg-emerald-500')
    expect(within(idle).getByText('—')).toBeVisible()
    expect(within(idle).getByText('—s')).toBeVisible()
    expect(within(idle).getByText('—t/s')).toBeVisible()
  })

  it('keeps group, endpoint and tag overflow counts with their own metadata', () => {
    const groups = ['default-with-a-long-group-name', 'premium', 'internal']
    const endpoints = ['openai-response', 'openai', 'claude', 'gemini', 'jina']
    const tags = [
      'video-generation',
      'high-resolution',
      'fast',
      'batch',
      'hd',
      'pro',
    ]
    render(
      <ModelCard
        model={pricingModel({
          enable_groups: groups,
          supported_endpoint_types: endpoints,
          tags: tags.join(','),
        })}
        onClick={vi.fn()}
      />
    )

    const groupField = screen.getByText('Groups').parentElement
    const endpointField = screen.getByText('Endpoints').parentElement
    if (!groupField || !endpointField) {
      throw new Error('Expected labeled group and endpoint fields')
    }
    const tagField = screen.getByRole('group', { name: 'Tags' })
    expect(within(groupField).getByText(groups[0])).toBeVisible()
    expect(within(groupField).getByText('+2')).toHaveAttribute(
      'title',
      groups.slice(1).join(', ')
    )
    expect(
      within(endpointField).getByText('openai-response, openai')
    ).toHaveAttribute('title', endpoints.join(', '))
    expect(within(endpointField).getByText('+3')).toBeVisible()
    expect(
      within(tagField).getByText('video-generation, high-resolution')
    ).toHaveAttribute('title', tags.join(', '))
    expect(within(tagField).getByText('+4')).toBeVisible()
    expect(
      within(screen.getByRole('group', { name: 'Pricing' })).getByText(
        'Token-based'
      )
    ).toBeVisible()
  })

  it('omits metadata fields when the model has no groups, endpoints or tags', () => {
    render(
      <ModelCard
        model={pricingModel({ enable_groups: [] })}
        onClick={vi.fn()}
      />
    )
    expect(screen.queryByText('Groups')).not.toBeInTheDocument()
    expect(screen.queryByText('Endpoints')).not.toBeInTheDocument()
    expect(
      screen.queryByRole('group', { name: 'Tags' })
    ).not.toBeInTheDocument()
  })

  it.each([
    { success_rate: 0, expected: '0.00%' },
    { success_rate: 99.8, expected: '99.80%' },
    { success_rate: Number.NaN, expected: '—' },
  ])(
    'shows $expected for the reported request success rate $success_rate',
    ({ success_rate, expected }) => {
      render(
        <ModelCard
          model={pricingModel()}
          onClick={vi.fn()}
          perf={{ avg_latency_ms: 1200, avg_tps: 42, success_rate }}
        />
      )
      const metrics = screen.getByLabelText(
        'Performance metrics for the last 24 hours'
      )
      expect(within(metrics).getAllByText(expected)[0]).toBeVisible()
      expect(within(metrics).getByText('Status')).toBeVisible()
      expect(within(metrics).getByText('1.20s')).toBeVisible()
      expect(within(metrics).getByText('42.0t/s')).toBeVisible()
    }
  )

  it('keeps group and recharge pricing correct when changing the token unit, including a free cache price', () => {
    const props = {
      model: pricingModel({ cache_ratio: 0 }),
      onClick: vi.fn(),
      selectedGroup: 'premium',
      showRechargePrice: true,
      priceRate: 3,
      usdExchangeRate: 6,
    }
    const { rerender } = render(<ModelCard {...props} tokenUnit='M' />)
    expect(screen.getByText('Input').parentElement).toHaveTextContent(
      /\$3\s*\/\s*1M/
    )
    expect(screen.getByText('Output').parentElement).toHaveTextContent(
      /\$9\s*\/\s*1M/
    )
    expect(screen.getByText('Cached').parentElement).toHaveTextContent(
      /\$0\s*\/\s*1M/
    )
    rerender(<ModelCard {...props} tokenUnit='K' />)
    expect(screen.getByText('Input').parentElement).toHaveTextContent(
      /\$0.003\s*\/\s*1K/
    )
    expect(screen.getByText('Output').parentElement).toHaveTextContent(
      /\$0.009\s*\/\s*1K/
    )
    expect(screen.getByText('Cached').parentElement).toHaveTextContent(
      /\$0\s*\/\s*1K/
    )
  })

  it('shows a per-request price with the selected group and recharge multiplier without a token unit', () => {
    render(
      <ModelCard
        model={pricingModel({ quota_type: 1, model_price: 0.4 })}
        onClick={vi.fn()}
        selectedGroup='premium'
        showRechargePrice
        priceRate={3}
        usdExchangeRate={6}
        tokenUnit='K'
      />
    )
    expect(screen.getByText(/\$0.6/)).toHaveTextContent(/\$0.6\s*\/\s*request/)
    expect(screen.queryByText(/1K|1M/)).not.toBeInTheDocument()
    expect(screen.getAllByText('Per Request')).toHaveLength(1)
    expect(screen.queryByText('Per-request')).not.toBeInTheDocument()
  })

  it('preserves expression prices and makes the selected token unit explicit', () => {
    render(
      <ModelCard
        model={pricingModel({
          billing_mode: 'tiered_expr',
          billing_expr: 'tier("base", p * 3 + c * 15)',
        })}
        onClick={vi.fn()}
        tokenUnit='K'
        selectedGroup='premium'
        showRechargePrice
        priceRate={3}
        usdExchangeRate={6}
      />
    )
    expect(screen.getByText('Input').parentElement).toHaveTextContent(
      /\$0.0045\s*\/\s*1K/
    )
    expect(screen.getByText('Output').parentElement).toHaveTextContent(
      /\$0.0225\s*\/\s*1K/
    )
  })

  it('shows the cache read price of an expression model after its input and output prices', () => {
    render(
      <ModelCard
        model={pricingModel({
          billing_mode: 'tiered_expr',
          billing_expr: 'tier("base", p * 0.15 + c * 0.6 + cr * 0.003)',
        })}
        onClick={vi.fn()}
        tokenUnit='M'
        selectedGroup='default'
      />
    )
    expect(screen.getByText('Input').parentElement).toHaveTextContent(
      /\$0.15\s*\/\s*1M/
    )
    expect(screen.getByText('Output').parentElement).toHaveTextContent(
      /\$0.6\s*\/\s*1M/
    )
    expect(screen.getByText('Cache Read').parentElement).toHaveTextContent(
      /\$0.003\s*\/\s*1M/
    )
  })

  it('shows the cache read price once when a plugin provider already lists it as a main price', () => {
    render(
      <ModelCard
        model={pricingModel({
          billing_plugin_variants: [
            {
              plugin_key: 'alpha',
              plugin_name: 'Alpha',
              billing_mode: 'tiered_expr',
              billing_expr: 'tier("base", p * 1 + c * 2 + cr * 0.5)',
              billing_usage_schema: {},
            },
          ],
        })}
        onClick={vi.fn()}
        tokenUnit='M'
        selectedGroup='default'
      />
    )
    expect(screen.getAllByText('Cache Read')).toHaveLength(1)
  })

  it('shows the cache read price of an expression model that prices nothing else', () => {
    render(
      <ModelCard
        model={pricingModel({
          billing_mode: 'tiered_expr',
          billing_expr: 'tier("base", cr * 0.5)',
        })}
        onClick={vi.fn()}
        tokenUnit='M'
        selectedGroup='default'
      />
    )
    expect(screen.getByText('Cache Read').parentElement).toHaveTextContent(
      /\$0.5\s*\/\s*1M/
    )
  })

  it('keeps task price ranges in their actual usage unit instead of the selected token unit', () => {
    render(
      <ModelCard
        model={pricingModel({
          billing_mode: 'tiered_expr',
          billing_expr:
            'u("mode") == "pro" ? tier("pro", u("seconds") * 0.8) : tier("std", u("seconds") * 0.4)',
          billing_usage_schema: {
            seconds: { type: 'number', unit: 'second' },
            mode: { enum: ['std', 'pro'] },
          },
        })}
        onClick={vi.fn()}
        tokenUnit='K'
      />
    )
    expect(screen.getByText(/0.4.*0.8/)).toHaveTextContent(/0.4 – \$0.8/)
    expect(screen.getByText(/^\/\s*s$/)).toBeVisible()
    expect(screen.queryByText(/1K|1M/)).not.toBeInTheDocument()
  })

  it('hides a task usage price left at zero while another usage price is set', () => {
    render(
      <ModelCard
        model={pricingModel({
          billing_mode: 'tiered_expr',
          billing_expr: 'tier("base", u("count") * 0 + u("seconds") * 0.7)',
          billing_usage_schema: {
            count: {
              type: 'number',
              unit: 'count',
              description: { en: 'Per-task price' },
            },
            seconds: {
              type: 'number',
              unit: 'second',
              description: { en: 'Video generation unit price' },
            },
          },
        })}
        onClick={vi.fn()}
      />
    )
    expect(screen.getByText('Video generation unit price')).toBeVisible()
    expect(screen.queryByText('Per-task price')).not.toBeInTheDocument()
  })

  it('does not show the tier name of a single-tier task price', () => {
    render(
      <ModelCard
        model={pricingModel({
          billing_mode: 'tiered_expr',
          billing_expr: 'tier("base", u("seconds") * 0.4)',
          billing_usage_schema: { seconds: { type: 'number', unit: 'second' } },
        })}
        onClick={vi.fn()}
      />
    )
    expect(screen.getByText(/^\$0\.4$/)).toBeVisible()
    expect(screen.queryByText('(base)')).not.toBeInTheDocument()
  })

  it('shows the unconfigured usage message without inventing a token price', () => {
    render(
      <ModelCard
        model={pricingModel({
          billing_usage_schema: { seconds: { type: 'number', unit: 'second' } },
        })}
        onClick={vi.fn()}
      />
    )
    expect(
      screen.getByText('Usage-based billing · price not configured')
    ).toBeVisible()
    expect(screen.queryByText('Input')).not.toBeInTheDocument()
  })

  it.each(['card', 'directory'] as const)(
    'shows a task token range with its unit and only shows an example price outside the directory (%s)',
    (presentation) => {
      render(
        <ModelCard
          model={pricingModel({
            billing_mode: 'tiered_expr',
            billing_expr:
              'u("mode") == "pro" ? tier("pro", u("tokens") * 70 / 1000000) : tier("std", u("tokens") * 42 / 1000000)',
            billing_usage_schema: {
              tokens: { type: 'number', unit: 'token' },
              mode: { enum: ['std', 'pro'] },
            },
            billing_usage_examples: [
              { label: '480p · 5s', facts: { tokens: 48000, mode: 'std' } },
            ],
          })}
          onClick={vi.fn()}
          tokenUnit='K'
          presentation={presentation}
        />
      )
      expect(screen.getByText('$42 – $70').parentElement).toHaveTextContent(
        '$42 – $70 / 1M token'
      )
      if (presentation === 'directory') {
        expect(screen.queryByText(/480p · 5s ≈/)).not.toBeInTheDocument()
        expect(screen.getByRole('group', { name: 'Pricing' })).toHaveClass(
          'truncate'
        )
      } else {
        expect(screen.getByText(/480p · 5s ≈/)).toBeVisible()
      }
    }
  )

  it('keeps an unrecognized expression visible with the special billing message', () => {
    const expression =
      'u("seconds") > 30 ? tier("long", u("seconds") * 0.3) : tier("short", u("seconds") * 0.4)'
    render(
      <ModelCard
        model={pricingModel({
          billing_mode: 'tiered_expr',
          billing_expr: expression,
          billing_usage_schema: { seconds: { type: 'number', unit: 'second' } },
        })}
        onClick={vi.fn()}
      />
    )
    expect(screen.getByText('Special billing expression')).toBeVisible()
    expect(screen.getByText(expression)).toBeVisible()
  })

  it('keeps browsing and missing metric values available after the metrics request fails', async () => {
    const request = vi
      .spyOn(api, 'get')
      .mockRejectedValue(new Error('metrics unavailable'))
    const onModelClick = vi.fn()
    render(
      <QueryClientProvider client={queryClient}>
        <ModelCardGrid models={[pricingModel()]} onModelClick={onModelClick} />
      </QueryClientProvider>
    )
    await waitFor(() =>
      expect(
        queryClient.getQueryState([
          'perf-metrics-summary',
          24,
          ['example-model'],
        ])?.status
      ).toBe('error')
    )
    expect(request).toHaveBeenCalledWith('/api/perf-metrics/summary', {
      params: { hours: 24, recent_model: ['example-model'] },
      paramsSerializer: { indexes: null },
    })
    expect(
      within(
        screen.getByLabelText('Performance metrics for the last 24 hours')
      ).getAllByText(/^—/)
    ).toHaveLength(4)
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: 'Details' }))
    expect(onModelClick).toHaveBeenCalledWith('example-model')
  })

  it('replaces repeated artwork without taking a later card’s cover', () => {
    vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { models: [] } },
    })
    const models = [
      'haya-seedance-2-5-m',
      'ty-seedance-2-5-1050-flxd',
      'mj-seedance-2-5-fd-720p',
    ].map((model_name, index) =>
      pricingModel({ id: index + 1, model_name, square_category: 'video' })
    )
    const covers = assignCatalogCovers(models)
    render(
      <QueryClientProvider client={queryClient}>
        <ModelCardGrid
          presentation='directory'
          models={models}
          covers={covers}
          onModelClick={vi.fn()}
        />
      </QueryClientProvider>
    )
    const grid = screen.getByRole('list', { name: 'Models' })
    const images = [...grid.querySelectorAll('[data-slot="model-cover"] img')]
    expect(images).toHaveLength(3)
    const sources = images.map((image) => image.getAttribute('src'))
    expect(new Set(sources).size).toBe(3)
    expect(sources[0]).toBe(covers.get(models[0].model_name))
    expect(sources[2]).toBe(covers.get(models[2].model_name))
  })

  it('paginates the model cards and disables navigation at both boundaries', async () => {
    const request = vi.spyOn(api, 'get').mockResolvedValue({
      data: { success: true, data: { models: [] } },
    })
    const models = Array.from({ length: 21 }, (_, index) =>
      pricingModel({ id: index + 1, model_name: `model-${index + 1}` })
    )
    const user = userEvent.setup()
    render(
      <QueryClientProvider client={queryClient}>
        <ModelCardGrid models={models} onModelClick={vi.fn()} />
      </QueryClientProvider>
    )
    expect(screen.getByRole('button', { name: 'Previous page' })).toBeDisabled()
    expect(screen.getAllByRole('heading', { level: 3 })).toHaveLength(20)
    expect(request).toHaveBeenCalledWith('/api/perf-metrics/summary', {
      params: {
        hours: 24,
        recent_model: models.slice(0, 20).map((model) => model.model_name),
      },
      paramsSerializer: { indexes: null },
    })
    await user.click(screen.getByRole('button', { name: 'Next page' }))
    expect(screen.getAllByRole('heading', { level: 3 })).toHaveLength(1)
    expect(screen.getByRole('heading', { name: 'model-21' })).toBeVisible()
    expect(request).toHaveBeenLastCalledWith('/api/perf-metrics/summary', {
      params: { hours: 24, recent_model: ['model-21'] },
      paramsSerializer: { indexes: null },
    })
    expect(screen.getByRole('button', { name: 'Next page' })).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Previous page' }))
    expect(screen.getByRole('heading', { name: 'model-1' })).toBeVisible()
  })

  it('switches the card grid to three columns at the xl breakpoint instead of 2xl', () => {
    queryClient.setQueryData(['perf-metrics-summary', 24, ['example-model']], {
      success: true,
      data: { models: [] },
    })
    render(
      <QueryClientProvider client={queryClient}>
        <ModelCardGrid models={[pricingModel()]} onModelClick={vi.fn()} />
      </QueryClientProvider>
    )
    const grid = screen
      .getByRole('heading', { name: 'example-model' })
      .closest('.grid')
    expect(grid).toHaveClass('xl:grid-cols-3')
    expect(grid).not.toHaveClass('2xl:grid-cols-3')
    expect(grid).not.toHaveClass('min-[1440px]:grid-cols-3')
  })

  it('shows only sparse valid samples in chronological order without placeholder bars', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-07T12:00:00.000Z'))
    const currentHourStart = Math.floor(Date.now() / 1000 / 3600) * 3600

    render(
      <ModelCard
        model={pricingModel()}
        onClick={vi.fn()}
        perf={{
          avg_latency_ms: 1200,
          avg_tps: 42,
          success_rate: 100,
          window_start: currentHourStart - 23 * 3600,
          recent_success_series: [
            { ts: currentHourStart, success_rate: 100 },
            { ts: currentHourStart - 5 * 3600, success_rate: 80 },
            { ts: currentHourStart - 3 * 3600, success_rate: 0 },
          ],
        }}
      />
    )

    const spans = [
      ...screen.getByRole('img', {
        name: 'Success rate',
      }).children,
    ]
    expect(spans).toHaveLength(3)
    expect(spans[0]).toHaveClass('bg-amber-500')
    expect(spans[1]).toHaveClass('bg-red-500')
    expect(spans[2]).toHaveClass('bg-emerald-500')
    vi.useRealTimers()
  })

  it('skips invalid samples without inserting gaps or dropping a zero success rate', () => {
    const windowStart = Date.parse('2026-09-07T00:00:00.000Z') / 1000
    render(
      <ModelCard
        model={pricingModel()}
        onClick={vi.fn()}
        perf={{
          avg_latency_ms: 1200,
          avg_tps: 42,
          success_rate: 50,
          window_start: windowStart,
          recent_success_series: [
            { ts: windowStart, success_rate: 100 },
            { ts: windowStart + 3600, success_rate: Number.NaN },
            { ts: windowStart + 2 * 3600, success_rate: -1 },
            { ts: windowStart + 3 * 3600, success_rate: 101 },
            {
              ts: windowStart + 4 * 3600,
              success_rate: Number.POSITIVE_INFINITY,
            },
            { ts: windowStart + 5 * 3600, success_rate: 0 },
            { ts: Number.NaN, success_rate: 100 },
          ],
        }}
      />
    )

    const spans = [
      ...screen.getByRole('img', {
        name: 'Success rate',
      }).children,
    ]
    expect(spans).toHaveLength(2)
    expect(spans[0]).toHaveClass('bg-emerald-500')
    expect(spans[1]).toHaveClass('bg-red-500')
  })

  it('shows the latest 24 valid samples without requiring hourly window metadata or mutating the series', () => {
    const recentSeries = [
      { ts: 0, success_rate: 0 },
      ...Array.from({ length: 24 }, (_, index) => ({
        ts: (index + 1) * 7200,
        success_rate: 100,
      })),
      { ts: 25 * 7200, success_rate: Number.NaN },
    ].reverse()
    const originalSeries = [...recentSeries]
    render(
      <ModelCard
        model={pricingModel()}
        onClick={vi.fn()}
        perf={{
          avg_latency_ms: 1200,
          avg_tps: 42,
          success_rate: 100,
          recent_success_series: recentSeries,
        }}
      />
    )

    const spans = [
      ...screen.getByRole('img', {
        name: 'Success rate',
      }).children,
    ]
    expect(spans).toHaveLength(24)
    spans.forEach((sample) => {
      expect(sample).toHaveClass('bg-emerald-500')
    })
    expect(recentSeries).toEqual(originalSeries)
  })

  it.each([undefined, [], [{ ts: 3600, success_rate: Number.NaN }]])(
    'shows an empty state without bars when the series has no valid samples: %j',
    (series) => {
      render(
        <ModelCard
          model={pricingModel()}
          onClick={vi.fn()}
          perf={{
            avg_latency_ms: 1200,
            avg_tps: 42,
            success_rate: 100,
            recent_success_series: series,
          }}
        />
      )

      const statusStrip = screen.getByRole('img', {
        name: 'No performance data available',
      })
      expect(statusStrip).toHaveTextContent('—')
      expect(statusStrip.querySelectorAll('[aria-hidden]')).toHaveLength(0)
    }
  )

  it('shows a single server sample without padding when the browser clock is a day ahead', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-07T12:37:00.000Z'))
    const currentHourStart = Math.floor(Date.now() / 1000 / 3600) * 3600

    render(
      <ModelCard
        model={pricingModel()}
        onClick={vi.fn()}
        perf={{
          avg_latency_ms: 1200,
          avg_tps: 42,
          success_rate: 80,
          window_start: currentHourStart - 47 * 3600,
          recent_success_series: [
            { ts: currentHourStart - 29 * 3600, success_rate: 80 },
          ],
        }}
      />
    )

    const spans = [
      ...screen.getByRole('img', {
        name: 'Success rate',
      }).children,
    ]
    expect(spans).toHaveLength(1)
    expect(spans[0]).toHaveClass('bg-amber-500')
    vi.useRealTimers()
  })
})
