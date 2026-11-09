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
import { act, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { useState } from 'react'
import { afterEach, expect, it, vi } from 'vitest'

import {
  evaluateTaskVisualConfig,
  tryParseTaskVisualConfig,
} from '@/features/pricing/lib/task-expr'
import type { BillingUsageSchema } from '@/features/pricing/types'

import { TaskPricingBranchDialog } from '../task-pricing-branch-dialog'
import { TaskUsagePricingEditor } from '../task-usage-pricing-editor'

function renderPricing(unitLabel?: Record<string, string>) {
  const onBillingExprChange = vi.fn()
  render(
    <TaskUsagePricingEditor
      billingExpr='tier("music", 1 + u("clips") * 11)'
      requestRuleExpr=''
      usageSchema={{
        action: {
          enum: ['music', 'lyrics'],
          enumLabels: {
            music: { en: 'Generate songs', zh: '生成歌曲' },
            lyrics: { en: 'Generate lyrics', zh: '生成歌词' },
          },
          description: { en: 'Generate songs or lyrics', zh: '生成歌曲或歌词' },
        },
        clips: {
          type: 'number',
          unit: 'count',
          unitLabel,
          description: { en: 'Song generation unit price', zh: '生成歌曲单价' },
        },
      }}
      onBillingExprChange={onBillingExprChange}
      onRequestRuleExprChange={vi.fn()}
    />
  )
  return onBillingExprChange
}

afterEach(async () => {
  await act(() => i18next.changeLanguage('en'))
})

const videoBranchSchema: BillingUsageSchema = {
  resolution: {
    enum: ['720p'],
    allowCustom: true,
    description: 'Resolution',
  },
  duration_tier: {
    enum: ['10', '20'],
    allowCustom: true,
    description: 'Duration',
  },
  video_input: {
    enum: ['none', 'video'],
    enumLabels: {
      none: { en: 'No reference video', zh: '无参考视频' },
      video: { en: 'With reference video', zh: '有参考视频' },
    },
    description: { en: 'Reference video input', zh: '参考视频输入' },
  },
  requests: { type: 'number', unit: 'count', description: 'Video price' },
}

it.each([
  { name: 'resolution only', combination: { resolution: '720p' } },
  { name: 'duration only', combination: { duration_tier: '10' } },
  {
    name: 'resolution and duration',
    combination: { resolution: '720p', duration_tier: '10' },
  },
  {
    name: 'resolution and reference video',
    combination: { resolution: '720p', video_input: 'video' },
  },
  {
    name: 'duration and reference video',
    combination: { duration_tier: '20', video_input: 'video' },
  },
  {
    name: 'resolution, duration and reference video',
    combination: {
      resolution: '1080p',
      duration_tier: '20',
      video_input: 'video',
    },
  },
])(
  'adds a branch with $name without requiring other conditions',
  async ({ combination }) => {
    const onAdd = vi.fn()
    const user = userEvent.setup()
    render(
      <TaskPricingBranchDialog
        open
        onOpenChange={vi.fn()}
        schema={videoBranchSchema}
        rows={[]}
        onAdd={onAdd}
      />
    )
    for (const [field, definition] of Object.entries(videoBranchSchema)) {
      if (!definition.enum) continue
      const label =
        typeof definition.description === 'string'
          ? definition.description
          : (definition.description?.en ?? field)
      const checkbox = screen.getByRole('checkbox', { name: label })
      const value = combination[field as keyof typeof combination]
      if (
        (value !== undefined) !==
        (checkbox.getAttribute('aria-checked') === 'true')
      ) {
        await user.click(checkbox)
      }
      if (value === undefined) continue
      if (definition.allowCustom) {
        await user.type(
          screen.getByRole('textbox', { name: `${label} value` }),
          value
        )
      } else {
        await user.click(
          screen.getByRole('combobox', { name: `${label} value` })
        )
        await user.click(
          screen.getByRole('option', { name: 'With reference video' })
        )
      }
    }
    await user.click(screen.getByRole('button', { name: 'Add tier' }))
    expect(onAdd).toHaveBeenCalledExactlyOnceWith(combination)
  }
)

it('removes unchecked values and rejects empty or whitespace-only conditions', async () => {
  const onAdd = vi.fn()
  const user = userEvent.setup()
  render(
    <TaskPricingBranchDialog
      open
      onOpenChange={vi.fn()}
      schema={videoBranchSchema}
      rows={[]}
      onAdd={onAdd}
    />
  )
  await user.type(
    screen.getByRole('textbox', { name: 'Resolution value' }),
    '720p'
  )
  await user.click(screen.getByRole('checkbox', { name: 'Resolution' }))
  expect(
    screen.queryByRole('textbox', { name: 'Resolution value' })
  ).not.toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: 'Add tier' }))
  expect(screen.getByRole('alert')).toHaveTextContent(
    'Select at least one condition.'
  )
  expect(onAdd).not.toHaveBeenCalled()

  screen.getByRole('checkbox', { name: 'Duration' }).focus()
  await user.keyboard('[Space]')
  expect(screen.getByRole('checkbox', { name: 'Duration' })).toBeChecked()
  const duration = screen.getByRole('textbox', { name: 'Duration value' })
  await user.type(duration, '   ')
  await user.click(screen.getByRole('button', { name: 'Add tier' }))
  expect(screen.getByRole('alert')).toHaveTextContent('Required')
  expect(duration).toHaveAttribute('aria-invalid', 'true')
  expect(onAdd).not.toHaveBeenCalled()
  await user.clear(duration)
  await user.type(duration, '20')
  await user.click(screen.getByRole('button', { name: 'Add tier' }))
  expect(onAdd).toHaveBeenCalledExactlyOnceWith({ duration_tier: '20' })
})

