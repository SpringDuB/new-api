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

import { getTaskClientRequest, getTaskResponse } from '../api'
import type { TaskClientRequestResponse, TaskResponseResponse } from '../types'

function TaskSnapshotDetails(props: {
  taskId: string
  open: boolean
  kind: 'request' | 'response'
}) {
  const { t } = useTranslation()
  const isResponse = props.kind === 'response'
  const request = useQuery<TaskClientRequestResponse | TaskResponseResponse>({
    queryKey: [
      'usage-logs',
      isResponse ? 'task-response' : 'task-client-request',
      props.taskId,
    ],
    queryFn: () =>
      isResponse
        ? getTaskResponse(props.taskId)
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
  if (!request.data.recorded || !request.data.snapshot) {
    return <p className='text-muted-foreground text-sm'>{t('Not recorded')}</p>
  }

  const snapshot = request.data.snapshot
  const code = JSON.stringify(snapshot, null, 2)
  let title = t('Original client request')
  if (isResponse) {
    title =
      'source' in request.data && request.data.source === 'upstream_result'
        ? t('Stored upstream result')
        : t('Client response')
  }
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
      <p className='text-muted-foreground text-xs'>
        {isResponse
          ? t(
              'Response snapshots are limited to 64 KiB and retained with the task record'
            )
          : t(
              'Request snapshots are limited to 64 KiB and retained with the task record'
            )}
      </p>
      {snapshot.body_kind === 'events' ? (
        <p className='text-muted-foreground text-xs'>
          {t('Streaming logs retain final image and error events only')}
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
      <CodeBlock
        code={code}
        language='json'
        title={title}
        showToolbar
        collapsedLines={12}
        maxExpandedLines={28}
      >
        <CopyButton value={code} />
      </CodeBlock>
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
