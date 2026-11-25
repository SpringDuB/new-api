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
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { act, cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { useAuthStore } from '@/stores/auth-store'

import { PublicHeader } from '../components/public-header'

vi.mock('@/hooks/use-system-config', () => ({
  useSystemConfig: () => ({
    systemName: 'Stolas',
    logo: '/stolas/mark-black.png',
    loading: false,
    logoLoaded: true,
  }),
}))
vi.mock('@/hooks/use-top-nav-links', () => ({
  useTopNavLinks: () => [
    {
      title: 'Infinite Canvas',
      href: 'https://canvas.stolas.cc/api/auth/stolas/start',
      external: true,
    },
  ],
}))
vi.mock('@/hooks/use-notifications', () => ({ useNotifications: () => ({}) }))
vi.mock('@/components/notification-popover', () => ({
  NotificationPopover: () => null,
}))
vi.mock('@/components/profile-dropdown', () => ({
  ProfileDropdown: () => null,
}))
vi.mock('@/components/language-switcher', () => ({
  LanguageSwitcher: () => null,
}))
vi.mock('@/components/theme-switch', () => ({ ThemeSwitch: () => null }))
vi.mock('@/features/system-update/system-update-action', () => ({
  SystemUpdateAction: () => null,
}))

afterEach(() => {
  cleanup()
  useAuthStore.getState().auth.reset()
})

async function renderHeader() {
  const router = createRouter({
    routeTree: createRootRoute({ component: PublicHeader }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  await act(async () => render(<RouterProvider router={router} />))
}

describe('Stolas public navigation', () => {
  it('keeps the closed mobile menu out of navigation until a keyboard user opens it', async () => {
    const user = userEvent.setup()
    await renderHeader()
    const toggle = screen.getByRole('button', {
      name: 'Toggle navigation menu',
    })
    const menu = document.querySelector('#stolas-mobile-navigation')

    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(menu).toHaveAttribute('inert')
    expect(menu).toHaveAttribute('aria-hidden', 'true')
    toggle.focus()
    await user.keyboard('{Enter}')

    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(menu).not.toHaveAttribute('inert')
    expect(menu).toHaveAttribute('aria-hidden', 'false')
    expect(document.body.style.overflow).toBe('hidden')
    await user.keyboard('{Escape}')
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(toggle).toHaveFocus()
    expect(document.body.style.overflow).toBe('')
  })

  it('keeps creator navigation on the fixed SSO entry and displays the official Stolas logo', async () => {
    await renderHeader()

    for (const link of screen.getAllByRole('link', {
      name: 'Infinite Canvas',
    })) {
      expect(link).toHaveAttribute(
        'href',
        'https://canvas.stolas.cc/api/auth/stolas/start'
      )
    }
    expect(screen.getByRole('img', { name: 'Stolas' })).toHaveAttribute(
      'src',
      '/stolas/lockup-dark.png'
    )
  })
})
