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

import { EmptyState } from '@/components/empty-state'
import { Progress } from '@/components/ui/progress'
import {
  ModelLink,
  VendorLink,
} from '@/features/rankings/components/entity-links'
import type { ModelRanking, VendorRanking } from '@/features/rankings/types'
import { toIntlLocale } from '@/i18n/languages'
import { formatCompactNumber, formatNumber } from '@/lib/format'

interface RankingListsProps {
  models: ModelRanking[]
  vendors: VendorRanking[]
}

export function RankingLists(props: RankingListsProps) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)

  return (
    <div className='grid gap-10 lg:grid-cols-[1.15fr_1fr] lg:gap-16'>
      <div className='min-w-0'>
        <div className='mb-3 flex items-baseline justify-between gap-4'>
          <h3 className='stolas-editorial text-2xl'>{t('Top Models')}</h3>
          <span className='text-muted-foreground text-xs'>{t('tokens')}</span>
        </div>
        {props.models.length === 0 ? (
          <EmptyState title={t('No model data available')} />
        ) : (
          <ol aria-label={t('Top Models')} className='divide-border divide-y'>
            {props.models.slice(0, 5).map((model) => (
              <li
                key={model.model_name}
                className='flex items-center gap-4 py-4'
              >
                <span className='text-muted-foreground w-6 shrink-0 font-mono text-xs tabular-nums'>
                  {formatNumber(model.rank, locale).padStart(2, '0')}
                </span>
                <div className='min-w-0 flex-1'>
                  <ModelLink
                    modelName={model.model_name}
                    className='block truncate text-sm font-medium'
                  />
                  <p className='text-muted-foreground mt-1 truncate text-xs'>
                    {model.vendor}
                  </p>
                </div>
                <span className='shrink-0 font-mono text-sm tabular-nums'>
                  {formatCompactNumber(model.total_tokens, locale)}
                </span>
              </li>
            ))}
          </ol>
        )}
      </div>
      <div className='min-w-0'>
        <div className='mb-3 flex items-baseline justify-between gap-4'>
          <h3 className='stolas-editorial text-2xl'>{t('Market Share')}</h3>
          <span className='text-muted-foreground text-xs'>
            {t('By model author')}
          </span>
        </div>
        {props.vendors.length === 0 ? (
          <EmptyState title={t('No vendor data available')} />
        ) : (
          <ol aria-label={t('Market Share')} className='divide-border divide-y'>
            {props.vendors.slice(0, 5).map((vendor) => (
              <li key={vendor.vendor} className='flex flex-col gap-3 py-4'>
                <div className='flex min-w-0 items-center justify-between gap-4 text-sm'>
                  <VendorLink
                    vendor={vendor.vendor}
                    className='truncate font-medium'
                  />
                  <span className='shrink-0 font-mono tabular-nums'>
                    {formatNumber(vendor.share * 100, locale)}%
                  </span>
                </div>
                <Progress
                  value={vendor.share * 100}
                  aria-label={vendor.vendor}
                />
              </li>
            ))}
          </ol>
        )}
      </div>
    </div>
  )
}
