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
import { createRef } from 'react'
import { afterEach, expect, it, vi } from 'vitest'

import type {
  ModelPricingConfig,
  ModelPricingPluginVariant,
} from '@/features/model-pricing/api'
import { USD_PRICING_CURRENCY } from '@/features/model-pricing/currency'
import { ModelPricingPanel } from '@/features/model-pricing/model-pricing-panel'
import {
  pricingFromDraft,
  pricingOptions,
} from '@/features/model-pricing/pricing'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'
import { usePricingPreferencesStore } from '@/stores/pricing-preferences-store'

import {
  ModelPricingEditorPanel,
  type ModelPricingEditorPanelHandle,
  type ModelRatioData,
} from '../model-pricing-sheet'
import {
  ModelRatioVisualEditor,
  type ModelRatioVisualEditorHandle,
} from '../model-ratio-visual-editor'
import { TaskPluginPricingEditor } from '../task-plugin-pricing-editor'

let client: QueryClient | undefined

afterEach(() => {
  client?.clear()
  useAuthStore.getState().auth.setUser(null)
  localStorage.clear()
  vi.restoreAllMocks()
})

it('keeps provider drafts across tabs and saves nested expressions with the model version', async () => {
  useAuthStore
    .getState()
    .auth.setUser({ id: 1, username: 'administrator', role: 100 })
  usePricingPreferencesStore.setState({ currency: 'USD' })
  const expression = 'tier("base", u("seconds") * 0.4)'
  const values = {
    'billing_setting.billing_mode': 'tiered_expr',
    'billing_setting.billing_expr': expression,
  }
  const snapshot: ModelPricingConfig = {
    entries: [
      {
        model_name: 'shared',
        version: 'version-with-providers',
        configured: values,
        effective: values,
        usage_schema: {
          seconds: {
            type: 'number',
            unit: 'second',
            description: 'Video unit price',
          },
        },
        plugin_variants: [
          {
            plugin_key: 'alpha',
            plugin_name: 'Alpha',
            usage_schema: {
              seconds: {
                type: 'number',
                unit: 'second',
                description: 'Video unit price',
              },
            },
            configured: '',
            effective: expression,
            compatible: true,
          },
          {
            plugin_key: 'beta',
            plugin_name: 'Beta',
            usage_schema: {
              credits: {
                type: 'number',
                unit: 'credit',
                description: 'Credit unit price',
              },
            },
            configured: '',
            effective: expression,
            compatible: false,
          },
        ],
      },
    ],
    options: pricingOptions({}),
    empty_version: 'empty',
  }
  vi.spyOn(api, 'get').mockImplementation(async (url) => {
    if (url === '/api/option/model_pricing') {
      return { data: { success: true, data: snapshot } }
    }
    if (url === '/api/pricing') {
      return { data: { success: true, data: [], vendors: [] } }
    }
    return { data: { success: true, data: {} } }
  })
  let finishSave: (() => void) | undefined
  const save = vi.spyOn(api, 'patch').mockImplementation(
    () =>
      new Promise((resolve) => {
        finishSave = () => resolve({ data: { success: true } })
      })
  )
  const dirty = vi.fn()
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <ModelPricingPanel modelName='shared' onDirtyChange={dirty} />
    </QueryClientProvider>
  )
  const user = userEvent.setup()
  const defaultTab = await screen.findByRole('tab', { name: 'Default' })
  expect(defaultTab).toHaveAttribute('aria-selected', 'true')
  expect(
    await screen.findByText(
      'The model-level expression cannot be evaluated by: Beta'
    )
  ).toBeVisible()
  await user.click(screen.getByRole('button', { name: 'Beta' }))
  const betaPanel = screen.getByRole('tabpanel', { name: 'Beta' })
  expect(
    within(betaPanel).getByText('Not configured for this provider')
  ).toBeVisible()
  const separate = within(betaPanel).getByRole('switch', {
    name: 'Set separately for this provider',
  })
  expect(separate).not.toBeChecked()
  await user.click(separate)
  expect(separate).toBeChecked()
  await user.click(separate)
  expect(separate).not.toBeChecked()
  expect(
    within(betaPanel).getByText('Use model-level expression')
  ).toBeVisible()
  expect(
    within(betaPanel).queryByRole('textbox', { name: 'credits' })
  ).not.toBeInTheDocument()
  await user.click(separate)
  const price = within(betaPanel).getByRole('textbox', { name: 'credits' })
  await user.clear(price)
  await user.type(price, '1.5')
  expect(dirty).toHaveBeenLastCalledWith(true)
  await user.click(defaultTab)
  expect(
    screen.queryByText(
      'The model-level expression cannot be evaluated by: Beta'
    )
  ).not.toBeInTheDocument()
  await user.click(screen.getByRole('tab', { name: 'Beta' }))
  expect(price).toHaveValue('1.5')
  await user.click(screen.getByRole('tab', { name: 'Ordinary OpenAI' }))
  await user.click(
    screen.getByRole('button', { name: 'Configure ordinary OpenAI pricing' })
  )
  const ordinaryPrice = screen.getByRole('textbox', { name: 'Input price' })
  await user.clear(ordinaryPrice)
  await user.type(ordinaryPrice, '2')
  const saveButton = screen.getByRole('button', { name: 'Save model prices' })
  await user.click(saveButton)
  await waitFor(() => expect(save).toHaveBeenCalledOnce())
  expect(saveButton).toBeDisabled()
  const changes = save.mock.calls[0][1] as {
    changes: { expected_version: string; pricing: Record<string, unknown> }[]
  }
  expect(changes.changes[0].expected_version).toBe('version-with-providers')
  expect(changes.changes[0].pricing['billing_setting.billing_expr']).toBe(
    'tier("base", p * 2 + c * 0)'
  )
  expect(
    changes.changes[0].pricing['billing_setting.plugin_billing_expr']
  ).toEqual({ alpha: expression, beta: 'tier("base", u("credits") * 1.5)' })
  await act(async () => {
    finishSave?.()
  })
  await waitFor(() => expect(saveButton).toBeEnabled())
})

