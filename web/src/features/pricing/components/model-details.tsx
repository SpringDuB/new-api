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
import { useNavigate, useParams, useSearch } from '@tanstack/react-router'
import { ArrowLeft, ChevronDown, Code2, HeartPulse, Layers } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { PublicLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getCanvasStartURL } from '@/features/canvas-connect/connection'
import { useStatus } from '@/hooks/use-status'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'
import { useSystemConfigStore } from '@/stores/system-config-store'

import '@/styles/stolas-model-details.css'

import { DEFAULT_TOKEN_UNIT } from '../constants'
import { useBillingTime } from '../hooks/use-billing-time'
import { usePricingData } from '../hooks/use-pricing-data'
import {
  getDynamicPriceUnitLabelKey,
  getDynamicPricingSummary,
  getTaskUsageQuantityUnitLabelKey,
  isUnconfiguredTaskUsageModel,
  type DynamicPriceEntry,
} from '../lib/dynamic-price'
import {
  getModelCategory,
  getModelCategoryLabels,
} from '../lib/model-categories'
import { getModelFamily } from '../lib/model-families'
import { isTokenBasedModel } from '../lib/model-helpers'
import { formatFixedPrice, formatGroupPrice } from '../lib/price'
import { getTaskEnumFields, getTaskNumberFields } from '../lib/task-expr'
import { getTaskPricingDisplayTiers } from '../lib/task-matrix-display'
import {
  hasSimpleTaskPricing,
  taskPriceLabel,
  taskUsageUnitLabel,
  pricingDisplayFallbackKey,
} from '../lib/task-price-display'
import type { PriceType, PricingModel, TokenUnit } from '../types'
import { DynamicPricingBreakdown } from './dynamic-pricing-breakdown'
import { ModelCapabilities } from './model-capabilities'
import { ModelDetailsApi } from './model-details-api'
import { ModelDetailsPerformance } from './model-details-performance'

// ----------------------------------------------------------------------------
// Local UI helpers
// ----------------------------------------------------------------------------

function SectionTitle(props: { children: React.ReactNode }) {
  return (
    <h2 className='text-muted-foreground mb-3 text-xs font-semibold tracking-wider uppercase'>
      {props.children}
    </h2>
  )
}

function DynamicPriceEntryLabel(props: { entry: DynamicPriceEntry }) {
  const { t, i18n } = useTranslation()
  if (props.entry.labelKind === 'schema') {
    return (
      <span className='break-words whitespace-normal'>
        {taskPriceLabel(
          props.entry.description,
          props.entry.shortLabel,
          i18n.language
        )}
      </span>
    )
  }
  return t(props.entry.shortLabel)
}

function UnconfiguredTaskPricingNotice(props: { model: PricingModel }) {
  const { t, i18n } = useTranslation()
  const numberFields = getTaskNumberFields(props.model.billing_usage_schema)
  const enumFields = getTaskEnumFields(props.model.billing_usage_schema)

  return (
    <div className='bg-muted/20 flex flex-col gap-3 rounded-lg border p-3'>
      <p className='text-muted-foreground text-sm'>
        {t(
          'This model is billed by usage, but the administrator has not configured its pricing yet.'
        )}
      </p>
      {numberFields.length + enumFields.length > 0 ? (
        <dl className='grid gap-2 sm:grid-cols-2'>
          {numberFields.map(([field, definition]) => (
            <div
              key={field}
              className='flex items-baseline justify-between gap-3'
            >
              <dt className='text-sm'>
                {taskPriceLabel(definition.description, field, i18n.language)}
              </dt>
              <dd className='text-muted-foreground text-xs'>
                {taskUsageUnitLabel(
                  definition,
                  i18n.language,
                  t(getTaskUsageQuantityUnitLabelKey(definition.unit))
                )}
              </dd>
            </div>
          ))}
          {enumFields.map(([field, definition]) => (
            <div
              key={field}
              className='flex items-baseline justify-between gap-3'
            >
              <dt className='text-sm'>
                <code>{field}</code>
              </dt>
              <dd className='text-muted-foreground text-right text-xs'>
                {(definition.enum ?? []).join(', ')}
              </dd>
            </div>
          ))}
        </dl>
      ) : null}
    </div>
  )
}

