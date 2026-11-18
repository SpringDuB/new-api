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
import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { getUserQuotaDates } from '@/features/dashboard/api'
import { useSummaryCardsConfig } from '@/features/dashboard/hooks/use-dashboard-config'
import { useStatus } from '@/hooks/use-status'
import { getCurrencyLabel, isCurrencyDisplayEnabled } from '@/lib/currency'
import { formatNumber, formatQuota } from '@/lib/format'
import { requireServerSuccess } from '@/lib/server-error-message'
import { computeTimeRange } from '@/lib/time'
import { useAuthStore } from '@/stores/auth-store'

export function SummaryCards() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const { status, loading } = useStatus()
  const timeRange = useMemo(() => computeTimeRange(1), [])
  const usageQuery = useQuery({
    queryKey: [
      'dashboard',
      'overview',
      'summary-sparklines',
      user?.id,
      timeRange.start_timestamp,
      timeRange.end_timestamp,
    ],
    queryFn: async () =>
      requireServerSuccess(
        await getUserQuotaDates({
          start_timestamp: timeRange.start_timestamp,
          end_timestamp: timeRange.end_timestamp,
          default_time: 'hour',
        })
      ),
    staleTime: 60 * 1000,
    enabled: Boolean(user),
  })
  const recentUsage = (usageQuery.data?.data ?? []).reduce(
    (total, item) => total + (Number(item.quota) || 0),
    0
  )
  const currencyEnabled =
    typeof status?.display_in_currency === 'boolean'
      ? status.display_in_currency
      : isCurrencyDisplayEnabled()
  const items = useSummaryCardsConfig({
    usedDisplay: formatQuota(Number(user?.used_quota ?? 0)),
    requestCountDisplay: formatNumber(Number(user?.request_count ?? 0)),
    todayUsageDisplay: formatQuota(recentUsage),
    currencyEnabled,
    currencyLabel: currencyEnabled ? getCurrencyLabel() : 'Tokens',
  })

  return (
    <section
      aria-labelledby='stolas-usage-title'
      className='stolas-usage-strip'
    >
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div className='flex flex-col gap-1'>
          <h3 id='stolas-usage-title' className='text-base font-semibold'>
            {t('Usage at a glance')}
          </h3>
          <p className='text-muted-foreground text-xs'>
            {t('Monitor balance, usage, and request volume')}
          </p>
        </div>
        <Button
          variant='ghost'
          size='sm'
          role='link'
          render={<Link to='/usage-logs' />}
        >
          {t('Usage Logs')} ↗
        </Button>
      </div>
      <dl className='stolas-usage-metrics'>
        {items.map((item) => {
          const pending =
            loading || (item.key === 'todayUsage' && usageQuery.isPending)
          const failed = item.key === 'todayUsage' && usageQuery.isError
          let valueContent: ReactNode = item.value
          if (failed) valueContent = '—'
          if (pending) valueContent = <Skeleton className='h-8 w-24' />
          return (
            <div
              key={item.key}
              className='stolas-usage-metric'
              aria-busy={pending}
            >
              <dt className='text-muted-foreground text-xs'>{item.title}</dt>
              <dd className='mt-3 font-mono text-2xl font-medium break-all tabular-nums'>
                {valueContent}
              </dd>
              <p className='text-muted-foreground mt-2 text-[11px]'>
                {failed ? t('Unable to load usage') : item.description}
              </p>
            </div>
          )
        })}
        <div className='stolas-usage-metric'>
          <dt className='text-muted-foreground text-xs'>
            {t('Credit remaining')}
          </dt>
          <dd className='stolas-editorial mt-3 text-3xl break-all tabular-nums'>
            {loading ? (
              <Skeleton className='h-9 w-24' />
            ) : (
              formatQuota(Number(user?.quota ?? 0))
            )}
          </dd>
          <Button
            variant='link'
            size='sm'
            role='link'
            className='mt-1 h-auto p-0 text-xs'
            render={<Link to='/wallet' />}
          >
            {t('Wallet')} ↗
          </Button>
        </div>
      </dl>
    </section>
  )
}