it('keeps long provider names accessible while allowing the tabs to wrap', () => {
  const longName =
    'Provider with a long international service name and a regional deployment designation'
  render(
    <TaskPluginPricingEditor
      variants={[
        {
          plugin_key: 'alpha',
          plugin_name: 'Alpha',
          usage_schema: {},
          configured: '',
          effective: '1',
          compatible: true,
        },
        {
          plugin_key: 'beta',
          plugin_name: longName,
          usage_schema: {},
          configured: '',
          effective: '1',
          compatible: true,
        },
      ]}
      expressions={{}}
      onChange={vi.fn()}
      modelExpression='1'
      modelBillingMode='tiered_expr'
      currency={USD_PRICING_CURRENCY}
    >
      <span>Default editor</span>
    </TaskPluginPricingEditor>
  )
  expect(screen.getByRole('tablist', { name: 'Provider' })).toHaveClass(
    'flex-wrap'
  )
  expect(screen.getByRole('tab', { name: longName })).toHaveClass(
    'max-w-full',
    'min-w-0'
  )
  expect(screen.getByText(longName)).toHaveAttribute('title', longName)
})

it('lets an administrator remove an unavailable provider’s saved price', async () => {
  const expression = 'u("images")'
  const { ref } = renderEditor(
    {
      name: 'shared',
      billingMode: 'tiered_expr',
      billingExpr: expression,
      pluginBillingExpr: { retired: expression },
    },
    [
      {
        plugin_key: 'retired',
        plugin_name: 'Retired',
        stale: true,
        usage_schema: {},
        configured: expression,
        effective: '',
        compatible: false,
      },
    ]
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole('tab', { name: 'Retired' }))
  expect(
    screen.getByText('This provider is unavailable for this model.')
  ).toBeVisible()
  await user.click(screen.getByRole('button', { name: 'Remove saved price' }))
  expect(await ref.current?.commitDraft()).toMatchObject({
    pluginBillingExpr: {},
    billingExpr: expression,
  })
  expect(screen.getByRole('tab', { name: 'Default' })).toHaveAttribute(
    'aria-selected',
    'true'
  )
})

it('shows pending server validation for an expression the browser cannot check', async () => {
  render(
    <TaskPluginPricingEditor
      variants={[
        {
          plugin_key: 'alpha',
          plugin_name: 'Alpha',
          usage_schema: {},
          configured: '',
          effective: '1',
          compatible: true,
        },
      ]}
      expressions={{}}
      onChange={vi.fn()}
      modelExpression='futureFunction()'
      modelBillingMode='tiered_expr'
      currency={USD_PRICING_CURRENCY}
    >
      <span>Default editor</span>
    </TaskPluginPricingEditor>
  )
  await userEvent.setup().click(screen.getByRole('tab', { name: 'Alpha' }))
  expect(
    screen.getByText('Expression compatibility will be checked when saving.')
  ).toBeVisible()
})

function renderEditor(
  editData: ModelRatioData,
  variants: ModelPricingPluginVariant[],
  dirty = vi.fn()
) {
  usePricingPreferencesStore.setState({ currency: 'USD' })
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: [], vendors: [] },
  })
  vi.spyOn(api, 'post').mockResolvedValue({
    data: { success: true, data: { effective: {} } },
  })
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const ref = createRef<ModelPricingEditorPanelHandle>()
  render(
    <QueryClientProvider client={client}>
      <ModelPricingEditorPanel
        ref={ref}
        embedded
        editData={editData}
        pluginVariants={variants}
        onDirtyChange={dirty}
      />
    </QueryClientProvider>
  )
  return { ref, dirty }
}

