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
import { ChevronRight } from 'lucide-react'
import { memo, useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardFooter, CardHeader } from '@/components/ui/card'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { DEFAULT_TOKEN_UNIT } from '../constants'
import { useBillingTime } from '../hooks/use-billing-time'
import {
  getCardExamplePrice,
  getDynamicDisplayGroupRatio,
  getDynamicPriceUnitLabelKey,
  getDynamicPricingSummary,
  isUnconfiguredTaskUsageModel,
} from '../lib/dynamic-price'
import { parseTags } from '../lib/filters'
import {
  getModelCategory,
  getModelCategoryLabels,
} from '../lib/model-categories'
import { isTokenBasedModel } from '../lib/model-helpers'
import { MODEL_SQUARE_COLORS } from '../lib/model-square-display'
import { formatPrice, formatRequestPrice } from '../lib/price'
import { taskPriceLabel, taskUsageUnitLabel } from '../lib/task-price-display'
import type { PricingModel, PriceType, TokenUnit } from '../types'
import { ImageSpecPanel } from './image-spec-panel'
import { ModelBillingModeBadge } from './model-billing-mode-badge'
import { ModelPerfBadge, type ModelPerfBadgeData } from './model-perf-badge'
import { ModelPromotionBadge } from './model-promotion-badge'
import { VideoSpecPanel } from './video-spec-panel'

export interface ModelCardProps {
  model: PricingModel
  cover?: string
  onClick: (modelName: string) => void
  priceRate?: number
  usdExchangeRate?: number
  tokenUnit?: TokenUnit
  showRechargePrice?: boolean
  selectedGroup?: string
  perf?: ModelPerfBadgeData
  presentation?: 'card' | 'directory'
}

function DirectoryPrice(props: {
  amount: string
  unit: string
  label?: string
  showLabel?: boolean
}) {
  return (
    <span
      className='stolas-model-price-entry'
      title={`${props.label ? `${props.label}: ` : ''}${props.amount} / ${props.unit}`}
    >
      {props.showLabel && props.label && (
        <span className='text-muted-foreground'>{props.label}: </span>
      )}
      <span className='font-semibold tabular-nums'>{props.amount}</span>
      <span className='text-muted-foreground'> / {props.unit}</span>
    </span>
  )
}

