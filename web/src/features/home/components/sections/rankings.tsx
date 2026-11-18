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
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { useRankings } from '@/features/rankings/hooks/use-rankings'
import type { RankingPeriod } from '@/features/rankings/types'
import { getModuleAccessFromStatus } from '@/lib/nav-modules'
import { statusQueryOptions } from '@/lib/status-query'
import { useAuthStore } from '@/stores/auth-store'

import { RankingLists } from '../ranking-lists'

export function HomeRankings() {
  const { t } = useTranslation()
  // Persisted branding placeholders may contain outdated access settings.
  const { data: status, error } = useQuery(statusQueryOptions)
  const user = useAuthStore((state) => state.auth.user)
  const access = getModuleAccessFromStatus(status ?? null, 'rankings')

  if (!status || error || !access.enabled) return null

  return (
    <section
      aria-label={t('Rankings')}
      className='mx-auto max-w-7xl px-6 pb-20 sm:px-10'
    >
      <Separator />
      <div className='flex flex-col gap-4 pt-12 pb-8'>
        <p className='stolas-eyebrow'>{t('Rankings')}</p>
        <h2 className='stolas-editorial text-3xl sm:text-4xl'>
          {t('The models people use.')}
        </h2>
        <p className='text-muted-foreground max-w-2xl text-sm leading-7'>
          {t(
            'Discover the most-used models and rising vendors on the platform, updated from live usage data.'
          )}
        </p>
      </div>
      {access.requireAuth && !user ? (
        <EmptyState
          title={t('Sign in to view rankings')}
          action={
            <Button
              variant='outline'
              role='link'
              render={<Link to='/sign-in' search={{ redirect: '/' }} />}
            >
              {t('Sign in')}
            </Button>
          }
        />
      ) : (
        <RankingsContent />
      )}
    </section>
  )
}

function RankingsContent() {
  const { t } = useTranslation()
  const [period, setPeriod] = useState<RankingPeriod>('week')
  const query = useRankings(period)
  const snapshot = query.data?.data
  let content: ReactNode

  if (query.isLoading) {
    content = (
      <div role='status'>
        <LoadingState message={t('Loading rankings...')} />
      </div>
    )
  } else if (!snapshot) {
    content = (
      <ErrorState
        title={t('Unable to load rankings')}
        description={t('Unable to load rankings data')}
        onRetry={() => {
          void query.refetch()
        }}
      />
    )
  } else if (snapshot.models.length === 0 && snapshot.vendors.length === 0) {
    content = (
      <EmptyState
        title={t('No ranking data yet')}
        description={t(
          'Rankings will appear when model traffic is recorded for this period.'
        )}
      />
    )
  } else {
    content = (
      <RankingLists models={snapshot.models} vendors={snapshot.vendors} />
    )
  }

  return (
    <div className='flex flex-col gap-8'>
      <ToggleGroup
        value={[period]}
        onValueChange={(values) => {
          const next = values[0]
          if (
            next === 'today' ||
            next === 'week' ||
            next === 'month' ||
            next === 'year'
          ) {
            setPeriod(next)
          }
        }}
        variant='outline'
        aria-label={t('Period')}
        className='max-w-full'
      >
        <ToggleGroupItem value='today'>{t('Today')}</ToggleGroupItem>
        <ToggleGroupItem value='week'>{t('Week')}</ToggleGroupItem>
        <ToggleGroupItem value='month'>{t('Month')}</ToggleGroupItem>
        <ToggleGroupItem value='year'>{t('Year')}</ToggleGroupItem>
      </ToggleGroup>
      {content}
    </div>
  )
}
