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
import { describe, expect, it, vi } from 'vitest'

import { connectCanvas, getCanvasStartURL } from '../connection'

const canvasURL = 'https://canvas.stolas.cc'
const request = {
  redirect_uri: `${canvasURL}/api/auth/stolas/callback`,
  state: 'browser-bound-state-1234567890123456',
  code_challenge: 'a'.repeat(43),
  code_challenge_method: 'S256',
}

function runtime(authenticated = true) {
  return {
    loadStatus: vi
      .fn()
      .mockResolvedValue({ canvas_enabled: true, canvas_url: canvasURL }),
    restoreSession: vi.fn().mockResolvedValue(authenticated),
    authorize: vi.fn().mockResolvedValue({
      redirect_url: `${request.redirect_uri}?code=single-use-code&state=${request.state}`,
    }),
  }
}

describe('canvas session connection', () => {
  it('restores the current session before authorizing and returns only the trusted callback', async () => {
    const dependencies = runtime()
    const result = await connectCanvas(request, dependencies)

    expect(
      dependencies.restoreSession.mock.invocationCallOrder[0]
    ).toBeLessThan(dependencies.authorize.mock.invocationCallOrder[0])
    expect(dependencies.authorize).toHaveBeenCalledWith(request)
    expect(result).toEqual({
      kind: 'canvas',
      href: `${request.redirect_uri}?code=single-use-code&state=${request.state}`,
    })
  })

  it('sends an anonymous visitor to sign in with the complete local continuation', async () => {
    const dependencies = runtime(false)
    const result = await connectCanvas(request, dependencies)

    expect(result.kind).toBe('sign-in')
    if (result.kind !== 'sign-in') throw new Error('Expected sign-in')
    const continuation = new URL(result.redirect, 'https://stolas.cc')
    expect(continuation.pathname).toBe('/canvas/connect')
    expect(Object.fromEntries(continuation.searchParams)).toEqual(request)
    expect(dependencies.authorize).not.toHaveBeenCalled()
  })

  it('starts a fresh browser-bound flow after the registration continuation', async () => {
    const dependencies = runtime()

    expect(await connectCanvas({}, dependencies)).toEqual({
      kind: 'canvas',
      href: `${canvasURL}/api/auth/stolas/start`,
    })
    expect(dependencies.authorize).not.toHaveBeenCalled()
  })

  it.each([
    { redirect_uri: 'https://evil.example/api/auth/stolas/callback' },
    { redirect_uri: `${request.redirect_uri}?next=https://evil.example` },
    {
      redirect_uri:
        'https://canvas.stolas.cc.evil.example/api/auth/stolas/callback',
    },
    { code_challenge: 'invalid' },
    { code_challenge_method: 'plain' },
    { state: '' },
  ])(
    'rejects invalid authorization parameters before restoring the session: %j',
    async (invalid) => {
      const dependencies = runtime()

      await expect(
        connectCanvas({ ...request, ...invalid }, dependencies)
      ).rejects.toThrow()
      expect(dependencies.restoreSession).not.toHaveBeenCalled()
      expect(dependencies.authorize).not.toHaveBeenCalled()
    }
  )

  it('rejects a partial authorization request instead of silently starting a new flow', async () => {
    const dependencies = runtime()
    await expect(
      connectCanvas({ state: request.state }, dependencies)
    ).rejects.toThrow()
  })

  it.each([
    'https://evil.example/api/auth/stolas/callback?code=x',
    `${request.redirect_uri}?code=x&state=another-browser`,
    `${request.redirect_uri}?code=x&state=${request.state}&access_token=secret`,
    `${request.redirect_uri}?state=${request.state}`,
    `${request.redirect_uri}?code=x&state=${request.state}#untrusted`,
  ])(
    'rejects an untrusted authorization response: %s',
    async (redirect_url) => {
      const dependencies = runtime()
      dependencies.authorize.mockResolvedValue({ redirect_url })
      await expect(connectCanvas(request, dependencies)).rejects.toThrow()
    }
  )

  it('fails closed when the configured canvas integration is disabled', async () => {
    const dependencies = runtime()
    dependencies.loadStatus.mockResolvedValue({
      canvas_enabled: false,
      canvas_url: canvasURL,
    })
    await expect(connectCanvas(request, dependencies)).rejects.toThrow()
    expect(dependencies.authorize).not.toHaveBeenCalled()
  })

  it('keeps the creator entry fixed when public configuration is missing or malformed', () => {
    expect(getCanvasStartURL(null)).toBe(`${canvasURL}/api/auth/stolas/start`)
    expect(getCanvasStartURL({ canvas_url: 'javascript:alert(1)' })).toBe(
      `${canvasURL}/api/auth/stolas/start`
    )
    expect(
      getCanvasStartURL({
        canvas_enabled: true,
        canvas_url: 'https://studio.example',
      })
    ).toBe('https://studio.example/api/auth/stolas/start')
  })

  it('surfaces a temporary session restoration failure without issuing authorization', async () => {
    const dependencies = runtime()
    dependencies.restoreSession.mockRejectedValue(
      new Error('Temporarily unavailable')
    )
    await expect(connectCanvas(request, dependencies)).rejects.toThrow(
      'Temporarily unavailable'
    )
    expect(dependencies.authorize).not.toHaveBeenCalled()
  })
})
