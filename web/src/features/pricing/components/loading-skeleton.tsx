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

import { Skeleton } from '@/components/ui/skeleton'

import { VIEW_MODES, type ViewMode } from '../constants'

export interface LoadingSkeletonProps {
  viewMode?: ViewMode
}

export function LoadingSkeleton(props: LoadingSkeletonProps) {
  const { t } = useTranslation()
  const isTable = props.viewMode === VIEW_MODES.TABLE
  return (
    <div role='status' aria-label={t('Loading...')} aria-busy='true'>
      <div className='stolas-catalog-header'>
        <div className='flex min-w-0 flex-col gap-4'>
          <Skeleton className='h-16 w-80 max-w-full' />
          <Skeleton className='h-5 w-full max-w-lg' />
        </div>
        <Skeleton className='h-11 w-full' />
      </div>
      <div className='stolas-catalog-layout'>
        <div className='stolas-catalog-filter-rail' aria-hidden='true'>
          <Skeleton className='mb-6 h-5 w-16' />
          <div className='flex flex-col gap-2'>
            {Array.from({ length: 10 }, (_, index) => (
              <Skeleton key={index} className='h-8 w-full' />
            ))}
          </div>
        </div>
        <div className='stolas-catalog-results'>
          <div className='flex flex-wrap gap-6 border-b py-4'>
            <Skeleton className='h-6 w-20' />
            <Skeleton className='h-6 w-20' />
            <Skeleton className='h-6 w-20' />
          </div>
          <div className='flex flex-wrap items-center justify-between gap-3 py-5'>
            <Skeleton className='h-8 w-36' />
            <Skeleton className='h-8 w-72 max-w-full' />
          </div>
          <div className={isTable ? 'border-t' : 'stolas-model-gallery'}>
            {Array.from({ length: isTable ? 10 : 6 }, (_, index) => (
              <div
                key={index}
                className={
                  isTable
                    ? 'grid gap-6 border-b py-7 md:grid-cols-[minmax(0,1fr)_minmax(12rem,0.5fr)]'
                    : 'overflow-hidden rounded-xl border'
                }
              >
                {isTable ? (
                  <>
                    <div className='flex min-w-0 gap-3'>
                      <Skeleton className='size-10 shrink-0' />
                      <div className='flex min-w-0 flex-1 flex-col gap-3'>
                        <Skeleton className='h-5 w-56 max-w-full' />
                        <Skeleton className='h-4 w-full max-w-lg' />
                        <Skeleton className='h-3 w-32' />
                      </div>
                    </div>
                    <div className='flex gap-6'>
                      <Skeleton className='h-11 w-24' />
                      <Skeleton className='h-11 w-24' />
                    </div>
                  </>
                ) : (
                  <>
                    <Skeleton className='stolas-model-artwork-frame rounded-none' />
                    <div className='flex flex-col gap-4 p-6'>
                      <Skeleton className='h-6 w-3/4' />
                      <Skeleton className='h-3 w-24' />
                      <Skeleton className='h-10 w-full' />
                      <Skeleton className='mt-4 h-8 w-1/2' />
                    </div>
                  </>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
