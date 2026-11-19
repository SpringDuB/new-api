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
import { getRouteApi } from '@tanstack/react-router'
import { flexRender, type Row } from '@tanstack/react-table'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  DISABLED_ROW_DESKTOP,
  DISABLED_ROW_MOBILE,
  DataTablePage,
  useDebouncedColumnFilter,
  useDataTable,
  type DataTableViewMode,
} from '@/components/data-table'
import { StatusBadge } from '@/components/status-badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { useTableUrlState } from '@/hooks/use-table-url-state'
import { createServerError } from '@/lib/server-error-message'

import { getApiKeys, searchApiKeys } from '../api'
import {
  API_KEY_STATUS,
  API_KEY_STATUS_OPTIONS,
  API_KEY_STATUSES,
  ERROR_MESSAGES,
} from '../constants'
import type { ApiKey } from '../types'
import { ApiKeyQuotaCell } from './api-key-quota-cell'
import { ApiKeyActivityCell } from './api-key-timestamp-cell'
import {
  ApiKeyCell,
  ModelLimitsCell,
  IpRestrictionsCell,
} from './api-keys-cells'
import { useApiKeysColumns } from './api-keys-columns'
import { useApiKeys } from './api-keys-provider'
import { DataTableBulkActions } from './data-table-bulk-actions'
import { DataTableRowActions } from './data-table-row-actions'

const route = getRouteApi('/_authenticated/keys/')
const API_KEYS_COLUMN_VISIBILITY_STORAGE_KEY = 'api-keys:column-visibility'
function isDisabledApiKeyRow(apiKey: ApiKey) {
  return apiKey.status !== API_KEY_STATUS.ENABLED
}

function ApiKeyDirectoryEntry(props: { row: Row<ApiKey>; now: number }) {
  const { t } = useTranslation()
  const apiKey = props.row.original
  const statusConfig = API_KEY_STATUSES[apiKey.status]
  const cells = props.row.getAllCells()
  const groupCell = cells.find((cell) => cell.column.id === 'group')
  const expiryCell = cells.find((cell) => cell.column.id === 'expired_time')

  return (
    <article aria-label={apiKey.name} className='stolas-key-entry'>
      <header className='stolas-key-identity'>
        <div className='flex min-w-0 items-start gap-3'>
          <Checkbox
            checked={props.row.getIsSelected()}
            onCheckedChange={(value) => props.row.toggleSelected(!!value)}
            aria-label={t('Select {{name}}', { name: apiKey.name })}
            className='mt-1'
          />
          <div className='min-w-0'>
            <div className='mb-2 flex flex-wrap items-center gap-3'>
              <h3 className='text-base font-semibold break-all'>
                {apiKey.name}
              </h3>
              {statusConfig && (
                <StatusBadge
                  label={t(statusConfig.label)}
                  variant={statusConfig.variant}
                  copyable={false}
                />
              )}
            </div>
            <ApiKeyCell apiKey={apiKey} />
          </div>
        </div>
        <DataTableRowActions row={props.row} />
      </header>
      <div className='stolas-key-details'>
        <section aria-label={t('Quota Settings')} className='stolas-key-budget'>
          <p className='stolas-eyebrow mb-3'>{t('Quota Settings')}</p>
          <ApiKeyQuotaCell apiKey={apiKey} now={props.now} variant='card' />
        </section>
        <section aria-label={t('Access restrictions')} className='space-y-3'>
          <p className='stolas-eyebrow'>{t('Access restrictions')}</p>
          <div>
            {groupCell &&
              flexRender(
                groupCell.column.columnDef.cell,
                groupCell.getContext()
              )}
          </div>
          <div className='flex flex-wrap gap-x-5 gap-y-2'>
            <ModelLimitsCell apiKey={apiKey} detailsTrigger='click' />
            <IpRestrictionsCell apiKey={apiKey} detailsTrigger='click' />
          </div>
        </section>
        <section aria-label={t('Activity')} className='space-y-3'>
          <ApiKeyActivityCell
            apiKey={apiKey}
            now={props.now}
            layout='columns'
          />
          <div className='flex items-center gap-3 text-xs'>
            <span className='text-muted-foreground'>{t('Expires')}</span>
            {expiryCell &&
              flexRender(
                expiryCell.column.columnDef.cell,
                expiryCell.getContext()
              )}
          </div>
        </section>
      </div>
    </article>
  )
}