export const ModelCard = memo(function ModelCard(props: ModelCardProps) {
  const { t, i18n } = useTranslation()
  const isDirectory = props.presentation === 'directory'
  const displayName = props.model.display_name || props.model.model_name
  const tokenUnit = props.tokenUnit ?? DEFAULT_TOKEN_UNIT
  const priceRate = props.priceRate ?? 1
  const usdExchangeRate = props.usdExchangeRate ?? 1
  const showRechargePrice = props.showRechargePrice ?? false
  const isTokenBased = isTokenBasedModel(props.model)
  const tokenUnitLabel = tokenUnit === 'K' ? '1K' : '1M'
  const groups = props.model.enable_groups || []
  const endpoints = props.model.supported_endpoint_types || []
  const modelIconKey =
    props.model.icon?.trim() || props.model.vendor_icon?.trim()
  const modelIcon = modelIconKey
    ? getLobeIcon(modelIconKey, isDirectory ? 16 : 28)
    : null
  const vendorIconKey = props.model.vendor_icon?.trim()
  const initial = props.model.model_name?.charAt(0).toUpperCase() || '?'
  const isUnconfiguredTaskUsage = isUnconfiguredTaskUsageModel(props.model)
  const billingTime = useBillingTime(props.model.billing_expr)
  const currency = useSystemConfigStore((state) => state.config.currency)
  const videoSpecs = props.model.video_specs
  const imageSpecs = props.model.image_specs
  const isSeedancePerCall = [
    'seedance-2-0-per-request',
    'seedance-2-5-per-request',
  ].includes(props.model.model_name)
  const hasSpecs = !!videoSpecs?.length || !!imageSpecs?.length
  const imageFallback = !hasSpecs && getModelCategory(props.model) === 'image'
  const inPanel = hasSpecs || imageFallback
  const tags = isDirectory ? [] : parseTags(props.model.tags)
  const artwork = isDirectory ? props.cover : undefined
  const dynamicPriceOptions = useMemo(
    () => ({
      now: billingTime === undefined ? undefined : new Date(billingTime),
      tokenUnit,
      showRechargePrice,
      priceRate,
      usdExchangeRate,
      groupRatioMultiplier: getDynamicDisplayGroupRatio(
        props.model,
        props.selectedGroup
      ),
    }),
    [
      props.model,
      props.selectedGroup,
      billingTime,
      tokenUnit,
      showRechargePrice,
      priceRate,
      usdExchangeRate,
    ]
  )
  const dynamicSummary = useMemo(
    () =>
      hasSpecs
        ? null
        : getDynamicPricingSummary(props.model, dynamicPriceOptions),
    // Currency is read indirectly by the price formatter.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [props.model, dynamicPriceOptions, currency, hasSpecs]
  )
  const cardExamplePrice = useMemo(
    () =>
      isDirectory || hasSpecs
        ? null
        : getCardExamplePrice(props.model, dynamicPriceOptions),
    // Currency is read indirectly by the price formatter.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [props.model, dynamicPriceOptions, currency, isDirectory, hasSpecs]
  )
  let priceSummary: ReactNode
  if (hasSpecs) {
    priceSummary = null // The specification panel owns pricing for this card.
  } else if (dynamicSummary) {
    const cacheReadEntries = dynamicSummary.secondaryEntries.filter(
      (entry) => entry.field === 'cacheReadPrice' && entry.unit === 'token'
    )
    const entries = [
      ...dynamicSummary.primaryEntries.slice(
        0,
        dynamicSummary.providerCount ? 2 : undefined
      ),
      ...cacheReadEntries,
    ]
    if (dynamicSummary.isSpecialExpression) {
      priceSummary = (
        <div className='col-span-full min-w-0'>
          <span className='text-warning'>
            {t('Special billing expression')}
          </span>
          {!isDirectory && (
            <code className='text-muted-foreground mt-1 line-clamp-2 block font-mono text-xs break-all'>
              {dynamicSummary.rawExpression}
            </code>
          )}
        </div>
      )
    } else if (entries.length > 0) {
      priceSummary = (
        <>
          {entries.map((entry) => {
            const unitLabelKey = getDynamicPriceUnitLabelKey(entry)
            const unitLabel = taskUsageUnitLabel(
              entry,
              i18n.language,
              unitLabelKey ? t(unitLabelKey) : tokenUnitLabel
            )
            const label =
              entry.labelKind === 'schema'
                ? taskPriceLabel(
                    entry.description,
                    entry.shortLabel,
                    i18n.language
                  )
                : t(entry.shortLabel)
            const amount = entry.formattedRange ?? entry.formatted
            if (isDirectory) {
              return (
                <DirectoryPrice
                  key={entry.key}
                  amount={amount}
                  unit={unitLabel}
                  label={label}
                  showLabel={entries.length > 1}
                />
              )
            }
            return (
              <div
                key={entry.key}
                className={cn(
                  'flex min-w-0 flex-col gap-1',
                  dynamicSummary.isTaskUsage && 'col-span-full'
                )}
              >
                {label && (
                  <span className='text-muted-foreground text-xs break-words whitespace-normal'>
                    {label}
                  </span>
                )}
                <span className='flex flex-wrap items-baseline gap-x-1 font-mono text-sm font-semibold tabular-nums'>
                  <span>{amount}</span>
                  <span className='text-muted-foreground text-xs font-normal whitespace-nowrap'>
                    {' '}
                    / {unitLabel}
                  </span>
                </span>
              </div>
            )
          })}
          {!isDirectory && dynamicSummary.isTimePricing && (
            <span className='text-muted-foreground col-span-full text-xs'>
              {t('Current period price')}
            </span>
          )}
          {!isDirectory && dynamicSummary.isMixedBilling && (
            <span className='text-muted-foreground col-span-full text-xs'>
              {t('Token or per-call pricing')}
            </span>
          )}
          {cardExamplePrice && (
            <span className='text-muted-foreground col-span-full text-xs break-words'>
              {cardExamplePrice.label} ≈ {cardExamplePrice.formatted}
            </span>
          )}
          {!isDirectory &&
            dynamicSummary.isTaskUsage &&
            dynamicSummary.tierCount > 1 &&
            dynamicSummary.tier?.label &&
            !dynamicSummary.primaryEntries.some(
              (entry) => entry.formattedRange
            ) && (
              <span className='text-muted-foreground col-span-full text-xs break-words'>
                {t('{{count}} tiers', { count: dynamicSummary.tierCount })}
              </span>
            )}
        </>
      )
    } else {
      priceSummary = (
        <span className='text-muted-foreground col-span-full'>
          {dynamicSummary.hasUnconfiguredProviders
            ? t('Usage-based billing · price not configured')
            : t('Dynamic Pricing')}
        </span>
      )
    }
  } else if (isUnconfiguredTaskUsage) {
    priceSummary = (
      <span className='text-muted-foreground col-span-full'>
        {t('Usage-based billing · price not configured')}
      </span>
    )
  } else if (isTokenBased) {
    const prices: { type: PriceType; label: string }[] = [
      { type: 'input', label: t('Input') },
      { type: 'output', label: t('Output') },
      ...(props.model.cache_ratio != null
        ? [{ type: 'cache' as const, label: t('Cached') }]
        : []),
    ]
    priceSummary = prices.map((price) => {
      const amount = formatPrice(
        props.model,
        price.type,
        tokenUnit,
        showRechargePrice,
        priceRate,
        usdExchangeRate,
        props.selectedGroup
      )
      if (isDirectory) {
        return (
          <DirectoryPrice
            key={price.type}
            amount={amount}
            unit={tokenUnitLabel}
            label={price.label}
            showLabel
          />
        )
      }
      return (
        <div key={price.type} className='flex min-w-0 flex-col gap-1'>
          <span className='text-muted-foreground text-xs'>{price.label}</span>
          <span className='font-mono text-sm font-semibold tabular-nums'>
            {amount}
            <span className='text-muted-foreground text-xs font-normal'>
              {' '}
              / {tokenUnitLabel}
            </span>
          </span>
        </div>
      )
    })
  } else {
    const amount = formatRequestPrice(
      props.model,
      showRechargePrice,
      priceRate,
      usdExchangeRate,
      props.selectedGroup
    )
    const unit = t('request')
    priceSummary = isDirectory ? (
      <DirectoryPrice amount={amount} unit={unit} />
    ) : (
      <div className='col-span-full flex min-w-0 flex-col gap-1'>
        <span className='font-mono text-sm font-semibold tabular-nums'>
          {amount}
          <span className='text-muted-foreground text-xs font-normal'>
            {' '}
            / {unit}
          </span>
        </span>
      </div>
    )
  }

  const promotionNote = props.model.square_display?.note && (
    <p
      className={cn(
        'font-semibold',
        isDirectory ? 'truncate text-xs' : 'text-sm break-words',
        MODEL_SQUARE_COLORS[props.model.square_display.color ?? 'red'].note
      )}
      title={isDirectory ? props.model.square_display.note : undefined}
    >
      {props.model.square_display.note}
    </p>
  )
  const pricing = isDirectory ? (
    <div
      role='group'
      aria-label={t('Pricing')}
      className='min-w-0 truncate text-xs'
    >
      {priceSummary}
    </div>
  ) : (
    <div
      role='group'
      aria-label={t('Pricing')}
      className='mt-auto flex min-w-0 flex-col gap-1.5'
    >
      <ModelBillingModeBadge model={props.model} appearance='caption' />
      {dynamicSummary?.providerCount && (
        <span className='text-muted-foreground text-xs break-words'>
          {t('{{count}} providers', { count: dynamicSummary.providerCount })}
          {dynamicSummary.hasUnconfiguredProviders &&
            ` · ${t('Not configured for some providers')}`}
        </span>
      )}
      <div className='grid grid-cols-[repeat(auto-fit,minmax(88px,1fr))] gap-x-3 gap-y-2'>
        {priceSummary}
      </div>
      {promotionNote}
    </div>
  )

  const detailsAction = (
    <Button
      variant='ghost'
      size='sm'
      aria-label={isDirectory ? t('Details') : undefined}
      title={isDirectory ? t('Details') : undefined}
      onClick={() => props.onClick(props.model.model_name || '')}
      className={isDirectory ? 'stolas-model-details' : undefined}
    >
      {t('Details')}
      <ChevronRight aria-hidden data-icon='inline-end' />
    </Button>
  )
  const copyAction = (
    <CopyButton
      value={props.model.model_name}
      tooltip={t('Copy model name')}
      className={isDirectory ? 'ms-auto size-5' : 'size-7'}
      iconClassName={isDirectory ? 'size-3' : 'size-3.5'}
    />
  )

  return (
    <Card
      role={isDirectory ? 'article' : undefined}
      aria-label={isDirectory ? displayName : undefined}
      data-highlight={props.model.square_display?.highlight || undefined}
      data-artwork={artwork ? true : undefined}
      className={cn(
        'hover:ring-foreground/20 h-full min-w-0 gap-3 transition-colors',
        isDirectory && 'stolas-model-tile',
        props.model.square_display?.highlight &&
          MODEL_SQUARE_COLORS[props.model.square_display.color ?? 'red'].ring
      )}
    >
      {isDirectory && (
        <div
          data-slot='model-cover'
          className={cn(
            'stolas-model-cover',
            artwork && 'stolas-model-artwork-frame'
          )}
        >
          {artwork ? (
            <img
              className='stolas-model-cover-image'
              src={artwork}
              alt=''
              width={640}
              height={480}
              loading='lazy'
              decoding='async'
            />
          ) : (
            <div aria-hidden='true' className='stolas-model-cover-icon'>
              {(vendorIconKey && getLobeIcon(vendorIconKey, 48)) ||
                (modelIconKey && getLobeIcon(modelIconKey, 48)) || (
                  <span>{initial}</span>
                )}
            </div>
          )}
          {props.model.square_display?.badge && (
            <div className='stolas-model-cover-promotion'>
              <ModelPromotionBadge decoration={props.model.square_display} />
            </div>
          )}
          {groups.length > 0 && (
            <Badge
              variant='secondary'
              className='stolas-model-cover-group'
              title={groups.join(', ')}
            >
              {groups[0]}
              {groups.length > 1 && ` +${groups.length - 1}`}
            </Badge>
          )}
        </div>
      )}
      {!isDirectory && props.model.square_display?.badge && (
        <div className='px-4 pb-1'>
          <ModelPromotionBadge decoration={props.model.square_display} />
        </div>
      )}
      <CardHeader
        className={cn(
          'flex flex-row items-start gap-3',
          isDirectory && 'stolas-model-identity'
        )}
      >
        {(!isDirectory || artwork) && (
          <div
            aria-hidden
            className={cn(
              'bg-muted/50 flex shrink-0 items-center justify-center rounded-lg',
              isDirectory ? 'size-6 rounded-md' : 'size-10'
            )}
          >
            {(artwork && vendorIconKey && getLobeIcon(vendorIconKey, 16)) ||
              modelIcon || (
                <span className='text-muted-foreground text-sm font-bold'>
                  {initial}
                </span>
              )}
          </div>
        )}
        <div className='min-w-0 flex-1'>
          <h3
            className={cn(
              'text-[15px] leading-snug font-semibold',
              !props.model.display_name && 'font-mono',
              isDirectory ? 'truncate' : 'line-clamp-2 [overflow-wrap:anywhere]'
            )}
            title={displayName}
          >
            {displayName}
          </h3>
          {isDirectory ? (
            <div className='stolas-model-meta'>
              {artwork && (
                <span className='stolas-model-category'>
                  {getModelCategoryLabels(t)[getModelCategory(props.model)]}
                </span>
              )}
              <code
                className='min-w-0 truncate font-mono'
                title={props.model.model_name}
              >
                {props.model.model_name}
              </code>
              {copyAction}
            </div>
          ) : (
            props.model.vendor_name && (
              <p
                className='text-muted-foreground mt-1 truncate text-xs'
                title={props.model.vendor_name}
              >
                {props.model.vendor_name}
              </p>
            )
          )}
        </div>
        {!isDirectory && copyAction}
      </CardHeader>
      <CardContent
        className={cn(
          'flex flex-1 flex-col gap-3',
          isDirectory && 'stolas-model-content'
        )}
      >
        {!!videoSpecs?.length && (
          <VideoSpecPanel
            key={props.model.model_name}
            specs={videoSpecs}
            anonymousCallPricing={
              isSeedancePerCall &&
              videoSpecs.every((spec) =>
                [
                  ...(spec.without_reference ?? []),
                  ...(spec.with_reference ?? []),
                ].every((rate) => rate.unit === 'request')
              )
            }
            priceOptions={dynamicPriceOptions}
          />
        )}
        {!!imageSpecs?.length && (
          <ImageSpecPanel
            key={props.model.model_name}
            specs={imageSpecs}
            priceOptions={dynamicPriceOptions}
          />
        )}
        {imageFallback && (
          <div className='bg-muted/30 min-w-0 rounded-xl border px-3 py-2'>
            {pricing}
          </div>
        )}
        <div
          className={cn(
            'flex min-w-0 flex-col gap-1.5',
            isDirectory && 'stolas-model-description'
          )}
        >
          <p
            className={cn(
              'text-muted-foreground text-[13px] leading-5 break-words',
              isDirectory && !inPanel ? 'line-clamp-3' : 'line-clamp-2',
              inPanel && 'text-xs leading-4'
            )}
          >
            {props.model.description || t('No description available.')}
          </p>
          {(isDirectory || hasSpecs) && promotionNote}
          {tags.length > 0 && (
            <div
              role='group'
              aria-label={t('Tags')}
              className='text-muted-foreground flex min-w-0 items-baseline gap-1.5 text-xs'
            >
              <span className='shrink-0'>{t('Tags')}</span>
              <span className='truncate' title={tags.join(', ')}>
                {tags.slice(0, 2).join(', ')}
              </span>
              {tags.length > 2 && (
                <span className='shrink-0' title={tags.slice(2).join(', ')}>
                  +{tags.length - 2}
                </span>
              )}
            </div>
          )}
        </div>
        {!inPanel && pricing}
        {!isDirectory && (groups.length > 0 || endpoints.length > 0) && (
          <dl
            className={cn(
              'grid min-w-0 grid-cols-2 gap-3 text-xs',
              (groups.length === 0 || endpoints.length === 0) && 'grid-cols-1'
            )}
          >
            {groups.length > 0 && (
              <div className='flex min-w-0 items-baseline gap-1.5'>
                <dt className='text-muted-foreground shrink-0'>
                  {t('Groups')}
                </dt>
                <dd className='flex min-w-0 items-baseline gap-1'>
                  <span className='truncate' title={groups.join(', ')}>
                    {groups[0]}
                  </span>
                  {groups.length > 1 && (
                    <span
                      className='text-muted-foreground shrink-0'
                      title={groups.slice(1).join(', ')}
                    >
                      +{groups.length - 1}
                    </span>
                  )}
                </dd>
              </div>
            )}
            {endpoints.length > 0 && (
              <div className='flex min-w-0 items-baseline gap-1.5'>
                <dt className='text-muted-foreground shrink-0'>
                  {t('Endpoints')}
                </dt>
                <dd className='flex min-w-0 items-baseline gap-1'>
                  <span className='truncate' title={endpoints.join(', ')}>
                    {endpoints.slice(0, 2).join(', ')}
                  </span>
                  {endpoints.length > 2 && (
                    <span
                      className='text-muted-foreground shrink-0'
                      title={endpoints.slice(2).join(', ')}
                    >
                      +{endpoints.length - 2}
                    </span>
                  )}
                </dd>
              </div>
            )}
          </dl>
        )}
      </CardContent>
      <CardFooter
        className={cn(
          'mt-auto border-0 bg-transparent pt-0',
          isDirectory && 'stolas-model-footer'
        )}
      >
        {isDirectory ? (
          <ModelPerfBadge directory perf={props.perf}>
            {detailsAction}
          </ModelPerfBadge>
        ) : (
          <ModelPerfBadge
            perf={props.perf}
            className='border-border/60 border-t pt-2'
          >
            {detailsAction}
          </ModelPerfBadge>
        )}
      </CardFooter>
    </Card>
  )
})