it('shows and removes a single provider override while retaining the per-call default', async () => {
  const expression = 'tier("base", u("seconds") * 0.4)'
  const { ref } = renderEditor(
    {
      name: 'shared',
      billingMode: 'per-request',
      price: '0.25',
      pluginBillingExpr: { alpha: expression },
    },
    [
      {
        plugin_key: 'alpha',
        plugin_name: 'Alpha',
        usage_schema: { seconds: { type: 'number', unit: 'second' } },
        configured: expression,
        effective: expression,
        compatible: true,
      },
    ]
  )
  expect(
    screen.getByRole('tab', { name: 'Per-request (deprecated)' })
  ).toHaveAttribute('aria-selected', 'true')
  const user = userEvent.setup()
  await user.click(screen.getByRole('tab', { name: 'Alpha' }))
  await user.click(
    screen.getByRole('switch', { name: 'Set separately for this provider' })
  )
  expect(screen.getByText('Use model-level pricing')).toBeVisible()
  expect(await ref.current?.commitDraft()).toMatchObject({
    billingMode: 'per-request',
    price: '0.25',
    pluginBillingExpr: {},
  })
})

it('clears dirty state after restoring an override in a different key order', async () => {
  const expression = 'tier("base", u("seconds") * 0.4)'
  const { dirty } = renderEditor(
    {
      name: 'shared',
      billingMode: 'tiered_expr',
      billingExpr: expression,
      pluginBillingExpr: { alpha: expression, beta: expression },
    },
    ['alpha', 'beta'].map((key) => ({
      plugin_key: key,
      plugin_name: key,
      usage_schema: { seconds: { type: 'number', unit: 'second' } },
      configured: expression,
      effective: expression,
      compatible: true,
    }))
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole('tab', { name: 'alpha' }))
  const toggle = screen.getByRole('switch', {
    name: 'Set separately for this provider',
  })
  await user.click(toggle)
  expect(dirty).toHaveBeenLastCalledWith(true)
  await user.click(toggle)
  expect(dirty).toHaveBeenLastCalledWith(false)
})

it('opens task expression pricing when only provider metadata supplies a schema', async () => {
  renderEditor(
    { name: 'shared', billingMode: 'per-token' },
    ['alpha', 'beta'].map((key) => ({
      plugin_key: key,
      plugin_name: key,
      usage_schema: { seconds: { type: 'number', unit: 'second' } },
      configured: '',
      effective: '',
      compatible: false,
    }))
  )
  await waitFor(() =>
    expect(screen.getByRole('tab', { name: 'Expression' })).toHaveAttribute(
      'aria-selected',
      'true'
    )
  )
  expect(screen.getByRole('textbox', { name: 'seconds' })).toBeVisible()
})

it('inherits each plugin default without saving a model price when editing a provider', async () => {
  const alphaDefault = 'tier("base", u("seconds") * 0.4)'
  const betaDefault = 'tier("base", u("seconds") * 0.7)'
  const { ref, dirty } = renderEditor(
    { name: 'shared', billingMode: 'per-token' },
    [
      ['alpha', alphaDefault],
      ['beta', betaDefault],
    ].map(([key, expression]) => ({
      plugin_key: key,
      plugin_name: key,
      usage_schema: { seconds: { type: 'number', unit: 'second' } },
      configured: '',
      default_expr: expression,
      effective: expression,
      compatible: true,
    }))
  )
  expect(screen.getByRole('textbox', { name: 'seconds' })).toHaveValue('0.4')
  expect(
    within(screen.getByRole('tabpanel', { name: 'Default' })).getByText(
      'Using plugin default pricing'
    )
  ).toBeVisible()
  expect(dirty).toHaveBeenLastCalledWith(false)
  const unchanged = await ref.current?.commitDraft()
  if (!unchanged) throw new Error('Expected a valid inherited pricing draft')
  expect(pricingFromDraft(unchanged)).toEqual({
    'billing_setting.plugin_billing_expr': {},
  })
  const user = userEvent.setup()
  await user.click(screen.getByRole('tab', { name: 'beta' }))
  const provider = screen.getByRole('tabpanel', { name: 'beta' })
  expect(
    within(provider).getByText('Using plugin default pricing')
  ).toBeVisible()
  expect(within(provider).getByText(betaDefault)).toBeVisible()
  const toggle = within(provider).getByRole('switch', {
    name: 'Set separately for this provider',
  })
  await user.click(toggle)
  expect(
    within(provider).getByRole('textbox', { name: 'seconds' })
  ).toHaveValue('0.7')
  const draft = await ref.current?.commitDraft()
  if (!draft) throw new Error('Expected a valid provider pricing draft')
  expect(pricingFromDraft(draft)).toEqual({
    'billing_setting.plugin_billing_expr': { beta: betaDefault },
  })
  await user.click(toggle)
  expect(within(provider).getByText(betaDefault)).toBeVisible()
  expect(dirty).toHaveBeenLastCalledWith(false)
  const restored = await ref.current?.commitDraft()
  if (!restored) throw new Error('Expected a valid inherited pricing draft')
  expect(pricingFromDraft(restored)).toEqual({
    'billing_setting.plugin_billing_expr': {},
  })
})