it.each<{
  existing: Record<string, string>
  duration: string | undefined
  duplicate: boolean
}>([
  { existing: { resolution: '720p' }, duration: undefined, duplicate: true },
  { existing: { resolution: '720p' }, duration: '10', duplicate: false },
  {
    existing: { resolution: '720p', duration_tier: '10' },
    duration: undefined,
    duplicate: false,
  },
  {
    existing: { duration_tier: '10', resolution: '720p' },
    duration: '10',
    duplicate: true,
  },
])(
  'compares exact condition sets for duplicates: $existing, duration $duration',
  async ({ existing, duration, duplicate }) => {
    const onAdd = vi.fn()
    const user = userEvent.setup()
    render(
      <TaskPricingBranchDialog
        open
        onOpenChange={vi.fn()}
        schema={videoBranchSchema}
        rows={[
          { combination: existing, constant: 0, unitPrices: { requests: 1 } },
        ]}
        onAdd={onAdd}
      />
    )
    await user.type(
      screen.getByRole('textbox', { name: 'Resolution value' }),
      '720p'
    )
    if (duration) {
      await user.click(screen.getByRole('checkbox', { name: 'Duration' }))
      await user.type(
        screen.getByRole('textbox', { name: 'Duration value' }),
        duration
      )
    }
    await user.click(screen.getByRole('button', { name: 'Add tier' }))
    if (duplicate) {
      expect(screen.getByRole('alert')).toHaveTextContent(
        'A tier with these conditions already exists.'
      )
      expect(onAdd).not.toHaveBeenCalled()
    } else {
      expect(onAdd).toHaveBeenCalledExactlyOnceWith(
        duration
          ? { resolution: '720p', duration_tier: duration }
          : { resolution: '720p' }
      )
    }
  }
)

it('localizes reference video choices while submitting raw enum values', async () => {
  await act(() => i18next.changeLanguage('zhCN'))
  const onAdd = vi.fn()
  const user = userEvent.setup()
  render(
    <TaskPricingBranchDialog
      open
      onOpenChange={vi.fn()}
      schema={videoBranchSchema}
      rows={[]}
      onAdd={onAdd}
    />
  )
  await user.click(screen.getByRole('checkbox', { name: 'Resolution' }))
  await user.click(screen.getByRole('checkbox', { name: '参考视频输入' }))
  await user.click(screen.getByRole('combobox'))
  expect(screen.getByRole('option', { name: '无参考视频' })).toBeVisible()
  await user.click(screen.getByRole('option', { name: '有参考视频' }))
  await act(() => i18next.changeLanguage('en'))
  expect(
    screen.getByRole('combobox', { name: 'Reference video input value' })
  ).toHaveValue('With reference video')
  await user.click(screen.getByRole('button', { name: 'Add tier' }))
  expect(onAdd).toHaveBeenCalledExactlyOnceWith({ video_input: 'video' })
})

