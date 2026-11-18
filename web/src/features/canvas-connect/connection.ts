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
import { z } from 'zod'

export const canvasSearchSchema = z.object({
  redirect_uri: z.string().optional(),
  state: z.string().optional(),
  code_challenge: z.string().optional(),
  code_challenge_method: z.string().optional(),
})

export type CanvasSearch = z.infer<typeof canvasSearchSchema>
export type CanvasStatus = {
  canvas_enabled?: unknown
  canvas_url?: unknown
}

const authorizationSchema = z.object({
  redirect_uri: z.string(),
  state: z.string().regex(/^[A-Za-z0-9._~-]{16,512}$/),
  code_challenge: z.string().regex(/^[A-Za-z0-9_-]{43}$/),
  code_challenge_method: z.literal('S256'),
})

type AuthorizationRequest = z.infer<typeof authorizationSchema>
type CanvasConnectionRuntime = {
  loadStatus: () => Promise<CanvasStatus | null>
  restoreSession: () => Promise<boolean>
  authorize: (
    request: AuthorizationRequest
  ) => Promise<{ redirect_url: string }>
}

type CanvasDestination =
  | { kind: 'canvas'; href: string }
  | { kind: 'sign-in'; redirect: string }

function configuredCanvasOrigin(status: CanvasStatus | null): string | null {
  if (
    status?.canvas_enabled !== true ||
    typeof status.canvas_url !== 'string'
  ) {
    return null
  }
  try {
    const url = new URL(status.canvas_url)
    if (
      url.protocol !== 'https:' ||
      url.username ||
      url.password ||
      url.pathname !== '/' ||
      url.search ||
      url.hash
    ) {
      return null
    }
    return url.origin
  } catch {
    return null
  }
}

export function getCanvasStartURL(status: CanvasStatus | null): string {
  const origin = configuredCanvasOrigin(status) ?? 'https://canvas.stolas.cc'
  return `${origin}/api/auth/stolas/start`
}

export async function connectCanvas(
  search: CanvasSearch,
  runtime: CanvasConnectionRuntime
): Promise<CanvasDestination> {
  const origin = configuredCanvasOrigin(await runtime.loadStatus())
  if (!origin) throw new Error('Canvas connection is unavailable')

  const hasAuthorization = Object.values(search).some(
    (value) => value !== undefined
  )
  let request: AuthorizationRequest | undefined
  let continuation = '/canvas/connect'
  if (hasAuthorization) {
    const parsed = authorizationSchema.safeParse(search)
    if (
      !parsed.success ||
      parsed.data.redirect_uri !== `${origin}/api/auth/stolas/callback`
    ) {
      throw new Error('Invalid canvas connection request')
    }
    request = parsed.data
    continuation += `?${new URLSearchParams(request).toString()}`
  }

  if (!(await runtime.restoreSession())) {
    return { kind: 'sign-in', redirect: continuation }
  }
  if (!request) {
    return { kind: 'canvas', href: `${origin}/api/auth/stolas/start` }
  }

  const response = await runtime.authorize(request)
  const destination = new URL(response.redirect_url)
  const params = destination.searchParams
  if (
    destination.origin !== origin ||
    destination.username ||
    destination.password ||
    destination.pathname !== '/api/auth/stolas/callback' ||
    destination.hash ||
    params.get('state') !== request.state ||
    !params.get('code') ||
    params.getAll('state').length !== 1 ||
    params.getAll('code').length !== 1 ||
    [...params.keys()].some((key) => key !== 'code' && key !== 'state')
  ) {
    throw new Error('Invalid canvas connection response')
  }

  return { kind: 'canvas', href: destination.href }
}