it('uses an edited model price after removing a provider override instead of its plugin default', async () => {
  const pluginDefault = 'tier("base", u("seconds") * 0.4)'
  const savedOverride = 'tier("base", u("seconds") * 0.8)'
  const { ref } = renderEditor(
    {
      name: 'shared',
      billingMode: 'per-token',
      pluginBillingExpr: { alpha: savedOverride },
    },
    [
      {
        plugin_key: 'alpha',
        plugin_name: 'Alpha',
        usage_schema: { seconds: { type: 'number', unit: 'second' } },
        configured: savedOverride,
        default_expr: pluginDefault,
        effective: savedOverride,
        compatible: true,
      },
    ]
  )
  const user = userEvent.setup()
  const modelPrice = screen.getByRole('textbox', { name: 'seconds' })
  expect(modelPrice).toHaveValue('0.4')
  await user.clear(modelPrice)
  await user.type(modelPrice, '0.6')
  await user.click(screen.getByRole('tab', { name: 'Alpha' }))
  const provider = screen.getByRole('tabpanel', { name: 'Alpha' })
  expect(
    within(provider).getByRole('textbox', { name: 'seconds' })
  ).toHaveValue('0.8')
  await user.click(
    within(provider).getByRole('switch', {
      name: 'Set separately for this provider',
    })
  )
  expect(within(provider).getByText('Use model-level expression')).toBeVisible()
  expect(
    within(provider).getByText('tier("base", u("seconds") * 0.6)')
  ).toBeVisible()
  expect(await ref.current?.commitDraft()).toMatchObject({
    billingMode: 'tiered_expr',
    billingExpr: 'tier("base", u("seconds") * 0.6)',
    pluginBillingExpr: {},
  })
})

it('preserves distinct plugin defaults when configuring ordinary OpenAI pricing', async () => {
  const alphaDefault = 'tier("base", u("seconds") * 0.4)'
  const betaDefault = 'tier("base", u("seconds") * 0.7)'
  const { ref } = renderEditor(
    { name: 'shared', billingMode: 'per-token' },
    [
      ['alpha', alphaDefault],
      ['beta', betaDefault],
    ].map(([key, expression]) => ({
      plugin_key: key,
      plugin_name: key,
      usage_schema: { seconds: { type: 'number', unit: 'second' } },
      configured: '',
      default_expr: expression,
      effective: expression,
      compatible: true,
    }))
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole('tab', { name: 'Ordinary OpenAI' }))
  await user.click(
    screen.getByRole('button', { name: 'Configure ordinary OpenAI pricing' })
  )
  const price = screen.getByRole('textbox', { name: 'Input price' })
  await user.clear(price)
  await user.type(price, '2.5')
  expect(await ref.current?.commitDraft()).toMatchObject({
    billingMode: 'tiered_expr',
    billingExpr: 'tier("base", p * 2.5 + c * 0)',
    pluginBillingExpr: { alpha: alphaDefault, beta: betaDefault },
  })
})

it('saves an explicit zero per-request price selected over a plugin default', async () => {
  const pluginDefault = 'tier("base", u("seconds") * 0.4)'
  const { ref } = renderEditor({ name: 'shared', billingMode: 'per-token' }, [
    {
      plugin_key: 'alpha',
      plugin_name: 'Alpha',
      usage_schema: { seconds: { type: 'number', unit: 'second' } },
      configured: '',
      default_expr: pluginDefault,
      effective: pluginDefault,
      compatible: true,
    },
  ])
  const user = userEvent.setup()
  await user.click(
    screen.getByRole('tab', { name: 'Per-request (deprecated)' })
  )
  await user.type(screen.getByRole('textbox', { name: 'Fixed price' }), '0')
  await user.click(screen.getByRole('tab', { name: 'Alpha' }))
  expect(screen.getByText('Use model-level pricing')).toBeVisible()
  expect(await ref.current?.commitDraft()).toMatchObject({
    billingMode: 'per-request',
    price: '0',
    pluginBillingExpr: {},
  })
})

