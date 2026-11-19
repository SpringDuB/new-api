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
import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { searchChannels } from '@/features/channels/api'
import {
  CHANNEL_STATUS,
  CHANNEL_STATUS_LABELS,
} from '@/features/channels/constants'
import { formatLatency } from '@/features/performance-metrics/lib/format'
import { toIntlLocale } from '@/i18n/languages'
import { ROLE } from '@/lib/roles'
import { requireServerSuccess } from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import {
  SEEDANCE_PAGE_SIZE,
  SeedanceHealthPagination,
} from './seedance-health-pagination'

export function SeedanceChannelStatus({
  page,
  onPageChange,
}: {
  page: number
  onPageChange: (page: number) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const user = useAuthStore((state) => state.auth.user)
  const isAdmin = (user?.role ?? 0) >= ROLE.ADMIN
  const query = useQuery({
    queryKey: ['stolas', 'seedance-channels', user?.id, user?.role, page],
    queryFn: async () =>
      requireServerSuccess(
        await searchChannels({
          model: 'seedance',
          p: page,
          page_size: SEEDANCE_PAGE_SIZE,
        })
      ),
    enabled: isAdmin,
    placeholderData: (previous, previousQuery) => {
      const previousKey = previousQuery?.queryKey
      if (previousKey?.[2] === user?.id && previousKey?.[3] === user?.role) {
        return previous
      }
    },
    staleTime: 60_000,
    refetchInterval: 60_000,
  })
  const channels = query.data?.data?.items ?? []
  const total = query.data?.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / SEEDANCE_PAGE_SIZE))
  const pageOutOfRange =
    !!query.data && !query.isPlaceholderData && page > pageCount
  useEffect(() => {
    if (isAdmin && pageOutOfRange) onPageChange(pageCount)
  }, [isAdmin, pageOutOfRange, pageCount, onPageChange])

  if (!isAdmin) return null
  if (query.isPending || pageOutOfRange) {
    return (
      <div
        role='status'
        aria-label={t('Loading channel status')}
        className='stolas-health-loading'
      >
        <Skeleton className='h-20 w-full' />
        <Skeleton className='h-20 w-full' />
      </div>
    )
  }
  if (query.isError) {
    return (
      <>
        <ErrorState
          title={t('Unable to load channel status')}
          onRetry={() => void query.refetch()}
        />
        {page > 1 && (
          <Button
            variant='outline'
            size='sm'
            onClick={() => onPageChange(page - 1)}
          >
            {t('Previous page')}
          </Button>
        )}
      </>
    )
  }

  return (
    <div>
      <SeedanceHealthPagination
        label={t('Channel status')}
        page={page}
        pageCount={pageCount}
        onPageChange={onPageChange}
      />
      <p className='stolas-health-note'>
        {t(
          'Channel state and saved test results are shown separately from model completion metrics.'
        )}
      </p>
      {channels.length === 0 && (
        <EmptyState title={t('No Seedance channels configured')} />
      )}
      <div
        aria-busy={query.isPlaceholderData}
        className={query.isPlaceholderData ? 'opacity-60' : undefined}
      >
        {channels.map((channel) => {
          const hasSeedanceTest =
            channel.test_time > 0 && /seedance/i.test(channel.test_model ?? '')
          const statusKey =
            CHANNEL_STATUS_LABELS[
              channel.status as keyof typeof CHANNEL_STATUS_LABELS
            ] ?? 'Unknown'
          return (
            <section
              key={channel.id}
              className='stolas-health-channel'
              aria-label={channel.name}
            >
              <div className='stolas-health-model-name'>
                <h3>{channel.name}</h3>
                <p>
                  #{channel.id} ·{' '}
                  {
                    channel.models
                      .split(',')
                      .filter((model) => /seedance/i.test(model)).length
                  }{' '}
                  {t('models')}
                </p>
              </div>
              <div
                className='stolas-health-channel-state'
                data-enabled={channel.status === CHANNEL_STATUS.ENABLED}
              >
                <span aria-hidden='true' />
                {t(statusKey)}
              </div>
              <dl className='stolas-health-model-metrics'>
                <div>
                  <dt>{t('Last Seedance test')}</dt>
                  <dd>
                    {hasSeedanceTest
                      ? new Date(channel.test_time * 1000).toLocaleString(
                          locale
                        )
                      : t('No Seedance test')}
                  </dd>
                </div>
                <div>
                  <dt>{t('Test latency')}</dt>
                  <dd>
                    {hasSeedanceTest
                      ? formatLatency(channel.response_time)
                      : '—'}
                  </dd>
                </div>
              </dl>
            </section>
          )
        })}
      </div>
      <footer className='stolas-health-channel-footer'>
        <Button
          variant='ghost'
          size='sm'
          render={<Link to='/channels' />}
          nativeButton={false}
        >
          {t('Manage channels')}
        </Button>
        <span>{t('{{count}} channels', { count: total })}</span>
      </footer>
    </div>
  )
}