const MODEL_DETAILS_SKELETON_KEYS = ['first', 'second', 'third', 'fourth']

// ----------------------------------------------------------------------------
// Model header (always visible above the detail sections)
// ----------------------------------------------------------------------------

function ModelHeader(props: { model: PricingModel }) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const model = props.model
  const displayName = model.display_name || model.model_name
  const modelIconKey = model.icon || model.vendor_icon
  const modelIcon = modelIconKey ? getLobeIcon(modelIconKey, 30) : null
  const description = model.description || model.vendor_description || null
  const category = getModelCategoryLabels(t)[getModelCategory(model)]
  const family = getModelFamily(model)

  return (
    <header className='stolas-model-detail-header'>
      <div className='stolas-model-detail-eyebrow'>
        {modelIcon}
        <span>
          {category}
          {family ? ` / ${family}` : ''}
        </span>
      </div>
      <div className='flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1'>
        <h1
          className={cn(
            'stolas-model-detail-name min-w-0',
            displayName !== model.model_name && 'basis-full'
          )}
        >
          {displayName}
        </h1>
        {displayName !== model.model_name && (
          <code className='text-muted-foreground text-sm [overflow-wrap:anywhere]'>
            {model.model_name}
          </code>
        )}
        <CopyButton
          value={model.model_name || ''}
          className='mt-1 size-7 shrink-0'
          iconClassName='size-3.5'
          tooltip={t('Copy model name')}
          successTooltip={t('Copied!')}
          aria-label={t('Copy model name')}
        />
      </div>
      {description && description.length <= 200 && (
        <p className='stolas-model-detail-description'>{description}</p>
      )}
      <div className='mt-5 flex flex-wrap items-center gap-3'>
        {status?.canvas_enabled && (
          <Button
            variant='secondary'
            size='sm'
            render={
              <a
                href={getCanvasStartURL(status)}
                target='_blank'
                rel='noopener noreferrer'
              />
            }
          >
            {t('Try in canvas')}
            <ArrowLeft aria-hidden='true' className='rotate-180' />
          </Button>
        )}
        {description && description.length > 200 && (
          <Collapsible className='stolas-model-detail-notes'>
            <CollapsibleTrigger
              className='group'
              render={<Button variant='ghost' size='sm' />}
            >
              {t('Usage notes')}
              <ChevronDown
                aria-hidden='true'
                className='group-data-[panel-open]:rotate-180'
              />
            </CollapsibleTrigger>
            <CollapsibleContent>
              <p className='stolas-model-detail-description'>{description}</p>
            </CollapsibleContent>
          </Collapsible>
        )}
      </div>
    </header>
  )
}

// ----------------------------------------------------------------------------
// Base price card (used in the Overview tab)
// ----------------------------------------------------------------------------