it('preserves a saved zero model expression instead of replacing it with a plugin default', async () => {
  const expression = 'tier("base", p * 0 + c * 0)'
  const { ref, dirty } = renderEditor(
    { name: 'shared', billingMode: 'tiered_expr', billingExpr: expression },
    [
      {
        plugin_key: 'alpha',
        plugin_name: 'Alpha',
        usage_schema: { seconds: { type: 'number', unit: 'second' } },
        configured: '',
        default_expr: 'tier("base", u("seconds") * 0.4)',
        effective: expression,
        compatible: true,
      },
    ]
  )
  await userEvent.setup().click(screen.getByRole('tab', { name: 'Alpha' }))
  const provider = screen.getByRole('tabpanel', { name: 'Alpha' })
  expect(within(provider).getByText('Use model-level expression')).toBeVisible()
  expect(within(provider).getByText(expression)).toBeVisible()
  expect(dirty).toHaveBeenLastCalledWith(false)
  expect(await ref.current?.commitDraft()).toMatchObject({
    billingMode: 'tiered_expr',
    billingExpr: expression,
    pluginBillingExpr: {},
  })
})

it('keeps a mapped provider price when editing the origin model and removing its direct override', async () => {
  const pluginDefault = 'tier("base", u("seconds") * 0.4)'
  const mappedOverride = 'tier("base", u("seconds") * 0.8)'
  const directOverride = 'tier("base", u("seconds") * 0.9)'
  const { ref } = renderEditor(
    {
      name: 'alias',
      billingMode: 'per-token',
      inheritModelPricing: true,
      pluginBillingExpr: { alpha: directOverride },
    },
    [
      {
        plugin_key: 'alpha',
        plugin_name: 'Alpha',
        usage_schema: { seconds: { type: 'number', unit: 'second' } },
        configured: directOverride,
        mapped_provider_expr: mappedOverride,
        default_expr: pluginDefault,
        effective: directOverride,
        compatible: true,
      },
    ]
  )
  const user = userEvent.setup()
  expect(screen.getByRole('textbox', { name: 'seconds' })).toHaveValue('0.8')
  await user.click(screen.getByRole('tab', { name: 'Alpha' }))
  const provider = screen.getByRole('tabpanel', { name: 'Alpha' })
  const separate = within(provider).getByRole('switch', {
    name: 'Set separately for this provider',
  })
  expect(
    within(provider).getByRole('textbox', { name: 'seconds' })
  ).toHaveValue('0.9')
  await user.click(separate)
  expect(within(provider).getByText(mappedOverride)).toBeVisible()
  await user.click(separate)
  expect(
    within(provider).getByRole('textbox', { name: 'seconds' })
  ).toHaveValue('0.8')
  await user.click(separate)
  const inherited = await ref.current?.commitDraft()
  if (!inherited) throw new Error('Expected a valid inherited pricing draft')
  expect(pricingFromDraft(inherited)).toEqual({
    'billing_setting.plugin_billing_expr': {},
  })
  await user.click(screen.getByRole('tab', { name: 'Default' }))
  const modelPrice = screen.getByRole('textbox', { name: 'seconds' })
  await user.clear(modelPrice)
  await user.type(modelPrice, '0.6')
  await user.click(screen.getByRole('tab', { name: 'Alpha' }))
  expect(within(provider).getByText(mappedOverride)).toBeVisible()
  await user.click(separate)
  expect(
    within(provider).getByRole('textbox', { name: 'seconds' })
  ).toHaveValue('0.8')
  await user.click(separate)
  expect(await ref.current?.commitDraft()).toMatchObject({
    billingExpr: 'tier("base", u("seconds") * 0.6)',
    pluginBillingExpr: {},
  })
})

it('inherits a mapped model expression while the origin explicitly uses legacy pricing', async () => {
  const mappedExpression = 'tier("base", u("seconds") * 0.8)'
  const { ref } = renderEditor(
    {
      name: 'alias',
      billingMode: 'per-request',
      price: '0',
      inheritModelPricing: false,
    },
    [
      {
        plugin_key: 'alpha',
        plugin_name: 'Alpha',
        usage_schema: { seconds: { type: 'number', unit: 'second' } },
        configured: '',
        mapped_model_expr: mappedExpression,
        default_expr: 'tier("base", u("seconds") * 0.4)',
        effective: mappedExpression,
        compatible: true,
      },
    ]
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole('tab', { name: 'Alpha' }))
  const provider = screen.getByRole('tabpanel', { name: 'Alpha' })
  expect(within(provider).getByText(mappedExpression)).toBeVisible()
  await user.click(
    within(provider).getByRole('switch', {
      name: 'Set separately for this provider',
    })
  )
  expect(
    within(provider).getByRole('textbox', { name: 'seconds' })
  ).toHaveValue('0.8')
  expect(await ref.current?.commitDraft()).toMatchObject({
    billingMode: 'per-request',
    price: '0',
    pluginBillingExpr: { alpha: mappedExpression },
  })
})