it.each(['720p', '1080p', '4k'])(
  'prices %s independently with and without reference video',
  async (resolution) => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    render(
      <TaskUsagePricingEditor
        billingExpr='tier("base", u("requests") * 0.5)'
        requestRuleExpr=''
        usageSchema={videoBranchSchema}
        onBillingExprChange={onChange}
        onRequestRuleExprChange={vi.fn()}
      />
    )
    for (const [videoInput, label, price] of [
      ['none', 'No reference video', '1'],
      ['video', 'With reference video', '2'],
    ]) {
      await user.click(
        screen.getByRole('button', { name: 'Add pricing branch' })
      )
      const dialog = within(screen.getByRole('dialog'))
      await user.type(
        dialog.getByRole('textbox', { name: 'Resolution value' }),
        resolution
      )
      await user.click(
        dialog.getByRole('checkbox', { name: 'Reference video input' })
      )
      if (videoInput === 'video') {
        await user.click(
          dialog.getByRole('combobox', { name: 'Reference video input value' })
        )
        await user.click(screen.getByRole('option', { name: label }))
      }
      await user.click(dialog.getByRole('button', { name: 'Add tier' }))
      const input = screen.getByRole('textbox', {
        name: `Video price: Resolution: ${resolution} · ${label}`,
      })
      await user.clear(input)
      await user.type(input, price)
    }
    const expression = onChange.mock.lastCall?.[0]
    expect(expression).toContain(`u("resolution") == "${resolution}"`)
    expect(expression).toContain('u("video_input") == "none"')
    expect(expression).toContain('u("video_input") == "video"')
    expect(expression).not.toContain('duration_tier')
    const config = tryParseTaskVisualConfig(expression, videoBranchSchema)
    expect(config).not.toBeNull()
    if (!config) return
    expect(
      evaluateTaskVisualConfig(config, {
        requests: 1,
        resolution,
        video_input: 'none',
      })?.total
    ).toBe(1)
    expect(
      evaluateTaskVisualConfig(config, {
        requests: 1,
        resolution,
        video_input: 'video',
      })?.total
    ).toBe(2)
    expect(
      evaluateTaskVisualConfig(config, {
        requests: 1,
        resolution: '8k',
        video_input: 'video',
      })?.total
    ).toBe(0.5)
  }
)

it('reuses optional price controls for a plugin without resolution tiers', async () => {
  const onChange = vi.fn()
  const user = userEvent.setup()
  render(
    <TaskUsagePricingEditor
      billingExpr='tier("base", u("jobs") * 2)'
      requestRuleExpr=''
      usageSchema={{
        jobs: { type: 'number', unit: 'count', description: 'Job price' },
        credits: {
          type: 'number',
          unit: 'credit',
          optional: true,
          description: 'Extra processing price',
        },
      }}
      onBillingExprChange={onChange}
      onRequestRuleExprChange={vi.fn()}
    />
  )
  const optional = screen.getByRole('checkbox', {
    name: 'Include Extra processing price',
  })
  const price = screen.getByRole('textbox', { name: 'credits' })
  expect(price).toBeDisabled()
  await user.click(optional)
  expect(price).toBeEnabled()
  await user.clear(price)
  await user.type(price, '0.1')
  expect(onChange.mock.lastCall?.[0]).toContain('u("credits") * 0.1')
  await user.click(optional)
  expect(onChange.mock.lastCall?.[0]).not.toContain('credits')
  expect(onChange.mock.lastCall?.[0]).toContain('u("jobs") * 2')
})

it('shows localized schema explanations in the price table and calculator', async () => {
  renderPricing()
  const table = screen.getByRole('table')
  expect(within(table).getByText('Song generation unit price')).toBeVisible()
  expect(within(table).getByText('Generate songs or lyrics')).toBeVisible()
  expect(
    screen.getByRole('spinbutton', {
      name: 'Usage · Song generation unit price',
    })
  ).toBeVisible()
  expect(
    within(table).getByText(
      'Added to the usage cost. Set to 0 for no additional charge.'
    )
  ).toBeVisible()
  await act(() => i18next.changeLanguage('zhCN'))
  expect(within(table).getByText('生成歌曲单价')).toBeVisible()
  expect(screen.getByRole('combobox', { name: '生成歌曲或歌词' })).toBeVisible()
})

it('uses count unit labels in the price matrix and calculator without changing the charge', async () => {
  renderPricing({ en: 'song', zh: '首' })
  expect(within(screen.getByRole('table')).getByText('$/song')).toBeVisible()
  expect(
    screen.getByText(
      'Additional charge: $1 + Song generation unit price: 1 song × $11/song = $12'
    )
  ).toBeVisible()
  await act(() => i18next.changeLanguage('zhCN'))
  expect(within(screen.getByRole('table')).getByText('$/首')).toBeVisible()
  expect(screen.getByText('首')).toBeVisible()
  expect(screen.getByText(/1 首 × \$11\/首 = \$12/)).toBeVisible()
})

