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
import { createFileRoute, redirect } from '@tanstack/react-router'

import { LoadingState } from '@/components/loading-state'
import { CanvasConnectError } from '@/features/canvas-connect'
import {
  canvasSearchSchema,
  connectCanvas,
} from '@/features/canvas-connect/connection'
import { api } from '@/lib/api'
import { resolveAuthentication } from '@/lib/auth-session'
import { AuthOperationError } from '@/lib/secure-verification'
import { requireServerSuccess } from '@/lib/server-error-message'
import { statusQueryOptions } from '@/lib/status-query'

export const Route = createFileRoute('/canvas/connect')({
  validateSearch: canvasSearchSchema,
  pendingComponent: () => <LoadingState />,
  errorComponent: () => <CanvasConnectError />,
  component: () => <LoadingState />,
  beforeLoad: async ({ search, context, preload }) => {
    if (preload) return
    const destination = await connectCanvas(search, {
      loadStatus: () => context.queryClient.fetchQuery(statusQueryOptions),
      restoreSession: async () => {
        const result = await resolveAuthentication()
        if (
          result.kind === 'transient_error' ||
          result.kind === 'out_of_sync'
        ) {
          throw new AuthOperationError(
            'Unable to restore your session. Please try again.'
          )
        }
        return result.kind === 'authenticated'
      },
      authorize: async (request) => {
        const response = await api.post('/api/canvas/authorize', request, {
          singleUseAuthorization: true,
          skipErrorHandler: true,
        })
        return requireServerSuccess(response.data).data as {
          redirect_url: string
        }
      },
    })
    if (destination.kind === 'sign-in') {
      throw redirect({
        to: '/sign-in',
        search: { redirect: destination.redirect },
        replace: true,
      })
    }
    throw redirect({
      href: destination.href,
      reloadDocument: true,
      replace: true,
    })
  },
})