it('keeps an explicitly empty tiered draft unconfigured instead of inheriting a mapped or plugin price', async () => {
  render(
    <TaskPluginPricingEditor
      variants={[
        {
          plugin_key: 'alpha',
          plugin_name: 'Alpha',
          usage_schema: { seconds: { type: 'number', unit: 'second' } },
          configured: '',
          mapped_model_expr: 'tier("base", u("seconds") * 0.8)',
          default_expr: 'tier("base", u("seconds") * 0.4)',
          effective: 'tier("base", u("seconds") * 0.8)',
          compatible: true,
        },
      ]}
      expressions={{}}
      onChange={vi.fn()}
      modelExpression=''
      modelBillingMode='tiered_expr'
      currency={USD_PRICING_CURRENCY}
    >
      <span>Default editor</span>
    </TaskPluginPricingEditor>
  )
  await userEvent.setup().click(screen.getByRole('tab', { name: 'Alpha' }))
  expect(screen.getByText('Not configured for this provider')).toBeVisible()
  expect(
    screen.queryByText('Using plugin default pricing')
  ).not.toBeInTheDocument()
})

it('does not report an incompatible origin expression for a provider with a mapped override', async () => {
  const mappedOverride = 'tier("base", u("seconds") * 0.8)'
  renderEditor(
    {
      name: 'alias',
      billingMode: 'tiered_expr',
      billingExpr: 'tier("base", u("credits") * 0.6)',
    },
    [
      {
        plugin_key: 'alpha',
        plugin_name: 'Alpha',
        usage_schema: { seconds: { type: 'number', unit: 'second' } },
        configured: '',
        mapped_provider_expr: mappedOverride,
        default_expr: 'tier("base", u("seconds") * 0.4)',
        effective: mappedOverride,
        compatible: true,
      },
    ]
  )
  expect(
    screen.queryByText(
      'The model-level expression cannot be evaluated by: Alpha'
    )
  ).not.toBeInTheDocument()
  await userEvent.setup().click(screen.getByRole('tab', { name: 'Alpha' }))
  expect(screen.getByText(mappedOverride)).toBeVisible()
  expect(
    screen.queryByText('Not configured for this provider')
  ).not.toBeInTheDocument()
})

