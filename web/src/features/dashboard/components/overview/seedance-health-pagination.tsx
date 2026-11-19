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
import { ArrowLeft01Icon, ArrowRight01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Pagination,
  PaginationContent,
  PaginationItem,
} from '@/components/ui/pagination'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

export const SEEDANCE_PAGE_SIZE = 6

export function SeedanceHealthPagination(props: {
  label: string
  page: number
  pageCount: number
  onPageChange: (page: number) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  if (props.pageCount <= 1) return null

  return (
    <Pagination aria-label={props.label} className='stolas-health-pagination'>
      <PaginationContent className='gap-2'>
        <PaginationItem>
          <Button
            variant='outline'
            size='sm'
            aria-label={t('Previous page')}
            disabled={props.page <= 1}
            onClick={() => props.onPageChange(props.page - 1)}
          >
            <HugeiconsIcon icon={ArrowLeft01Icon} aria-hidden='true' />
            <span className='hidden sm:inline'>{t('Previous page')}</span>
          </Button>
        </PaginationItem>
        <PaginationItem>
          <span
            role='status'
            className='text-muted-foreground px-1 text-xs tabular-nums'
          >
            {t('Page {{current}} of {{total}}', {
              current: formatNumber(props.page, locale),
              total: formatNumber(props.pageCount, locale),
            })}
          </span>
        </PaginationItem>
        <PaginationItem>
          <Button
            variant='outline'
            size='sm'
            aria-label={t('Next page')}
            disabled={props.page >= props.pageCount}
            onClick={() => props.onPageChange(props.page + 1)}
          >
            <span className='hidden sm:inline'>{t('Next page')}</span>
            <HugeiconsIcon icon={ArrowRight01Icon} aria-hidden='true' />
          </Button>
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  )
}