it('identifies pricing conditions and keeps the additional charge unchanged when sample usage changes', async () => {
  renderPricing()
  expect(
    screen.getByText('Current pricing conditions: Generate songs')
  ).toBeVisible()
  expect(
    screen.getByText(
      'Additional charge: $1 + Song generation unit price: 1 unit × $11/unit = $12'
    )
  ).toBeVisible()
  const user = userEvent.setup()
  const quantity = screen.getByRole('spinbutton', {
    name: 'Usage · Song generation unit price',
  })
  await user.clear(quantity)
  await user.type(quantity, '2')
  expect(
    screen.getByText(
      'Additional charge: $1 + Song generation unit price: 2 unit × $11/unit = $23'
    )
  ).toBeVisible()
})

it('shows localized enum choices while preserving raw values in generated billing expressions', async () => {
  const onChange = renderPricing()
  const user = userEvent.setup()
  await user.click(
    screen.getByRole('combobox', { name: 'Generate songs or lyrics' })
  )
  await user.click(screen.getByRole('option', { name: 'Generate lyrics' }))
  expect(
    screen.getByText('Current pricing conditions: Generate lyrics')
  ).toBeVisible()
  const price = screen.getByRole('textbox', {
    name: 'Song generation unit price: Generate lyrics',
  })
  await user.clear(price)
  await user.type(price, '3')
  const expression = onChange.mock.lastCall?.[0]
  expect(expression).toContain('u("action") == "music"')
  expect(expression).toContain('tier("lyrics"')
  expect(expression).not.toContain('Generate lyrics')
})

function TaskPricingDraft(props: {
  expression: string
  onChange: (next: string) => void
}) {
  const [billingExpr, setBillingExpr] = useState(props.expression)
  const [requestRuleExpr, setRequestRuleExpr] = useState(
    '(header("x-priority") == "high" ? 2 : 1)'
  )
  return (
    <TaskUsagePricingEditor
      billingExpr={billingExpr}
      requestRuleExpr={requestRuleExpr}
      usageSchema={{
        seconds: {
          type: 'number',
          unit: 'second',
          description: 'Seconds price',
        },
        mode: { enum: ['std', 'pro'] },
      }}
      onBillingExprChange={(next) => {
        setBillingExpr(next)
        props.onChange(next)
      }}
      onRequestRuleExprChange={setRequestRuleExpr}
    />
  )
}

it('lets users cancel or discard an unsupported expression and its request rules', async () => {
  const user = userEvent.setup()
  const onChange = vi.fn()
  render(
    <TaskPricingDraft
      expression='tier("custom", u("seconds") * u("seconds"))'
      onChange={onChange}
    />
  )
  const original = screen.getByRole('textbox', { name: 'Billing expression' })
  const originalValue = (original as HTMLTextAreaElement).value
  await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
  await user.click(screen.getByRole('option', { name: 'Visual editor' }))
  let dialog = screen.getByRole('alertdialog')
  expect(dialog).toHaveTextContent('resets all prices to zero')
  expect(onChange).not.toHaveBeenCalled()
  await user.click(within(dialog).getByRole('button', { name: 'Cancel' }))
  expect(original).toHaveValue(originalValue)
  expect(onChange).not.toHaveBeenCalled()

  await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
  await user.click(screen.getByRole('option', { name: 'Visual editor' }))
  dialog = screen.getByRole('alertdialog')
  await user.click(
    within(dialog).getByRole('button', { name: 'Switch to visual editor' })
  )
  expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
  expect(screen.getByRole('table')).toBeVisible()
  expect(onChange).toHaveBeenLastCalledWith('tier("base", u("seconds") * 0)')
  for (const price of within(screen.getByRole('table')).getAllByRole(
    'textbox'
  )) {
    expect(price).toHaveValue('0')
  }
  await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
  await user.click(screen.getByRole('option', { name: 'Expression editor' }))
  expect(
    screen.getByRole('textbox', { name: 'Billing expression' })
  ).toHaveValue('tier("base", u("seconds") * 0)')
})

const retiredBranchExpression =
  'u("mode") == "ultra" ? tier("ultra", u("seconds") * 1.2) : u("mode") == "pro" ? tier("pro", u("seconds") * 0.8) : tier("std", u("seconds") * 0.4)'

