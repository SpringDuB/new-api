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
import {
  flexRender,
  type Cell,
  type Row,
  type Table,
} from '@tanstack/react-table'
import { Database } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { formatLogQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import { LOG_TYPE_ENUM } from '../constants'
import type { UsageLog } from '../data/schema'
import { TASK_MOBILE_SUMMARY_FIELDS } from '../lib/task-mobile-layout'
import type { LogCategory, TaskLog } from '../types'
import { CommonLogMobileCard } from './common-log-mobile-card'
import { useLogsViewScope, useUsageLogsContext } from './usage-logs-provider'

const logTypeRowTint: Record<number, string> = {
  [LOG_TYPE_ENUM.ERROR]:
    'bg-rose-50/40 dark:bg-rose-950/20 border-rose-200/50 dark:border-rose-900/30',
  [LOG_TYPE_ENUM.REFUND]:
    'bg-blue-50/30 dark:bg-blue-950/15 border-blue-200/50 dark:border-blue-900/30',
}

interface UsageLogsMobileListProps<TData> {
  table: Table<TData>
  isLoading?: boolean
  emptyTitle?: string
  emptyDescription?: string
  logCategory: LogCategory
}

function UsageLogsMobileSkeleton(props: { separate: boolean }) {
  const { t } = useTranslation()
  return (
    <div
      role='status'
      aria-label={t('Loading')}
      aria-busy='true'
      className={
        props.separate
          ? 'min-w-0 space-y-3'
          : 'border-border/50 bg-card overflow-hidden rounded-lg border'
      }
    >
      {[1, 2, 3].map((i) => (
        <div
          key={i}
          className={
            props.separate
              ? 'border-border/60 bg-card space-y-3 rounded-xl border p-3.5'
              : 'border-border/40 space-y-2.5 border-b p-3 last:border-b-0'
          }
        >
          <div className='flex items-center justify-between gap-3'>
            <Skeleton className='h-5 w-40 rounded-md' />
            <Skeleton className='h-5 w-16 rounded-md' />
          </div>
          <div className='grid grid-cols-2 gap-x-4 gap-y-2'>
            {[1, 2, 3, 4, 5, 6].map((j) => (
              <div key={j} className='min-w-0 space-y-1'>
                <Skeleton className='h-3 w-10 rounded' />
                <Skeleton className='h-4 w-full rounded' />
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function CompactCell<TData>({
  cell,
  fallback = '-',
  className,
  primaryOnly = false,
}: {
  cell?: Cell<TData, unknown>
  fallback?: string
  className?: string
  primaryOnly?: boolean
}) {
  return (
    <div
      className={cn(
        'min-w-0 overflow-hidden leading-tight [&_button]:max-w-full [&_span]:max-w-full',
        primaryOnly &&
          '[&_.flex-col]:min-w-0 [&_.flex-col>*:not(:first-child)]:hidden',
        className
      )}
    >
      {cell ? (
        flexRender(cell.column.columnDef.cell, cell.getContext())
      ) : (
        <span className='text-muted-foreground/50'>{fallback}</span>
      )}
    </div>
  )
}

function SummaryField<TData>({
  label,
  cell,
  className,
  valueClassName,
  primaryOnly = false,
}: {
  label?: string
  cell?: Cell<TData, unknown>
  className?: string
  valueClassName?: string
  primaryOnly?: boolean
}) {
  if (!cell) return null

  return (
    <div className={cn('stolas-activity-field min-w-0', className)}>
      {label != null && label !== '' && (
        <div className='stolas-activity-label'>{label}</div>
      )}
      <CompactCell
        cell={cell}
        primaryOnly={primaryOnly}
        className={valueClassName}
      />
    </div>
  )
}

function TaskLogsCard<TData>({
  cells,
  log,
}: {
  cells: Map<string, Cell<TData, unknown>>
  log: TaskLog
}) {
  const { t } = useTranslation()
  const { isAdminView } = useLogsViewScope()
  const { sensitiveVisible } = useUsageLogsContext()

  const taskIdCell = cells.get('task_id')
  const statusCell = cells.get('status')

  return (
    <div className='stolas-activity-task'>
      <div className='stolas-activity-task-time'>
        <SummaryField
          label={t('Submit Time')}
          cell={cells.get('submit_time')}
        />
      </div>
      <div className='stolas-activity-identity'>
        {cells.has('model') ? (
          <SummaryField label={t('Model')} cell={cells.get('model')} />
        ) : null}
        <CompactCell cell={taskIdCell} className='stolas-activity-task-id' />
      </div>
      <div className='stolas-activity-outcome'>
        <CompactCell cell={statusCell} />
        {isAdminView && (
          <div className='stolas-activity-cost'>
            <span className='stolas-activity-label'>{t('Cost')}</span>
            <span className='font-semibold tabular-nums'>
              {sensitiveVisible ? formatLogQuota(log.quota) : '••••'}
            </span>
          </div>
        )}
        <div className='flex flex-wrap items-start gap-x-5 gap-y-2'>
          <SummaryField
            label={t('Duration')}
            cell={cells.get('duration')}
            primaryOnly
          />
          <SummaryField label={t('Progress')} cell={cells.get('progress')} />
        </div>
      </div>
      <div className='stolas-activity-metadata'>
        {cells.has('request_mode') ? (
          <SummaryField
            label={t('Request type')}
            cell={cells.get('request_mode')}
          />
        ) : null}
        {TASK_MOBILE_SUMMARY_FIELDS.filter((field) =>
          ['user', 'plugin', 'channel_id'].includes(field.id)
        ).map((field) => (
          <SummaryField
            key={field.id}
            label={t(field.label)}
            cell={cells.get(field.id)}
            primaryOnly={field.primaryOnly}
          />
        ))}
      </div>
      <div className='stolas-activity-task-actions'>
        <CompactCell cell={cells.get('artifacts')} />
        <CompactCell cell={cells.get('fail_reason')} />
      </div>
    </div>
  )
}

export function UsageLogActivityRecord<TData>(props: {
  row: Row<TData>
  logCategory: LogCategory
}) {
  const { t } = useTranslation()
  const cells = new Map(
    props.row.getVisibleCells().map((cell) => [cell.column.id, cell])
  )

  return (
    <article
      aria-label={t('Activity record')}
      className='stolas-activity-record'
    >
      {props.logCategory === 'common' ? (
        <CommonLogMobileCard
          log={props.row.original as UsageLog}
          cells={cells}
        />
      ) : (
        <TaskLogsCard log={props.row.original as TaskLog} cells={cells} />
      )}
    </article>
  )
}

export function UsageLogsMobileList<TData>({
  table,
  isLoading = false,
  emptyTitle,
  emptyDescription,
  logCategory,
}: UsageLogsMobileListProps<TData>) {
  const { t } = useTranslation()

  const resolvedEmptyTitle = emptyTitle ?? t('No Logs Found')
  const resolvedEmptyDescription =
    emptyDescription ??
    t('No usage logs available. Logs will appear here once API calls are made.')

  if (isLoading) {
    return <UsageLogsMobileSkeleton separate={logCategory === 'common'} />
  }

  const rows = table.getRowModel().rows

  if (!rows || rows.length === 0) {
    return (
      <div className='rounded-lg border p-6'>
        <Empty className='border-none p-0'>
          <EmptyHeader>
            <EmptyMedia variant='icon'>
              <Database className='size-6' />
            </EmptyMedia>
            <EmptyTitle>{resolvedEmptyTitle}</EmptyTitle>
            <EmptyDescription>{resolvedEmptyDescription}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      </div>
    )
  }

  return (
    <div className='stolas-activity-list min-w-0'>
      {rows.map((row) => {
        const logType = (row.original as Record<string, unknown>).type as
          | number
          | undefined
        const tintClass = logType != null ? (logTypeRowTint[logType] ?? '') : ''

        return (
          <div
            key={row.id}
            className={cn('stolas-activity-record-wrapper min-w-0', tintClass)}
          >
            <UsageLogActivityRecord row={row} logCategory={logCategory} />
          </div>
        )
      })}
    </div>
  )
}