it('keeps a cleared model expression explicit after editing an inherited default', async () => {
  const expression = 'tier("base", u("seconds") * 0.4)'
  const { dirty } = renderEditor(
    {
      name: 'shared',
      billingMode: 'per-token',
      inheritModelPricing: true,
    },
    [
      {
        plugin_key: 'alpha',
        plugin_name: 'Alpha',
        usage_schema: { seconds: { type: 'number', unit: 'second' } },
        configured: '',
        default_expr: expression,
        effective: expression,
        compatible: true,
      },
    ]
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
  await user.click(screen.getByRole('option', { name: 'Expression editor' }))
  await user.clear(screen.getByRole('textbox', { name: 'Billing expression' }))
  expect(dirty).toHaveBeenLastCalledWith(true)
  await user.click(screen.getByRole('tab', { name: 'Alpha' }))
  const provider = screen.getByRole('tabpanel', { name: 'Alpha' })
  expect(
    within(provider).getByText('Not configured for this provider')
  ).toBeVisible()
  expect(
    within(provider).queryByText('Using plugin default pricing')
  ).not.toBeInTheDocument()
})

it('commits the confirmed visual reset of an inherited custom expression', async () => {
  const expression = 'tier("base", u("seconds") * u("seconds") * 0.4)'
  const { ref } = renderEditor(
    { name: 'shared', billingMode: 'per-token', inheritModelPricing: true },
    [
      {
        plugin_key: 'alpha',
        plugin_name: 'Alpha',
        usage_schema: { seconds: { type: 'number', unit: 'second' } },
        configured: '',
        default_expr: expression,
        effective: expression,
        compatible: true,
      },
    ]
  )
  const user = userEvent.setup()
  expect(
    screen.getByRole('textbox', { name: 'Billing expression' })
  ).toHaveValue(expression)
  await user.click(screen.getByRole('combobox', { name: 'Editor mode' }))
  await user.click(screen.getByRole('option', { name: 'Visual editor' }))
  await user.click(
    within(screen.getByRole('alertdialog')).getByRole('button', {
      name: 'Switch to visual editor',
    })
  )
  expect(screen.getByRole('textbox', { name: 'seconds' })).toHaveValue('0')
  expect(await ref.current?.commitDraft()).toMatchObject({
    billingMode: 'tiered_expr',
    billingExpr: 'tier("base", u("seconds") * 0)',
  })
})

it.each([
  {
    kind: 'explicit ratio-only',
    mode: '{"shared":"ratio"}',
    cacheRatio: '{}',
    configured: { 'billing_setting.billing_mode': 'ratio' },
    inherited: false,
  },
  {
    kind: 'modifier-only',
    mode: '{}',
    cacheRatio: '{"shared":0.25}',
    configured: { CacheRatio: 0.25 },
    inherited: true,
  },
])(
  'retains $kind configuration when editing a provider through the global model table',
  async (fixture) => {
    useAuthStore
      .getState()
      .auth.setUser({ id: 1, username: 'administrator', role: 100 })
    usePricingPreferencesStore.setState({ currency: 'USD' })
    const mode = fixture.mode
    const expression = 'tier("base", u("seconds") * 0.4)'
    const snapshot: ModelPricingConfig = {
      options: pricingOptions({ BillingMode: mode }),
      empty_version: 'empty',
      entries: [
        {
          model_name: 'shared',
          version: 'explicit-ratio',
          configured: fixture.configured,
          effective: fixture.configured,
          usage_schema: { seconds: { type: 'number', unit: 'second' } },
          plugin_variants: [
            {
              plugin_key: 'alpha',
              plugin_name: 'Alpha',
              usage_schema: { seconds: { type: 'number', unit: 'second' } },
              configured: '',
              default_expr: expression,
              effective: fixture.inherited ? expression : '',
              compatible: fixture.inherited,
            },
          ],
        },
      ],
    }
    vi.spyOn(api, 'get').mockImplementation(async (url) => {
      if (url === '/api/option/model_pricing') {
        return { data: { success: true, data: snapshot } }
      }
      return { data: { success: true, data: [], vendors: [] } }
    })
    vi.spyOn(api, 'post').mockResolvedValue({
      data: { success: true, data: { effective: {} } },
    })
    client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    const ref = createRef<ModelRatioVisualEditorHandle>()
    const onChange = vi.fn()
    render(
      <QueryClientProvider client={client}>
        <ModelRatioVisualEditor
          ref={ref}
          savedModelPrice='{}'
          savedModelRatio='{}'
          savedCacheRatio={fixture.cacheRatio}
          savedCreateCacheRatio='{}'
          savedCompletionRatio='{}'
          savedImageRatio='{}'
          savedAudioRatio='{}'
          savedAudioCompletionRatio='{}'
          savedBillingMode={mode}
          savedBillingExpr='{}'
          modelPrice='{}'
          modelRatio='{}'
          cacheRatio={fixture.cacheRatio}
          createCacheRatio='{}'
          completionRatio='{}'
          imageRatio='{}'
          audioRatio='{}'
          audioCompletionRatio='{}'
          billingMode={mode}
          billingExpr='{}'
          onChange={onChange}
          onSave={vi.fn()}
          isSaving={false}
        />
      </QueryClientProvider>
    )
    const user = userEvent.setup()
    await waitFor(() => expect(client?.isFetching()).toBe(0))
    await user.click(
      within(await screen.findByRole('row', { name: /shared/ })).getByText(
        'shared'
      )
    )
    await screen.findByRole('tab', { name: 'Alpha' })
    expect(
      screen.getByRole('tab', {
        name: fixture.inherited ? 'Expression' : 'Per-token (deprecated)',
      })
    ).toHaveAttribute('aria-selected', 'true')
    await user.click(screen.getByRole('tab', { name: 'Alpha' }))
    const provider = screen.getByRole('tabpanel', { name: 'Alpha' })
    expect(
      within(provider).getByText(
        fixture.inherited
          ? 'Using plugin default pricing'
          : 'Use model-level pricing'
      )
    ).toBeVisible()
    await user.click(
      screen.getByRole('switch', { name: 'Set separately for this provider' })
    )
    const price = screen.getByRole('textbox', { name: 'seconds' })
    await user.clear(price)
    await user.type(price, '0.8')
    await act(async () => {
      expect(await ref.current?.commitOpenEditor()).toBe(true)
    })
    const saved = Object.fromEntries(
      onChange.mock.calls.map(([key, value]) => [key, JSON.parse(value)])
    )
    expect(saved['billing_setting.billing_mode']).toEqual(
      fixture.inherited ? {} : { shared: 'ratio' }
    )
    expect(saved.CacheRatio).toEqual(JSON.parse(fixture.cacheRatio))
    expect(saved['billing_setting.billing_expr']).toEqual({})
    expect(saved['billing_setting.plugin_billing_expr']).toEqual({
      'alpha::shared': 'tier("base", u("seconds") * 0.8)',
    })
  }
)

it('configures ordinary OpenAI pricing while preserving inherited and saved plugin prices', async () => {
  const expression = 'tier("base", u("seconds") * 0.4)'
  const savedOverride = 'tier("base", u("seconds") * 0.7)'
  const { ref, dirty } = renderEditor(
    {
      name: 'shared',
      billingMode: 'tiered_expr',
      billingExpr: expression,
      pluginBillingExpr: { beta: savedOverride },
    },
    ['alpha', 'beta'].map((key) => ({
      plugin_key: key,
      plugin_name: key,
      usage_schema: { seconds: { type: 'number', unit: 'second' } },
      configured: key === 'beta' ? savedOverride : '',
      effective: key === 'beta' ? savedOverride : expression,
      compatible: true,
    }))
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole('tab', { name: 'Ordinary OpenAI' }))
  expect(dirty).toHaveBeenLastCalledWith(false)
  expect(await ref.current?.commitDraft()).toMatchObject({
    billingExpr: expression,
    pluginBillingExpr: { beta: savedOverride },
  })
  await user.click(
    screen.getByRole('button', { name: 'Configure ordinary OpenAI pricing' })
  )
  const price = screen.getByRole('textbox', { name: 'Input price' })
  await user.clear(price)
  await user.type(price, '2.5')
  expect(dirty).toHaveBeenLastCalledWith(true)
  await user.click(screen.getByRole('tab', { name: 'alpha' }))
  expect(screen.getByRole('textbox', { name: 'seconds' })).toHaveValue('0.4')
  await user.click(screen.getByRole('tab', { name: 'beta' }))
  expect(screen.getByRole('textbox', { name: 'seconds' })).toHaveValue('0.7')
  const draft = await ref.current?.commitDraft()
  expect(draft).toMatchObject({
    billingMode: 'tiered_expr',
    billingExpr: 'tier("base", p * 2.5 + c * 0)',
    pluginBillingExpr: { alpha: expression, beta: savedOverride },
  })
})

it('starts ordinary image pricing per returned image and keeps the plugin image price', async () => {
  const expression = 'tier("base", u("image_count") * 0.04)'
  const { ref } = renderEditor(
    {
      name: 'image-model',
      billingMode: 'tiered_expr',
      billingExpr: expression,
    },
    [
      {
        plugin_key: 'alibaba',
        plugin_name: 'Alibaba',
        usage_schema: { image_count: { type: 'number', unit: 'count' } },
        configured: '',
        effective: expression,
        compatible: true,
      },
    ]
  )
  const user = userEvent.setup()
  await user.click(screen.getByRole('tab', { name: 'Ordinary OpenAI' }))
  await user.click(
    screen.getByRole('button', { name: 'Configure ordinary OpenAI pricing' })
  )
  const price = screen.getByRole('textbox', { name: 'Price per image' })
  await user.clear(price)
  await user.type(price, '0.02')
  expect(
    within(screen.getByRole('tabpanel', { name: 'Ordinary OpenAI' })).getByRole(
      'tablist'
    )
  ).toHaveClass('flex-wrap')
  const ordinary = screen.getByRole('tab', { name: 'Ordinary OpenAI' })
  await user.click(ordinary)
  await user.keyboard('{ArrowRight}{Enter}')
  expect(screen.getByRole('tab', { name: 'Alibaba' })).toHaveAttribute(
    'aria-selected',
    'true'
  )
  await user.keyboard('{ArrowLeft}{Enter}')
  expect(ordinary).toHaveAttribute('aria-selected', 'true')
  expect(price).toHaveValue('0.02')
  expect(await ref.current?.commitDraft()).toMatchObject({
    billingExpr: 'tier("image", fixed(0.02)) * image_count',
    pluginBillingExpr: { alibaba: expression },
  })
})

it.each([
  ['token', 'tier("base", p * 2 + c * 4)', 'Input price', '2'],
  [
    'per-image',
    'tier("image", fixed(0.04)) * image_count',
    'Price per image',
    '0.04',
  ],
  ['explicit zero', 'tier("free", fixed(0))', 'Price per request', '0'],
])(
  'loads ordinary %s pricing independently of plugin usage metadata',
  async (_kind, expression, field, value) => {
    const override = 'tier("base", u("seconds") * 0.4)'
    const { ref } = renderEditor(
      {
        name: 'shared',
        billingMode: 'tiered_expr',
        billingExpr: expression,
        pluginBillingExpr: { alpha: override },
      },
      [
        {
          plugin_key: 'alpha',
          plugin_name: 'Alpha',
          usage_schema: { seconds: { type: 'number', unit: 'second' } },
          configured: override,
          effective: override,
          compatible: true,
        },
      ]
    )
    expect(
      screen.getByRole('tab', { name: 'Ordinary OpenAI' })
    ).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('textbox', { name: field })).toHaveValue(value)
    const user = userEvent.setup()
    await user.click(screen.getByRole('tab', { name: 'Alpha' }))
    const price = screen.getByRole('textbox', { name: 'seconds' })
    expect(
      screen.getByRole('switch', { name: 'Set separately for this provider' })
    ).toHaveAttribute('aria-disabled', 'true')
    await user.clear(price)
    await user.type(price, '0.6')
    expect(await ref.current?.commitDraft()).toMatchObject({
      billingExpr: expression,
      pluginBillingExpr: { alpha: 'tier("base", u("seconds") * 0.6)' },
    })
  }
)