it('opens an expression with a retired enum branch in the visual editor and keeps the remaining prices', () => {
  const onChange = vi.fn()
  render(
    <TaskPricingDraft
      expression={retiredBranchExpression}
      onChange={onChange}
    />
  )
  expect(screen.getByRole('table')).toBeVisible()
  expect(
    screen.getByRole('textbox', { name: 'Seconds price: mode: std' })
  ).toHaveValue('0.4')
  expect(
    screen.getByRole('textbox', { name: 'Seconds price: mode: pro' })
  ).toHaveValue('0.8')
  expect(
    screen.queryByText(
      'This expression cannot be represented by the price table',
      {
        exact: false,
      }
    )
  ).not.toBeInTheDocument()
  expect(onChange).not.toHaveBeenCalled()
})

it('regenerates an edited expression without its retired enum branch', async () => {
  const user = userEvent.setup()
  const onChange = vi.fn()
  render(
    <TaskPricingDraft
      expression={retiredBranchExpression}
      onChange={onChange}
    />
  )
  const price = screen.getByRole('textbox', {
    name: 'Seconds price: mode: std',
  })
  await user.clear(price)
  await user.type(price, '2')
  expect(onChange).toHaveBeenLastCalledWith(
    'u("mode") == "std" ? tier("std", u("seconds") * 2) : tier("pro", u("seconds") * 0.8)'
  )
})

it('confirms regeneration of supported expressions and preserves their prices and request rules', async () => {
  const user = userEvent.setup()
  const onChange = vi.fn()
  const expression =
    'u("mode") == "std" ? tier("old-standard", u("seconds") * 0.4) : tier("old-pro", u("seconds") * 0.8)'
  render(<TaskPricingDraft expression={expression} onChange={onChange} />)
  await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
  await user.click(screen.getByRole('option', { name: 'Expression editor' }))
  expect(
    (
      screen.getByRole('textbox', {
        name: 'Billing expression',
      }) as HTMLTextAreaElement
    ).value
  ).toContain(expression)
  expect(onChange).not.toHaveBeenCalled()
  await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
  await user.click(screen.getByRole('option', { name: 'Visual editor' }))
  const dialog = screen.getByRole('alertdialog')
  expect(dialog).toHaveTextContent(
    'replaces its original formatting and tier names'
  )
  expect(onChange).not.toHaveBeenCalled()
  await user.click(
    within(dialog).getByRole('button', { name: 'Switch to visual editor' })
  )
  expect(onChange).toHaveBeenLastCalledWith(
    'u("mode") == "std" ? tier("std", u("seconds") * 0.4) : tier("pro", u("seconds") * 0.8)'
  )
  expect(
    screen.getByRole('textbox', { name: 'Seconds price: mode: std' })
  ).toHaveValue('0.4')
  expect(
    screen.getByRole('textbox', { name: 'Seconds price: mode: pro' })
  ).toHaveValue('0.8')
  await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
  await user.click(screen.getByRole('option', { name: 'Expression editor' }))
  expect(
    (
      screen.getByRole('textbox', {
        name: 'Billing expression',
      }) as HTMLTextAreaElement
    ).value
  ).toContain('header("x-priority")')
})

it.each(['image', 'video'] as const)(
  'limits pricing fields to the resolved %s model schema',
  (kind) => {
    const usageSchema = {
      image: {
        image_count: {
          type: 'number' as const,
          unit: 'count' as const,
          unitLabel: { en: 'image', zh: '张' },
          description: { en: 'Image quantity' },
        },
      },
      video: {
        seconds: {
          type: 'number' as const,
          unit: 'second' as const,
          description: { en: 'Video duration' },
        },
        resolution: {
          enum: ['720P', '1080P'],
          description: { en: 'Resolution' },
        },
      },
    }
    const field = kind === 'image' ? 'image_count' : 'seconds'
    render(
      <TaskUsagePricingEditor
        billingExpr={`tier("base", u("${field}") * 1)`}
        requestRuleExpr=''
        usageSchema={usageSchema[kind]}
        onBillingExprChange={vi.fn()}
        onRequestRuleExprChange={vi.fn()}
      />
    )
    const presentLabel = kind === 'image' ? 'Image quantity' : 'Video duration'
    const absentLabel = kind === 'image' ? 'Video duration' : 'Image quantity'
    expect(
      screen.getByRole('spinbutton', { name: `Usage · ${presentLabel}` })
    ).toBeVisible()
    expect(
      screen.queryByRole('spinbutton', { name: `Usage · ${absentLabel}` })
    ).not.toBeInTheDocument()
    if (kind === 'image') {
      expect(screen.getByText('$/image')).toBeVisible()
      expect(
        screen.queryByRole('combobox', { name: 'Resolution' })
      ).not.toBeInTheDocument()
    } else {
      expect(screen.getByRole('combobox', { name: 'Resolution' })).toBeVisible()
    }
  }
)

