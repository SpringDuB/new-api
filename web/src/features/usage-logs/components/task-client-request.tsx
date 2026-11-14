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
import { useTranslation } from 'react-i18next'

import { CodeBlock } from '@/components/ai-elements/code-block'
import { CopyButton } from '@/components/copy-button'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Badge } from '@/components/ui/badge'
import { formatTimestampToDate } from '@/lib/format'

import { getTaskClientRequest, getTaskResponse } from '../api'
import type {
  TaskClientRequestResponse,
  TaskResponseResponse,
  TaskClientRequestSnapshot,
  TaskResponseSnapshot,
} from '../types'

function TaskSnapshotDetails(props: {
  taskId: string
  open: boolean
  kind: 'request' | 'response' | 'upstream'
}) {
  const { t } = useTranslation()
  const isResponse = props.kind !== 'request'
  const request = useQuery<TaskClientRequestResponse | TaskResponseResponse>({
    queryKey: [
      'usage-logs',
      isResponse ? 'task-response' : 'task-client-request',
      props.taskId,
      props.kind,
    ],
    queryFn: () =>
      isResponse
        ? getTaskResponse(
            props.taskId,
            props.kind === 'upstream' ? 'upstream_result' : undefined
          )
        : getTaskClientRequest(props.taskId),
    enabled: props.open,
    retry: false,
    gcTime: 0,
    meta: { errorToast: false },
  })

  if (!props.open) return null
  if (request.isPending) return <LoadingState className='min-h-20' size='sm' />
  if (request.isError) {
    return (
      <ErrorState
        className='min-h-24'
        title={
          isResponse
            ? t('Failed to load response')
            : t('Failed to load client request')
        }
        onRetry={() => void request.refetch()}
      />
    )
  }
  const data = request.data
  let title = t('Original client request')
  const current = 'source' in data && data.source === 'current_platform_result'
  if (isResponse) {
    title = t('Client response')
    if (props.kind === 'upstream') title = t('Stored upstream result')
    else if (current) title = t('Current platform query result')
  }
  let result = (
    <p className='text-muted-foreground text-sm'>{t('Not recorded')}</p>
  )
  if (data.recorded && data.snapshot) {
    result = (
      <TaskSnapshotContent
        snapshot={data.snapshot}
        isResponse={isResponse}
        title={title}
        current={current}
      />
    )
  }
  if ('unavailable' in data && data.unavailable) {
    result = (
      <ErrorState
        className='min-h-24'
        title={t('Platform response is unavailable')}
        onRetry={() => void request.refetch()}
      />
    )
  }
  return (
    <div className='min-w-0 space-y-3'>
      {current && data.recorded ? (
        <p className='text-muted-foreground text-xs'>
          {t(
            'Generated from current task state; signed URLs may differ from earlier responses'
          )}
          {'generated_at' in data && data.generated_at
            ? ` · ${formatTimestampToDate(data.generated_at, 'seconds')}`
            : null}
          {'query_path' in data && data.query_path ? (
            <span className='mt-1 block font-mono break-all'>
              GET {data.query_path}
            </span>
          ) : null}
        </p>
      ) : null}
      {result}
      {'submission_snapshot' in data && data.submission_snapshot ? (
        <TaskSnapshotContent
          snapshot={data.submission_snapshot}
          isResponse
          title={t('Submission response')}
        />
      ) : null}
    </div>
  )
}

function TaskSnapshotContent(props: {
  snapshot: TaskClientRequestSnapshot | TaskResponseSnapshot
  isResponse: boolean
  title: string
  current?: boolean
}) {
  const { t } = useTranslation()
  const snapshot = props.snapshot
  const isResponse = props.isResponse
  const code =
    JSON.stringify(isResponse ? snapshot.body : snapshot, null, 2) ?? ''
  let omissionReason = t('Content unavailable')
  if (snapshot.body_omitted === 'size_limit') {
    omissionReason = t('size limit exceeded')
  } else if (snapshot.body_omitted === 'invalid_or_too_deep') {
    omissionReason = t('Invalid or deeply nested content')
  }

  return (
    <div className='min-w-0 space-y-2'>
      {!isResponse ? (
        <p className='text-muted-foreground text-xs'>
          {t('Captured before request decoding and parameter conversion')}
        </p>
      ) : null}
      {!props.current ? (
        <p className='text-muted-foreground text-xs'>
          {isResponse
            ? t(
                'Response snapshots are limited to 64 KiB and retained with the task record'
              )
            : t(
                'Request snapshots are limited to 64 KiB and retained with the task record'
              )}
        </p>
      ) : null}
      {'captured_at' in snapshot && snapshot.captured_at ? (
        <p className='text-muted-foreground text-xs'>
          {t('Captured at')}:{' '}
          {formatTimestampToDate(snapshot.captured_at, 'seconds')}
        </p>
      ) : null}
      {'status_code' in snapshot && snapshot.status_code ? (
        <Badge variant='outline'>HTTP {snapshot.status_code}</Badge>
      ) : null}
      {snapshot.body_kind === 'events' ? (
        <p className='text-muted-foreground text-xs'>
          {t('Streaming logs retain final result and error events only')}
        </p>
      ) : null}
      <div className='flex flex-wrap gap-1.5'>
        {snapshot.redacted ? (
          <Badge variant='secondary'>{t('Sensitive values redacted')}</Badge>
        ) : null}
        {'files_omitted' in snapshot && snapshot.files_omitted ? (
          <Badge variant='secondary'>{t('File contents omitted')}</Badge>
        ) : null}
        {'binary_omitted' in snapshot && snapshot.binary_omitted ? (
          <Badge variant='secondary'>
            {t('Image binaries omitted from response logs')}
          </Badge>
        ) : null}
      </div>
      {snapshot.body_omitted ? (
        <p role='status' className='text-muted-foreground text-xs'>
          {isResponse
            ? t('Response body omitted: {{reason}}', { reason: omissionReason })
            : t('Request body omitted: {{reason}}', { reason: omissionReason })}
        </p>
      ) : null}
      {code ? (
        <CodeBlock
          code={code}
          language='json'
          title={props.title}
          showToolbar
          collapsedLines={12}
          maxExpandedLines={28}
        >
          <CopyButton value={code} />
        </CodeBlock>
      ) : (
        <p className='text-muted-foreground text-xs'>{props.title}</p>
      )}
    </div>
  )
}

export function TaskClientRequestDetails(props: {
  taskId: string
  open: boolean
}) {
  return <TaskSnapshotDetails {...props} kind='request' />
}

export function TaskResponseDetails(props: { taskId: string; open: boolean }) {
  return <TaskSnapshotDetails {...props} kind='response' />
}

export function TaskUpstreamResponseDetails(props: {
  taskId: string
  open: boolean
}) {
  return <TaskSnapshotDetails {...props} kind='upstream' />
}