function PriceSection(props: {
  model: PricingModel
  priceRate: number
  usdExchangeRate: number
  tokenUnit: TokenUnit
  showRechargePrice: boolean
}) {
  const { t, i18n } = useTranslation()
  const isTokenBased = isTokenBasedModel(props.model)
  const tokenUnitLabel = props.tokenUnit === 'K' ? '1K' : '1M'
  const baseGroupKey = '_base'
  const baseGroupRatioMap = { [baseGroupKey]: 1 }
  const currency = useSystemConfigStore((state) => state.config.currency)
  const billingTime = useBillingTime(props.model.billing_expr)
  const dynamicSummary = useMemo(
    () =>
      getDynamicPricingSummary(props.model, {
        now: billingTime === undefined ? undefined : new Date(billingTime),
        tokenUnit: props.tokenUnit,
        showRechargePrice: props.showRechargePrice,
        priceRate: props.priceRate,
        usdExchangeRate: props.usdExchangeRate,
        groupRatioMultiplier: 1,
      }),
    // Currency is read indirectly by the price formatter.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [
      props.model,
      props.tokenUnit,
      props.showRechargePrice,
      props.priceRate,
      props.usdExchangeRate,
      billingTime,
      currency,
    ]
  )

  const primaryPriceTypes: { label: string; type: PriceType }[] = [
    { label: t('Input'), type: 'input' },
    { label: t('Output'), type: 'output' },
  ]
  const secondaryPriceTypes: {
    label: string
    type: PriceType
    available: boolean
  }[] = [
    {
      label: t('Cached input'),
      type: 'cache',
      available: props.model.cache_ratio != null,
    },
    {
      label: t('Cache write'),
      type: 'create_cache',
      available: props.model.create_cache_ratio != null,
    },
    {
      label: t('Image input'),
      type: 'image',
      available: props.model.image_ratio != null,
    },
    {
      label: t('Audio input'),
      type: 'audio_input',
      available: props.model.audio_ratio != null,
    },
    {
      label: t('Audio output'),
      type: 'audio_output',
      available:
        props.model.audio_ratio != null &&
        props.model.audio_completion_ratio != null,
    },
  ]

  if (dynamicSummary) {
    if (dynamicSummary.isSpecialExpression) {
      return (
        <section>
          <SectionTitle>{t('Base Price')}</SectionTitle>
          <div className='rounded-lg border border-amber-200/70 bg-amber-50/70 p-3 dark:border-amber-500/20 dark:bg-amber-500/10'>
            <div className='text-sm font-medium text-amber-800 dark:text-amber-200'>
              {t('Special billing expression')}
            </div>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t(
                pricingDisplayFallbackKey(
                  dynamicSummary.rawExpression,
                  props.model.billing_usage_schema
                )
              )}
            </p>
            <div className='mt-3'>
              <div className='text-muted-foreground mb-1 text-[10px] font-medium tracking-wider uppercase'>
                {t('Raw expression')}
              </div>
              <code className='text-muted-foreground bg-background/80 block max-h-28 overflow-auto rounded-md border px-2 py-1.5 font-mono text-xs break-all'>
                {dynamicSummary.rawExpression}
              </code>
            </div>
          </div>
        </section>
      )
    }

    const variants = props.model.billing_plugin_variants
    const pricedVariants = variants?.filter(
      (variant) => variant.billing_expr && variant.billing_usage_schema
    )
    if (
      pricedVariants?.length &&
      variants?.every((variant) => variant.billing_mode !== 'ratio')
    ) {
      return (
        <section aria-label={t('Base Price')}>
          <SectionTitle>{t('Base Price')}</SectionTitle>
          <div className='space-y-3'>
            {pricedVariants.map((variant) => (
              <DynamicPricingBreakdown
                key={variant.plugin_key}
                billingExpr={variant.billing_expr}
                usageSchema={variant.billing_usage_schema}
                compact
                taskPriceOptions={{
                  showRechargePrice: props.showRechargePrice,
                  priceRate: props.priceRate,
                  usdExchangeRate: props.usdExchangeRate,
                }}
              />
            ))}
          </div>
        </section>
      )
    }

    return (
      <section>
        <SectionTitle>{t('Base Price')}</SectionTitle>
        {dynamicSummary.isMixedBilling && (
          <p className='text-muted-foreground mb-2 text-xs'>
            {t('Token or per-call pricing')}
          </p>
        )}
        {dynamicSummary.primaryEntries.length > 0 ? (
          <div
            className={cn(
              'grid gap-2',
              dynamicSummary.primaryEntries.length > 1 && 'grid-cols-2'
            )}
          >
            {dynamicSummary.primaryEntries.map((entry) => {
              const unitLabelKey = getDynamicPriceUnitLabelKey(entry)
              const unitLabel = taskUsageUnitLabel(
                entry,
                i18n.language,
                unitLabelKey ? t(unitLabelKey) : tokenUnitLabel
              )
              return (
                <div
                  key={entry.key}
                  className='bg-muted/20 rounded-lg border p-3'
                >
                  <div className='text-muted-foreground text-xs'>
                    <DynamicPriceEntryLabel entry={entry} />
                  </div>
                  <div className='text-foreground mt-1 font-mono text-base font-semibold tabular-nums'>
                    {entry.formattedRange ?? entry.formatted}
                    <span className='text-muted-foreground/40 ml-1 text-xs font-normal'>
                      / {unitLabel}
                    </span>
                  </div>
                </div>
              )
            })}
          </div>
        ) : (
          <p className='text-muted-foreground text-sm'>
            {dynamicSummary.hasUnconfiguredProviders
              ? t('Not configured')
              : t('Dynamic Pricing')}
          </p>
        )}
        {dynamicSummary.secondaryEntries.length > 0 && (
          <div className='bg-muted/20 mt-3 rounded-lg border px-3 py-2.5'>
            <div className='space-y-1.5'>
              {dynamicSummary.secondaryEntries.map((entry) => {
                const unitLabelKey = getDynamicPriceUnitLabelKey(entry)
                const unitLabel = taskUsageUnitLabel(
                  entry,
                  i18n.language,
                  unitLabelKey ? t(unitLabelKey) : tokenUnitLabel
                )
                return (
                  <div
                    key={entry.key}
                    className='flex items-baseline justify-between gap-4'
                  >
                    <span className='text-muted-foreground/70 text-sm'>
                      <DynamicPriceEntryLabel entry={entry} />
                    </span>
                    <span className='text-muted-foreground font-mono text-sm tabular-nums'>
                      {entry.formattedRange ?? entry.formatted}
                      <span className='text-muted-foreground/40 ml-1 text-xs font-normal'>
                        / {unitLabel}
                      </span>
                    </span>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </section>
    )
  }

  if (isUnconfiguredTaskUsageModel(props.model)) {
    return (
      <section>
        <SectionTitle>{t('Base Price')}</SectionTitle>
        <UnconfiguredTaskPricingNotice model={props.model} />
      </section>
    )
  }

  if (!isTokenBased) {
    return (
      <section>
        <SectionTitle>{t('Base Price')}</SectionTitle>
        <div className='flex items-baseline justify-between'>
          <span className='text-muted-foreground text-sm'>
            {t('Per request')}
          </span>
          <span className='text-foreground font-mono text-sm font-semibold tabular-nums'>
            {formatFixedPrice(
              props.model,
              baseGroupKey,
              props.showRechargePrice,
              props.priceRate,
              props.usdExchangeRate,
              baseGroupRatioMap
            )}
          </span>
        </div>
      </section>
    )
  }

  const secondaryItems = secondaryPriceTypes.filter((p) => p.available)
  const renderPrice = (type: PriceType) => (
    <>
      {formatGroupPrice(
        props.model,
        baseGroupKey,
        type,
        props.tokenUnit,
        props.showRechargePrice,
        props.priceRate,
        props.usdExchangeRate,
        baseGroupRatioMap
      )}
      <span className='text-muted-foreground/40 ml-1 text-xs font-normal'>
        / {tokenUnitLabel}
      </span>
    </>
  )

  return (
    <section>
      <SectionTitle>{t('Base Price')}</SectionTitle>
      <div className='grid grid-cols-2 gap-2'>
        {primaryPriceTypes.map((item) => (
          <div key={item.type} className='bg-muted/20 rounded-lg border p-3'>
            <div className='text-muted-foreground text-xs'>{item.label}</div>
            <div className='text-foreground mt-1 font-mono text-base font-semibold tabular-nums'>
              {renderPrice(item.type)}
            </div>
          </div>
        ))}
      </div>
      {secondaryItems.length > 0 && (
        <div className='bg-muted/20 mt-3 rounded-lg border px-3 py-2.5'>
          <div className='space-y-1.5'>
            {secondaryItems.map((item) => (
              <div
                key={item.type}
                className='flex items-baseline justify-between gap-4'
              >
                <span className='text-muted-foreground/70 text-sm'>
                  {item.label}
                </span>
                <span className='text-muted-foreground font-mono text-sm tabular-nums'>
                  {renderPrice(item.type)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  )
}

const TAB_VALUES = ['overview', 'api', 'performance'] as const
type TabValue = (typeof TAB_VALUES)[number]

const TAB_META: Record<
  TabValue,
  { icon: React.ComponentType<{ className?: string }>; labelKey: string }
> = {
  overview: { icon: Layers, labelKey: 'Overview' },
  performance: { icon: HeartPulse, labelKey: 'Performance' },
  api: { icon: Code2, labelKey: 'API' },
}

export interface ModelDetailsContentProps {
  model: PricingModel
  groupRatio: Record<string, number>
  usableGroup: Record<string, { desc: string; ratio: number }>
  endpointMap: Record<string, { path?: string; method?: string }>
  autoGroups: string[]
  priceRate: number
  usdExchangeRate: number
  tokenUnit: TokenUnit
  showRechargePrice?: boolean
}

export function ModelDetailsContent(props: ModelDetailsContentProps) {
  const { t } = useTranslation()
  const showRechargePrice = props.showRechargePrice ?? false

  const isDynamic =
    props.model.billing_mode === 'tiered_expr' &&
    Boolean(props.model.billing_expr)

  const simpleTaskPricing = hasSimpleTaskPricing(props.model)
  const taskTiers = getTaskPricingDisplayTiers(
    props.model.billing_expr,
    props.model.billing_usage_schema
  )
  const showBasePrices =
    !props.model.billing_usage_schema ||
    simpleTaskPricing ||
    taskTiers.length === 0

  return (
    <div className='stolas-model-detail @container/details'>
      <ModelHeader model={props.model} />

      <Tabs
        key={props.model.model_name}
        defaultValue='overview'
        className='gap-6'
      >
        <TabsList variant='line' className='stolas-model-detail-tabs'>
          {TAB_VALUES.map((value) => {
            const Icon = TAB_META[value].icon
            return (
              <TabsTrigger
                key={value}
                value={value}
                className='min-w-0 gap-1.5 px-1 text-xs sm:text-sm'
              >
                <Icon aria-hidden='true' className='hidden size-3.5 sm:block' />
                <span className='truncate'>{t(TAB_META[value].labelKey)}</span>
              </TabsTrigger>
            )
          })}
        </TabsList>

        <TabsContent value='overview' className='space-y-8 outline-none'>
          <ModelCapabilities model={props.model} />
          <section
            aria-label={t('Pricing')}
            className='stolas-model-detail-pricing'
          >
            {showBasePrices && (
              <PriceSection
                model={props.model}
                priceRate={props.priceRate}
                usdExchangeRate={props.usdExchangeRate}
                tokenUnit={props.tokenUnit}
                showRechargePrice={showRechargePrice}
              />
            )}
            {isDynamic && !simpleTaskPricing && (
              <DynamicPricingBreakdown
                billingExpr={props.model.billing_expr}
                usageSchema={props.model.billing_usage_schema}
                taskPriceOptions={{
                  showRechargePrice,
                  priceRate: props.priceRate,
                  usdExchangeRate: props.usdExchangeRate,
                }}
              />
            )}
          </section>
        </TabsContent>

        <TabsContent value='performance' className='outline-none'>
          <ModelDetailsPerformance model={props.model} />
        </TabsContent>

        <TabsContent value='api' className='outline-none'>
          <ModelDetailsApi
            model={props.model}
            endpointMap={props.endpointMap}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}

// ----------------------------------------------------------------------------
// Drawer & page wrappers
// ----------------------------------------------------------------------------

export interface ModelDetailsDrawerProps extends ModelDetailsContentProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ModelDetailsDrawer(props: ModelDetailsDrawerProps) {
  const { t } = useTranslation()
  const { open, onOpenChange, ...contentProps } = props

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={props.model.display_name || props.model.model_name}
      description={t('Model details')}
      headerClassName='sr-only'
      contentClassName='stolas-model-detail-dialog'
      bodyClassName='stolas-model-detail-dialog-body'
    >
      <ModelDetailsContent {...contentProps} />
    </Dialog>
  )
}

export function ModelDetails() {
  const { t } = useTranslation()
  const { modelId } = useParams({ from: '/pricing/$modelId/' })
  const search = useSearch({ from: '/pricing/$modelId/' })
  const navigate = useNavigate()

  const {
    models,
    groupRatio,
    usableGroup,
    endpointMap,
    autoGroups,
    isLoading,
    priceRate,
    usdExchangeRate,
  } = usePricingData(true, { refetchOnMount: 'always' })

  const tokenUnit: TokenUnit =
    search.tokenUnit === 'K' ? 'K' : DEFAULT_TOKEN_UNIT

  const model = useMemo(() => {
    if (!models || !modelId) return null
    return models.find((m) => m.model_name === modelId) || null
  }, [models, modelId])

  const handleBack = () => {
    navigate({ to: '/pricing', search })
  }

  if (isLoading) {
    return (
      <PublicLayout>
        <div className='mx-auto max-w-5xl px-4 sm:px-6'>
          <Skeleton className='mb-4 h-5 w-16' />
          <div className='space-y-2'>
            <Skeleton className='h-7 w-64' />
            <Skeleton className='h-4 w-40' />
            <Skeleton className='h-4 w-full max-w-md' />
          </div>
          <div className='mt-6 grid grid-cols-2 gap-2 sm:grid-cols-4'>
            {MODEL_DETAILS_SKELETON_KEYS.map((key) => (
              <Skeleton key={`metric-${key}`} className='h-16 w-full' />
            ))}
          </div>
          <div className='mt-6 space-y-3'>
            {MODEL_DETAILS_SKELETON_KEYS.map((key) => (
              <Skeleton key={`section-${key}`} className='h-24 w-full' />
            ))}
          </div>
        </div>
      </PublicLayout>
    )
  }

  if (!model) {
    return (
      <PublicLayout>
        <div className='mx-auto max-w-2xl px-4 text-center sm:px-6'>
          <h2 className='mb-1 text-base font-semibold'>
            {t('Model not found')}
          </h2>
          <p className='text-muted-foreground mb-4 text-sm'>
            {t("The model you're looking for doesn't exist.")}
          </p>
          <Button onClick={handleBack} variant='outline' size='sm'>
            {t('Back to Models')}
          </Button>
        </div>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout>
      <div className='mx-auto max-w-5xl px-4 sm:px-6'>
        <Button
          variant='ghost'
          size='sm'
          onClick={handleBack}
          className='text-muted-foreground hover:text-foreground mb-4 h-auto gap-1 px-0 py-1 text-xs'
        >
          <ArrowLeft className='size-3.5' />
          {t('Back')}
        </Button>

        <ModelDetailsContent
          model={model}
          groupRatio={groupRatio || {}}
          usableGroup={usableGroup || {}}
          autoGroups={autoGroups || []}
          priceRate={priceRate ?? 1}
          usdExchangeRate={usdExchangeRate ?? 1}
          tokenUnit={tokenUnit}
          showRechargePrice={search.rechargePrice ?? false}
          endpointMap={
            (endpointMap as Record<
              string,
              { path?: string; method?: string }
            >) || {}
          }
        />
      </div>
    </PublicLayout>
  )
}