it('adds and removes custom resolution branches while keeping the fallback price', async () => {
  const onChange = vi.fn()
  const user = userEvent.setup()
  render(
    <TaskUsagePricingEditor
      billingExpr='tier("base", u("seconds") * 0.2)'
      requestRuleExpr=''
      usageSchema={{
        resolution: {
          enum: ['720p'],
          allowCustom: true,
          description: 'Resolution',
        },
        seconds: {
          type: 'number',
          unit: 'second',
          description: 'Output price',
        },
        reference_seconds: {
          type: 'number',
          unit: 'second',
          optional: true,
          description: 'Reference video unit price',
        },
      }}
      onBillingExprChange={onChange}
      onRequestRuleExprChange={vi.fn()}
    />
  )
  await user.click(screen.getByRole('button', { name: 'Add pricing branch' }))
  const resolution = screen.getByRole('textbox', { name: 'Resolution value' })
  await user.type(resolution, '1440p')
  await user.click(screen.getByRole('button', { name: 'Add tier' }))
  expect(screen.getByText('1440p')).toBeVisible()
  const price = screen.getByRole('textbox', {
    name: 'Output price: Resolution: 1440p',
  })
  await user.clear(price)
  await user.type(price, '0.8')
  expect(onChange.mock.lastCall?.[0]).toContain('u("resolution") == "1440p"')
  expect(onChange.mock.lastCall?.[0]).toContain('u("seconds") * 0.8')
  expect(onChange.mock.lastCall?.[0]).toContain('u("seconds") * 0.2')
  await user.click(screen.getByRole('button', { name: 'Add pricing branch' }))
  await user.type(
    screen.getByRole('textbox', { name: 'Resolution value' }),
    '1440p'
  )
  await user.click(screen.getByRole('button', { name: 'Add tier' }))
  expect(screen.getByRole('alert')).toHaveTextContent(
    'A tier with these conditions already exists.'
  )
  await user.click(screen.getByRole('button', { name: 'Cancel' }))
  await user.click(
    screen.getByRole('button', { name: 'Remove tier Resolution: 1440p' })
  )
  expect(screen.queryByText('1440p')).not.toBeInTheDocument()
  expect(onChange.mock.lastCall?.[0]).not.toContain('1440p')
  expect(onChange.mock.lastCall?.[0]).toContain('u("seconds") * 0.2')
})

it('disables reference video charges without changing output prices and restores saved custom branches', async () => {
  const onChange = vi.fn()
  const user = userEvent.setup()
  render(
    <TaskUsagePricingEditor
      billingExpr='u("resolution") == "1440p" ? tier("high", u("seconds") * 0.4 + u("reference_seconds") * 0.1) : tier("base", u("seconds") * 0.2 + u("reference_seconds") * 0.05)'
      requestRuleExpr=''
      usageSchema={{
        resolution: {
          enum: ['720p'],
          allowCustom: true,
          description: 'Resolution',
        },
        seconds: {
          type: 'number',
          unit: 'second',
          description: 'Output price',
        },
        reference_seconds: {
          type: 'number',
          unit: 'second',
          optional: true,
          description: 'Reference video unit price',
        },
      }}
      onBillingExprChange={onChange}
      onRequestRuleExprChange={vi.fn()}
    />
  )
  expect(screen.getByText('1440p')).toBeVisible()
  const toggle = screen.getByRole('checkbox', {
    name: 'Include Reference video unit price',
  })
  expect(toggle).toBeChecked()
  await user.click(toggle)
  expect(
    screen.getByRole('textbox', {
      name: 'Reference video unit price: Resolution: 1440p',
    })
  ).toBeDisabled()
  expect(onChange.mock.lastCall?.[0]).not.toContain('reference_seconds')
  expect(onChange.mock.lastCall?.[0]).toContain('u("seconds") * 0.4')
  await user.click(toggle)
  const price = screen.getByRole('textbox', {
    name: 'Reference video unit price: Resolution: 1440p',
  })
  expect(price).toBeEnabled()
  await user.clear(price)
  await user.type(price, '0.3')
  expect(onChange.mock.lastCall?.[0]).toContain('u("reference_seconds") * 0.3')
})
