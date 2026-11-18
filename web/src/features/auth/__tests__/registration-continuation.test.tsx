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
import { act, cleanup, renderHook } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { useAuthRedirect } from '../hooks/use-auth-redirect'

const { navigate } = vi.hoisted(() => ({ navigate: vi.fn() }))
vi.mock('@tanstack/react-router', () => ({ useNavigate: () => navigate }))

afterEach(() => cleanup())

describe('registration continuation', () => {
  it('keeps the trusted canvas continuation when registration finishes', () => {
    const redirect =
      '/canvas/connect?state=browser-state&code_challenge=challenge'
    const { result } = renderHook(() => useAuthRedirect())

    act(() => result.current.redirectToLogin(redirect))

    expect(navigate).toHaveBeenCalledWith({
      to: '/sign-in',
      search: { redirect },
      replace: true,
    })
  })

  it('discards an external return address when registration finishes', () => {
    const { result } = renderHook(() => useAuthRedirect())

    act(() => result.current.redirectToLogin('https://untrusted.example/steal'))

    expect(navigate).toHaveBeenCalledWith({
      to: '/sign-in',
      search: { redirect: undefined },
      replace: true,
    })
  })
})
