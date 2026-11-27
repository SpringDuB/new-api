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
import { ImageIcon, Music2, ScanFace, Video } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { resolveLocalizedText } from '@/lib/localized-text'

import {
  formatTaskUsageUnitPrice,
  type DynamicPriceOptions,
} from '../lib/dynamic-price'
import { resolutionHeight } from '../lib/model-helpers'
import type {
  CatalogVideoSpec,
  VideoInputCombination,
  VideoUnitRate,
} from '../types'
import { SpecOptions } from './spec-options'

function MaterialLimits({
  combination,
}: {
  combination: VideoInputCombination
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.language)
  const {
    maxima,
    minima,
    total,
    videoSeconds,
    videoTotalSeconds,
    videoWithinOutput,
    nativeFace,
  } = combination
  const sum = maxima.reduce((n, count) => n + count, 0)
  const materials = [
    { kind: 'image', label: t('Images'), Icon: ImageIcon },
    { kind: 'video', label: t('Videos'), Icon: Video },
    { kind: 'audio', label: t('Audio'), Icon: Music2 },
  ]
  return (
    <div className='flex flex-wrap items-center gap-1'>
      {sum === 0 && <Badge variant='outline'>{t('No reference files')}</Badge>}
      {materials.map(({ kind, label, Icon }, index) => {
        const max = maxima[index]
        if (!max) return null
        const min = minima[index]
        const upper = formatNumber(max, locale)
        let range = `≤${upper}`
        if (min === max) range = upper
        else if (min > 0) range = `${formatNumber(min, locale)}–${upper}`
        return (
          <Badge key={kind} variant='outline' className='gap-1 font-normal'>
            <Icon aria-hidden='true' />
            {label} {range}
          </Badge>
        )
      })}
      {total < sum && (
        <Badge variant='outline'>
          {t('Total ≤{{limit}}', { limit: formatNumber(total, locale) })}
        </Badge>
      )}
      {!!videoSeconds && (
        <Badge variant='outline'>
          {t('≤{{seconds}} s per clip', {
            seconds: formatNumber(videoSeconds, locale),
          })}
        </Badge>
      )}
      {!!videoTotalSeconds && (
        <Badge variant='outline'>
          {t('≤{{seconds}} s of video in total', {
            seconds: formatNumber(videoTotalSeconds, locale),
          })}
        </Badge>
      )}
      {videoWithinOutput && (
        <Badge variant='outline' className='h-auto whitespace-normal'>
          {t('Reference duration ≤ output duration')}
        </Badge>
      )}
      {nativeFace && (
        <Badge
          variant='outline'
          className='border-emerald-600/30 bg-emerald-500/5 text-emerald-700 dark:text-emerald-300'
        >
          <ScanFace aria-hidden='true' />
          {t('Native face support')}
        </Badge>
      )}
    </div>
  )
}

// Anonymous per-call quotes must cover every active route for this resolution.
function callPriceRange(
  rows: CatalogVideoSpec[],
  mode: 'without_reference' | 'with_reference',
  options: DynamicPriceOptions
) {
  const amounts = rows.flatMap((row) =>
    (row[mode] ?? [])
      .filter((rate) => rate.unit === 'request')
      .map((rate) => rate.amount)
  )
  if (!amounts.length) return undefined
  const min = formatTaskUsageUnitPrice(Math.min(...amounts), options)
  const max = formatTaskUsageUnitPrice(Math.max(...amounts), options)
  return min === max ? min : `${min} – ${max}`
}