export function ApiKeysTable() {
  const { t } = useTranslation()
  const { refreshTrigger } = useApiKeys()
  const [now, setNow] = useState(() => Date.now())
  const [viewMode, setViewMode] = useState<DataTableViewMode>('card')
  const columns = useApiKeysColumns(now)

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      setNow(Date.now())
    }, 30_000)

    return () => window.clearInterval(intervalId)
  }, [])

  const {
    globalFilter,
    onGlobalFilterChange,
    columnFilters,
    onColumnFiltersChange,
    pagination,
    onPaginationChange,
    ensurePageInRange,
  } = useTableUrlState({
    search: route.useSearch(),
    navigate: route.useNavigate(),
    pagination: { defaultPage: 1, defaultPageSize: 20 },
    globalFilter: { enabled: true, key: 'filter' },
    columnFilters: [
      { columnId: 'status', searchKey: 'status', type: 'array' },
      { columnId: '_tokenSearch', searchKey: 'token', type: 'string' },
    ],
  })

  const {
    value: tokenFilter,
    inputValue: tokenFilterInput,
    setInputValue: setTokenFilterInput,
  } = useDebouncedColumnFilter({
    columnFilters,
    columnId: '_tokenSearch',
    onColumnFiltersChange,
  })
  const shouldSearch = Boolean(globalFilter?.trim() || tokenFilter.trim())

  // Fetch data with React Query
  // eslint-disable-next-line @tanstack/query/exhaustive-deps
  const { data, isLoading, isFetching } = useQuery({
    queryKey: [
      'keys',
      pagination.pageIndex + 1,
      pagination.pageSize,
      globalFilter,
      tokenFilter,
      refreshTrigger,
    ],
    queryFn: async () => {
      const result = shouldSearch
        ? await searchApiKeys({
            keyword: globalFilter,
            token: tokenFilter,
            p: pagination.pageIndex + 1,
            size: pagination.pageSize,
          })
        : await getApiKeys({
            p: pagination.pageIndex + 1,
            size: pagination.pageSize,
          })

      if (!result.success) {
        throw createServerError(
          result,
          t(
            shouldSearch
              ? ERROR_MESSAGES.SEARCH_FAILED
              : ERROR_MESSAGES.LOAD_FAILED
          )
        )
      }

      return {
        items: result.data?.items || [],
        total: result.data?.total || 0,
      }
    },
    placeholderData: (previousData) => previousData,
  })

  const apiKeys = data?.items || []

  const { table } = useDataTable({
    data: apiKeys,
    columns,
    enableRowSelection: true,
    columnFilters,
    columnVisibilityStorageKey: API_KEYS_COLUMN_VISIBILITY_STORAGE_KEY,
    globalFilter,
    pagination,
    globalFilterFn: () => true,
    onPaginationChange,
    onGlobalFilterChange,
    onColumnFiltersChange,
    manualPagination: true,
    totalCount: data?.total || 0,
    ensurePageInRange,
  })

  const columnVisibility = table.getState().columnVisibility
  useEffect(() => {
    // Restore the dates hidden by the previous default when adopting the combined time column.
    if (
      columnVisibility.activity_time === undefined &&
      columnVisibility.created_time === false &&
      columnVisibility.accessed_time === false &&
      columnVisibility.expired_time === false
    ) {
      table.setColumnVisibility((previous) => ({
        ...previous,
        activity_time: true,
        expired_time: true,
      }))
    }
  }, [columnVisibility, table])

  return (
    <DataTablePage
      table={table}
      columns={columns}
      isLoading={isLoading}
      isFetching={isFetching}
      emptyTitle={t('No API Keys Found')}
      emptyDescription={t(
        'No API keys available. Create your first API key to get started.'
      )}
      skeletonKeyPrefix='api-keys-skeleton'
      applyHeaderSize
      enableCardView
      viewMode={viewMode}
      onViewModeChange={setViewMode}
      fixedHeight={false}
      cardGridClassName='stolas-key-directory'
      renderCard={(row) => <ApiKeyDirectoryEntry row={row} now={now} />}
      showMobileBulkActions
      toolbarProps={{
        hideViewOptions: viewMode === 'card',
        leftActions: viewMode === 'card' && (
          <div className='text-muted-foreground flex items-center gap-2 text-xs'>
            <Checkbox
              aria-label={t('Select all')}
              checked={table.getIsAllPageRowsSelected()}
              indeterminate={table.getIsSomePageRowsSelected()}
              onCheckedChange={(value) =>
                table.toggleAllPageRowsSelected(!!value)
              }
            />
            {t('Select all')}
          </div>
        ),
        searchPlaceholder: t('Filter by name...'),
        searchDebounceMs: 500,
        additionalSearch: (
          <Input
            placeholder={t('Filter by API key...')}
            aria-label={t('Filter by API key...')}
            value={tokenFilterInput}
            onChange={(e) => setTokenFilterInput(e.target.value)}
            className='w-full sm:w-50 lg:w-60'
          />
        ),
        filters: [
          {
            columnId: 'status',
            title: t('Status'),
            options: API_KEY_STATUS_OPTIONS,
            singleSelect: true,
          },
        ],
      }}
      getRowClassName={(row, context) => {
        if (!isDisabledApiKeyRow(row.original)) return undefined
        return context.isMobile ? DISABLED_ROW_MOBILE : DISABLED_ROW_DESKTOP
      }}
      bulkActions={<DataTableBulkActions table={table} />}
    />
  )
}
