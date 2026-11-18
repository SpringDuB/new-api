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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowUpRight, RefreshCw } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { getPerfMetricsSummary } from '@/features/performance-metrics/api'
import {
  formatLatency,
  formatUptimePct,
  getSuccessRateDotClass,
} from '@/features/performance-metrics/lib/format'
import type { PerfModelSummary } from '@/features/performance-metrics/types'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { requireServerSuccess } from '@/lib/server-error-message'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

import '@/styles/stolas-health.css'

import { SeedanceChannelStatus } from './seedance-channel-status'

const HOURS = 24
const INITIAL_MODEL_COUNT = 6

interface SeedanceHealthPanelProps {
  models: string[]
  loading?: boolean
  error?: boolean
  onRetry?: () => void
}

function SeedanceModelActivity(props: {
  name: string
  metrics?: PerfModelSummary
  windowEnd?: number
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const metrics = props.metrics
  const hasMetrics =
    metrics != null &&
    (metrics.request_count == null || metrics.request_count > 0)
  const latestHour = Math.floor((props.windowEnd ?? 0) / 3600) * 3600
  const points = new Map(
    (metrics?.recent_success_series ?? [])
      .filter(
        (point) =>
          Number.isFinite(point.success_rate) &&
          point.success_rate >= 0 &&
          point.success_rate <= 100
      )
      .map((point) => [Math.floor(point.ts / 3600) * 3600, point.success_rate])
  )
  const slots = Array.from({ length: HOURS }, (_, index) => {
    const ts = latestHour - (HOURS - index - 1) * 3600
    return { ts, rate: points.get(ts) }
  })
  const observedHours = slots.filter((slot) => slot.rate != null).length
  const titleId = `seedance-${encodeURIComponent(props.name)}`

  return (
    <section className='stolas-health-model' aria-labelledby={titleId}>
      <div className='stolas-health-model-name'>
        <h3 id={titleId}>{props.name}</h3>
        <p>{hasMetrics ? t('Recorded activity') : t('No recent requests')}</p>
      </div>
      <div className='stolas-health-history'>
        <div
          className='stolas-health-slots'
          role='img'
          aria-label={t('{{count}} of 24 hours have recorded requests', {
            count: observedHours,
          })}
        >
          {slots.map((slot) => (
            <span
              key={slot.ts}
              data-observed={slot.rate != null}
              className={cn(
                'stolas-health-slot',
                slot.rate != null && getSuccessRateDotClass(slot.rate)
              )}
              title={
                slot.rate == null
                  ? t('No recorded requests')
                  : `${new Date(slot.ts * 1000).toLocaleTimeString(locale, {
                      hour: '2-digit',
                      minute: '2-digit',
                    })} · ${formatUptimePct(slot.rate)}`
              }
            />
          ))}
        </div>
        <span className='stolas-health-history-caption'>
          {t('Past 24 hours')}
        </span>
      </div>
      <dl className='stolas-health-model-metrics'>
        <div>
          <dt>{t('Success rate')}</dt>
          <dd>{hasMetrics ? formatUptimePct(metrics.success_rate) : '—'}</dd>
        </div>
        <div>
          <dt>{t('Average completion time')}</dt>
          <dd>{hasMetrics ? formatLatency(metrics.avg_latency_ms) : '—'}</dd>
        </div>
        <div>
          <dt>{t('Observed requests')}</dt>
          <dd>
            {metrics?.request_count == null
              ? '—'
              : formatNumber(metrics.request_count, locale)}
          </dd>
        </div>
      </dl>
    </section>
  )
}

export function SeedanceHealthPanel(props: SeedanceHealthPanelProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const user = useAuthStore((state) => state.auth.user)
  const isAdmin = (user?.role ?? 0) >= ROLE.ADMIN
  const [selectedTab, setSelectedTab] = useState('models')
  const [showAll, setShowAll] = useState(false)
  const activeTab = isAdmin ? selectedTab : 'models'
  const modelNames = useMemo(
    () =>
      [
        ...new Set(props.models.filter((name) => /seedance/i.test(name))),
      ].sort(),
    [props.models]
  )
  const metricsQuery = useQuery({
    queryKey: ['stolas', 'seedance-health', user?.id, HOURS],
    queryFn: async () =>
      requireServerSuccess(await getPerfMetricsSummary(HOURS)),
    enabled: !props.loading && !props.error && modelNames.length > 0,
    staleTime: 60_000,
    refetchInterval: 60_000,
  })
  const metricsByName = useMemo(
    () =>
      new Map(
        (metricsQuery.data?.data.models ?? []).map((item) => [
          item.model_name,
          item,
        ])
      ),
    [metricsQuery.data]
  )
  const orderedNames = useMemo(
    () =>
      [...modelNames].sort(
        (a, b) =>
          (metricsByName.get(b)?.request_count ?? 0) -
            (metricsByName.get(a)?.request_count ?? 0) || a.localeCompare(b)
      ),
    [modelNames, metricsByName]
  )
  const visibleNames = showAll
    ? orderedNames
    : orderedNames.slice(0, INITIAL_MODEL_COUNT)
  const loading =
    !!props.loading ||
    (!props.error && modelNames.length > 0 && metricsQuery.isPending)
  const failed = !!props.error || metricsQuery.isError
  const refresh = () => {
    if (activeTab === 'channels') {
      void queryClient.invalidateQueries({
        queryKey: ['stolas', 'seedance-channels', user?.id, user?.role],
      })
      return
    }
    props.onRetry?.()
    if (modelNames.length > 0 && !props.loading && !props.error) {
      void metricsQuery.refetch()
    }
  }

  return (
    <section className='stolas-health' aria-labelledby='stolas-health-title'>
      <header className='stolas-health-heading'>
        <div>
          <p className='stolas-eyebrow'>{t('Model availability')}</p>
          <h2 className='stolas-editorial' id='stolas-health-title'>
            {t('Seedance model health')}
          </h2>
          <p>
            {t(
              'Recent model activity and the routes behind your video generations.'
            )}
          </p>
        </div>
        <Button
          variant='ghost'
          size='sm'
          render={<Link to='/pricing' />}
          nativeButton={false}
        >
          {t('Model library')}
          <ArrowUpRight aria-hidden='true' />
        </Button>
      </header>
      <Tabs
        value={activeTab}
        onValueChange={setSelectedTab}
        className='stolas-health-tabs'
      >
        <div className='stolas-health-controls'>
          <TabsList
            variant='line'
            activateOnFocus
            aria-label={t('Seedance monitoring')}
          >
            <TabsTrigger value='models'>{t('Model performance')}</TabsTrigger>
            {isAdmin && (
              <TabsTrigger value='channels'>{t('Channel status')}</TabsTrigger>
            )}
          </TabsList>
          <Button
            variant='ghost'
            size='sm'
            disabled={
              metricsQuery.isFetching ||
              !!props.loading ||
              (activeTab === 'models' &&
                !props.error &&
                modelNames.length === 0)
            }
            onClick={refresh}
          >
            <RefreshCw
              aria-hidden='true'
              className={cn(metricsQuery.isFetching && 'animate-spin')}
            />
            {t('Refresh')}
          </Button>
        </div>
        <TabsContent value='models'>
          {loading && (
            <div
              role='status'
              aria-label={t('Loading Seedance activity')}
              className='stolas-health-loading'
            >
              <Skeleton className='h-20 w-full' />
              <Skeleton className='h-20 w-full' />
            </div>
          )}
          {!loading && failed && (
            <ErrorState
              title={t('Unable to load Seedance activity')}
              onRetry={refresh}
            />
          )}
          {!loading && !failed && modelNames.length === 0 && (
            <EmptyState
              title={t('No Seedance models available')}
              description={t(
                'Seedance models enabled for your account will appear here.'
              )}
            />
          )}
          {!loading && !failed && modelNames.length > 0 && (
            <>
              <div>
                {visibleNames.map((name) => (
                  <SeedanceModelActivity
                    key={name}
                    name={name}
                    metrics={metricsByName.get(name)}
                    windowEnd={metricsQuery.data?.data.window_end}
                  />
                ))}
              </div>
              {orderedNames.length > INITIAL_MODEL_COUNT && (
                <Button
                  variant='ghost'
                  className='stolas-health-more'
                  onClick={() => setShowAll(!showAll)}
                  aria-expanded={showAll}
                >
                  {showAll
                    ? t('Show fewer models')
                    : t('Show all {{count}} models', {
                        count: orderedNames.length,
                      })}
                </Button>
              )}
              <p className='stolas-health-note'>
                {t('Platform-wide observations for your available models.')}{' '}
                {t(
                  'Each segment is one hour. Gray means no requests were recorded. These are observed results, not a live availability guarantee.'
                )}
              </p>
            </>
          )}
        </TabsContent>
        {isAdmin && (
          <TabsContent value='channels'>
            <SeedanceChannelStatus />
          </TabsContent>
        )}
      </Tabs>
    </section>
  )
}
