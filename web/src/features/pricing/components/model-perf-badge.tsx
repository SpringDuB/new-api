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
import { memo, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  formatLatency,
  formatThroughput,
  formatUptimePct,
  getSuccessRateDotClass,
} from '@/features/performance-metrics/lib/format'
import type { SuccessRatePoint } from '@/features/performance-metrics/types'
import { toIntlLocale } from '@/i18n/languages'
import { cn } from '@/lib/utils'

export type ModelPerfBadgeData = {
  window_start?: number
  window_end?: number
  avg_latency_ms: number
  success_rate: number
  avg_tps: number
  recent_success_series?: SuccessRatePoint[]
}

export interface ModelPerfBadgeProps extends React.HTMLAttributes<HTMLDivElement> {
  directory?: boolean
  perf: ModelPerfBadgeData | undefined
}

const STATUS_SAMPLE_LIMIT = 24

export const ModelPerfBadge = memo(function ModelPerfBadge(
  props: ModelPerfBadgeProps
) {
  const { t, i18n } = useTranslation()
  const latencyText = formatLatency(props.perf?.avg_latency_ms ?? 0)
  const throughputText = formatThroughput(props.perf?.avg_tps ?? 0).replace(
    ' t/s',
    't/s'
  )
  const successRate = props.perf?.success_rate
  const hasSuccessRate =
    successRate != null &&
    Number.isFinite(successRate) &&
    successRate >= 0 &&
    successRate <= 100
  const performanceLabel = t('Performance metrics for the last 24 hours')
  const successRateText = hasSuccessRate ? formatUptimePct(successRate) : '—'
  const successRateHint = t(
    'Success rate excludes business rejections and includes the current partial hour.'
  )
  const statusSamples = useMemo(
    () =>
      (props.perf?.recent_success_series ?? [])
        .filter(
          (point) =>
            Number.isFinite(point.ts) &&
            Number.isFinite(point.success_rate) &&
            point.success_rate >= 0 &&
            point.success_rate <= 100
        )
        .sort((a, b) => a.ts - b.ts)
        .slice(-STATUS_SAMPLE_LIMIT),
    [props.perf?.recent_success_series]
  )
  const statusLabel =
    statusSamples.length > 0
      ? t('Success rate')
      : t('No performance data available')

  if (props.directory) {
    const emptySlots = STATUS_SAMPLE_LIMIT - statusSamples.length
    const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
    const clock = new Intl.DateTimeFormat(locale, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
    return (
      <div
        role='group'
        aria-label={t('Success rate over the latest active hours')}
        className='stolas-model-health'
      >
        <div className='text-muted-foreground flex items-baseline gap-1.5 text-xs'>
          {t('Success rate')} <span aria-hidden='true'>·</span> {t('7 Days')}
        </div>
        <strong className='text-base font-semibold tabular-nums'>
          {successRateText}
        </strong>
        <div
          className='stolas-model-hours'
          role='img'
          aria-label={t('Latest active hours')}
        >
          {Array.from(
            { length: emptySlots },
            (_, i) => `empty-${emptySlots - i}`
          ).map((key) => (
            <span key={key} data-observed='false' />
          ))}
          {statusSamples.map(({ ts, success_rate }) => {
            let color = 'bg-red-500'
            if (success_rate >= 90) color = 'bg-emerald-500'
            else if (success_rate >= 50) color = 'bg-amber-500'
            return (
              <span
                key={ts}
                data-observed='true'
                className={color}
                title={`${clock.format(ts * 1000)} · ${formatUptimePct(success_rate)}`}
              />
            )
          })}
        </div>
        <div className='text-muted-foreground flex justify-between gap-2 text-[11px]'>
          <span>{t('Latest 24 hours with requests')}</span>
          <span>{t('Most recent')}</span>
        </div>
        <div className='mt-0.5 flex items-center justify-between gap-2'>
          <Popover>
            <PopoverTrigger
              openOnHover
              className='text-muted-foreground hover:text-foreground focus-visible:outline-ring rounded-sm py-1 text-xs underline decoration-dotted underline-offset-4'
            >
              {t('How this is calculated')}
            </PopoverTrigger>
            <PopoverContent
              align='start'
              side='top'
              aria-label={t('How this is calculated')}
              className='bg-foreground text-background w-72 max-w-[calc(100vw-2rem)] px-3 py-2 text-xs leading-relaxed'
            >
              {t(
                'Hourly success: green ≥90%, yellow 50–<90%, red <50%. Gray means no valid samples. Identified policy, parameter and media rejections do not count as failures.'
              )}
            </PopoverContent>
          </Popover>
          {props.children}
        </div>
      </div>
    )
  }

  return (
    <div
      role='group'
      aria-label={performanceLabel}
      className={cn(
        'flex w-full min-w-0 items-center justify-between gap-3',
        props.className
      )}
    >
      <dl className='flex min-w-0 items-start gap-5 text-xs tabular-nums'>
        <div className='w-24 shrink-0'>
          <dt
            title={successRateHint}
            className='text-muted-foreground flex items-center justify-between gap-1 text-[11px] leading-4'
          >
            <span>{t('Status')}</span>
            <span className='font-mono'>{successRateText}</span>
          </dt>
          <dd
            role='img'
            aria-label={statusLabel}
            title={statusLabel}
            className='mt-1 flex h-3 w-24 items-center gap-px'
          >
            {statusSamples.length > 0 ? (
              statusSamples.map((sample) => (
                <span
                  key={sample.ts}
                  aria-hidden
                  className={cn(
                    'h-full w-[3px] shrink-0 rounded-xs',
                    getSuccessRateDotClass(sample.success_rate)
                  )}
                />
              ))
            ) : (
              <span className='text-muted-foreground'>—</span>
            )}
          </dd>
        </div>
        <div title={t('Average latency')} className='shrink-0'>
          <dt className='text-muted-foreground text-[11px] leading-4'>
            {t('Latency short')}
          </dt>
          <dd className='mt-1 font-mono whitespace-nowrap'>
            {latencyText === '—' ? '—s' : latencyText}
          </dd>
        </div>
        <div title={t('Throughput')} className='shrink-0'>
          <dt className='text-muted-foreground text-[11px] leading-4'>
            {t('Throughput short')}
          </dt>
          <dd className='mt-1 font-mono whitespace-nowrap'>
            {throughputText === '—' ? '—t/s' : throughputText}
          </dd>
        </div>
      </dl>
      {props.children}
    </div>
  )
})
