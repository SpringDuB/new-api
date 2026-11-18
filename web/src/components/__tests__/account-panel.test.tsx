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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import {
  act,
  cleanup,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { ThemeProvider } from '@/context/theme-provider'
import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { ProfileDropdown } from '../profile-dropdown'

let client: QueryClient

afterEach(() => {
  cleanup()
  client?.clear()
  useAuthStore.getState().auth.reset()
  localStorage.clear()
  document.documentElement.classList.remove('light', 'dark')
})

async function renderAccount(
  options: { role?: number; hiddenModules?: boolean } = {}
) {
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: {} } })
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], {
    SidebarModulesAdmin: options.hiddenModules
      ? JSON.stringify({
          personal: { enabled: true, topup: false, security: false },
        })
      : '',
  })
  useAuthStore.getState().auth.setUser({
    id: 2,
    username: 'member',
    display_name: 'Canvas Member',
    role: options.role ?? ROLE.USER,
  })
  const router = createRouter({
    routeTree: createRootRoute({ component: () => <ProfileDropdown /> }),
    history: createMemoryHistory({ initialEntries: ['/profile'] }),
  })
  await router.load()
  await act(async () =>
    render(
      <QueryClientProvider client={client}>
        <ThemeProvider defaultTheme='light'>
          <RouterProvider router={router} />
        </ThemeProvider>
      </QueryClientProvider>
    )
  )
}

describe('Stolas account panel', () => {
  it('opens by keyboard and returns focus to the account trigger when Escape closes it', async () => {
    const user = userEvent.setup()
    await renderAccount()
    const trigger = screen.getByRole('button', { name: 'Account menu' })
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    trigger.focus()
    await user.keyboard('{Enter}')
    expect(
      await screen.findByRole('dialog', { name: 'Canvas Member' })
    ).toBeVisible()
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    await user.keyboard('{Escape}')
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    )
    expect(trigger).toHaveFocus()
  })

  it('hides disabled account destinations and system settings from a regular member', async () => {
    await renderAccount({ hiddenModules: true })
    await userEvent
      .setup()
      .click(screen.getByRole('button', { name: 'Account menu' }))
    const panel = await screen.findByRole('dialog', { name: 'Canvas Member' })
    expect(within(panel).getByRole('button', { name: 'Profile' })).toBeVisible()
    expect(
      within(panel).queryByRole('button', { name: 'Wallet' })
    ).not.toBeInTheDocument()
    expect(
      within(panel).queryByRole('button', { name: 'Security & Access' })
    ).not.toBeInTheDocument()
    expect(
      within(panel).queryByRole('button', { name: 'System Settings' })
    ).not.toBeInTheDocument()
  })

  it('lets a super administrator open appearance options inside the account panel and apply a theme', async () => {
    const user = userEvent.setup()
    await renderAccount({ role: ROLE.SUPER_ADMIN })
    await user.click(screen.getByRole('button', { name: 'Account menu' }))
    expect(
      await screen.findByRole('button', { name: 'System Settings' })
    ).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Toggle theme' }))
    await user.click(await screen.findByRole('menuitem', { name: 'Dark' }))
    expect(document.documentElement).toHaveClass('dark')
    expect(screen.getByRole('dialog', { name: 'Canvas Member' })).toBeVisible()
  })

  it('opens the existing sign out confirmation after closing the account panel', async () => {
    const user = userEvent.setup()
    await renderAccount()
    await user.click(screen.getByRole('button', { name: 'Account menu' }))
    await user.click(await screen.findByRole('button', { name: 'Sign out' }))
    expect(
      await screen.findByRole('alertdialog', { name: 'Sign out' })
    ).toBeVisible()
    expect(
      screen.queryByRole('dialog', { name: 'Canvas Member' })
    ).not.toBeInTheDocument()
    expect(useAuthStore.getState().auth.user?.id).toBe(2)
  })
})
