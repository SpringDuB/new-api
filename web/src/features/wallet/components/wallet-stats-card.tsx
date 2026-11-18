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
import { useTranslation } from 'react-i18next'

import { StolasLogo } from '@/components/stolas-logo'
import { Skeleton } from '@/components/ui/skeleton'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatQuota } from '@/lib/format'

import type { UserWalletData } from '../types'

interface WalletStatsCardProps {
  user: UserWalletData | null
  loading?: boolean
}

export function WalletStatsCard(props: WalletStatsCardProps) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const busy = !!props.loading || !props.user
  return (
    <section
      aria-label={t('Balance overview')}
      aria-busy={busy}
      className='stolas-wallet-summary flex flex-col gap-8'
    >
      <div className='flex items-start justify-between gap-4'>
        <p className='text-muted-foreground text-sm'>{t('Current Balance')}</p>
        <StolasLogo variant='mark' className='size-9' />
      </div>
      <div>
        {busy || !props.user ? (
          <Skeleton className='h-14 w-44' />
        ) : (
          <h3 className='stolas-editorial text-5xl break-all tabular-nums sm:text-6xl'>
            {formatQuota(props.user.quota)}
          </h3>
        )}
        <p className='text-muted-foreground mt-3 max-w-xs text-xs leading-relaxed'>
          {t('Available across your Stolas workspace.')}
        </p>
      </div>
      <dl className='border-border grid grid-cols-2 gap-4 border-t pt-6'>
        <div className='flex min-w-0 flex-col gap-2'>
          <dt className='text-muted-foreground text-xs'>{t('Total Usage')}</dt>
          <dd className='font-mono text-lg break-all tabular-nums'>
            {busy || !props.user ? (
              <Skeleton className='h-6 w-20' />
            ) : (
              formatQuota(props.user.used_quota)
            )}
          </dd>
        </div>
        <div className='flex min-w-0 flex-col gap-2'>
          <dt className='text-muted-foreground text-xs'>{t('API Requests')}</dt>
          <dd className='font-mono text-lg break-all tabular-nums'>
            {busy || !props.user ? (
              <Skeleton className='h-6 w-20' />
            ) : (
              formatNumber(props.user.request_count, locale)
            )}
          </dd>
        </div>
      </dl>
    </section>
  )
}