export function VideoSpecPanel({
  specs,
  anonymousCallPricing = false,
  priceOptions,
}: {
  anonymousCallPricing?: boolean
  specs: CatalogVideoSpec[]
  priceOptions: DynamicPriceOptions
}) {
  const { t, i18n } = useTranslation()
  const [resolution, setResolution] = useState<string>()
  const [provider, setProvider] = useState<string>()
  const [duration, setDuration] = useState<string>()
  const resolutions = [...new Set(specs.map((s) => s.resolution))].sort(
    (a, b) => resolutionHeight(a) - resolutionHeight(b)
  )
  const activeResolution =
    resolution && resolutions.includes(resolution) ? resolution : resolutions[0]
  const rows = specs.filter((s) => s.resolution === activeResolution)
  const providers = [
    ...new Set(
      rows.map((s) => s.provider).filter((name): name is string => !!name)
    ),
  ]
  const activeProvider =
    provider && providers.includes(provider) ? provider : providers[0]
  const bands = rows.filter((s) => s.provider === activeProvider)
  const spec = bands.find((s) => s.duration_tier === duration) ?? bands[0]
  // Without a provider selector, list distinct supported combinations as alternatives.
  // Taking separate maxima would advertise input combinations no route accepts.
  const combinations =
    anonymousCallPricing && rows.length > 1
      ? [
          ...new Map(
            rows
              .flatMap((row) => row.combinations)
              .map((c) => [JSON.stringify(c), c])
          ).values(),
        ]
      : spec.combinations
  const durationLabel = (row: CatalogVideoSpec) =>
    resolveLocalizedText(row.duration_label, i18n.language) || row.duration_tier
  const plain = spec.without_reference?.[0]
  const withVideo = spec.with_reference?.[0]
  const referenceRate = spec.with_reference?.find(
    (rate) => rate.unit === 'reference_second'
  )
  const imageRate = [
    ...(spec.without_reference ?? []),
    ...(spec.with_reference ?? []),
  ].find((r) => r.unit === 'reference_image')
  const primary = plain ?? withVideo
  const differentOutput =
    !!plain &&
    !!withVideo &&
    (plain.unit !== withVideo.unit || plain.amount !== withVideo.amount)
  const unitLabels: Record<VideoUnitRate['unit'], string> = {
    request: t('call'),
    second: t('second'),
    reference_image: t('image'),
    reference_second: t('second'),
    video_tokens: t('1M tokens'),
  }
  const plainPrice = anonymousCallPricing
    ? callPriceRange(rows, 'without_reference', priceOptions)
    : plain && formatTaskUsageUnitPrice(plain.amount, priceOptions)
  const videoPrice = anonymousCallPricing
    ? callPriceRange(rows, 'with_reference', priceOptions)
    : withVideo && formatTaskUsageUnitPrice(withVideo.amount, priceOptions)
  const showSeparatePrices = anonymousCallPricing
    ? !!plainPrice && !!videoPrice && plainPrice !== videoPrice
    : differentOutput
  return (
    <div className='grid min-w-0 gap-2'>
      <div className='stolas-video-spec-box bg-muted/30 min-w-0 rounded-xl border px-3 py-2'>
        <SpecOptions
          label={t('Output resolution')}
          values={resolutions}
          active={spec.resolution}
          onChange={setResolution}
        />
        {!anonymousCallPricing && providers.length > 1 && (
          <ToggleGroup
            aria-label={t('Provider')}
            value={activeProvider ? [activeProvider] : []}
            onValueChange={([value]) => {
              if (value) setProvider(value)
            }}
            size='sm'
            spacing={1}
            className='mt-1 flex-wrap'
          >
            {providers.map((name) => (
              <ToggleGroupItem key={name} value={name}>
                {name}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        )}
        {!anonymousCallPricing &&
          (bands.length > 1 ? (
            <ToggleGroup
              aria-label={t('Duration')}
              value={spec.duration_tier ? [spec.duration_tier] : []}
              onValueChange={([value]) => {
                if (value) setDuration(value)
              }}
              size='sm'
              spacing={1}
              className='mt-1 flex-wrap'
            >
              {bands.map((row) => (
                <ToggleGroupItem
                  key={row.duration_tier}
                  value={row.duration_tier ?? ''}
                >
                  {durationLabel(row)}
                </ToggleGroupItem>
              ))}
            </ToggleGroup>
          ) : (
            spec.duration_tier && (
              <span className='text-muted-foreground text-xs'>
                {durationLabel(spec)}
              </span>
            )
          ))}
        <div
          role='group'
          aria-label={t('Pricing')}
          aria-live='polite'
          aria-atomic='true'
          className='mt-1.5 grid gap-1 border-t pt-1.5 text-xs tabular-nums'
        >
          {primary && (
            <div>
              {showSeparatePrices && (
                <span className='text-muted-foreground me-1'>
                  {t('Without reference video')}
                </span>
              )}
              {!plainPrice && (
                <span className='text-muted-foreground me-1'>
                  {t('With reference video')}
                </span>
              )}
              <strong className='text-sm'>{plainPrice ?? videoPrice}</strong>
              <span className='text-muted-foreground'>
                {' '}
                / {unitLabels[primary.unit]}
              </span>
            </div>
          )}
          {showSeparatePrices && (
            <div>
              <span className='text-muted-foreground me-1'>
                {t('With reference video')}
              </span>
              <strong>{videoPrice}</strong> /{' '}
              {anonymousCallPricing
                ? unitLabels.request
                : withVideo && unitLabels[withVideo.unit]}
            </div>
          )}
          {referenceRate && referenceRate.amount > 0 && (
            <div className='text-muted-foreground'>
              {t('Reference video input')}{' '}
              <span className='text-foreground'>
                +{formatTaskUsageUnitPrice(referenceRate.amount, priceOptions)}
              </span>{' '}
              / {t('second')}
            </div>
          )}
          {imageRate && (
            <div className='text-muted-foreground'>
              {t('After {{count}} images', { count: spec.includedImages ?? 0 })}{' '}
              +{formatTaskUsageUnitPrice(imageRate.amount, priceOptions)} /{' '}
              {t('image')}
            </div>
          )}
        </div>
      </div>
      {!anonymousCallPricing && spec.requiresSourceTask && (
        <Badge variant='outline'>{t('Completed source task required')}</Badge>
      )}
      {combinations.length > 0 && (
        <ul aria-label={t('Reference material limits')} className='grid gap-1'>
          {combinations.map((combination, index) => (
            <li key={JSON.stringify(combination)}>
              {index > 0 && (
                <span className='text-muted-foreground mb-1 block text-xs'>
                  {t('or')}
                </span>
              )}
              <MaterialLimits combination={combination} />
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
