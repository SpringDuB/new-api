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
import { act, cleanup, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from '@/lib/api'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { AuthenticatedLayout } from '../components/authenticated-layout'

vi.mock('@/components/config-drawer', () => ({ ConfigDrawer: () => null }))
vi.mock('@/components/profile-dropdown', () => ({
  ProfileDropdown: () => null,
}))
vi.mock('@/components/language-switcher', () => ({
  LanguageSwitcher: () => null,
}))
vi.mock('@/components/notification-popover', () => ({
  NotificationPopover: () => null,
}))
vi.mock('@/hooks/use-notifications', () => ({ useNotifications: () => ({}) }))
vi.mock('@/features/system-update/system-update-action', () => ({
  SystemUpdateAction: () => null,
}))
vi.mock('@/components/command-menu', () => ({ CommandMenu: () => null }))

let client: QueryClient

afterEach(() => {
  cleanup()
  client?.clear()
  useAuthStore.getState().auth.reset()
  vi.unstubAllGlobals()
})

async function renderWorkspace(
  path: string,
  options: {
    role?: number
    adminModules?: object
    userModules?: object
    canvasEnabled?: boolean
  } = {}
) {
  vi.spyOn(api, 'get').mockResolvedValue({ data: { success: true, data: [] } })
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], {
    system_name: 'New API',
    canvas_enabled: options.canvasEnabled ?? true,
    canvas_url: 'https://canvas.stolas.cc',
    chats: [{ Studio: 'https://chat.example.test' }],
    SidebarModulesAdmin: options.adminModules
      ? JSON.stringify(options.adminModules)
      : '',
  })
  useAuthStore.getState().auth.setUser({
    id: 1,
    username: 'studio-user',
    role: options.role ?? ROLE.USER,
    sidebar_modules: options.userModules
      ? JSON.stringify(options.userModules)
      : '',
  })
  const router = createRouter({
    routeTree: createRootRoute({
      component: () => (
        <AuthenticatedLayout>
          <main>
            <h1>Page content</h1>
          </main>
        </AuthenticatedLayout>
      ),
    }),
    history: createMemoryHistory({ initialEntries: [path] }),
  })
  await router.load()
  await act(async () =>
    render(
      <QueryClientProvider client={client}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    )
  )
  return router
}

describe('Stolas workspace navigation', () => {
  it('marks the developer area and current API key page without showing other areas in the context navigation', async () => {
    await renderWorkspace('/keys')
    const areas = screen.getByRole('navigation', { name: 'Workspace areas' })
    expect(
      within(areas).getByRole('link', { name: 'API access' })
    ).toHaveAttribute('aria-current', 'location')
    const pages = screen.getByRole('navigation', { name: 'Workspace pages' })
    expect(
      within(pages).getByRole('link', { name: 'API Keys' })
    ).toHaveAttribute('aria-current', 'page')
    expect(
      within(pages).queryByRole('link', { name: 'Wallet' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: 'Toggle Sidebar' })
    ).not.toBeInTheDocument()
  })

  it('keeps administrator destinations out of a regular user workspace', async () => {
    await renderWorkspace('/wallet')
    expect(
      screen.queryByRole('link', { name: 'Administration' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('link', { name: 'Channels' })
    ).not.toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Account & billing' })
    ).toHaveAttribute('aria-current', 'page')
  })

  it('honors administrator and user module visibility in the reorganized developer navigation', async () => {
    await renderWorkspace('/keys', {
      userModules: { console: { log: false, audit: false } },
    })
    const pages = screen.getByRole('navigation', { name: 'Workspace pages' })
    expect(within(pages).getByRole('link', { name: 'API Keys' })).toBeVisible()
    expect(
      within(pages).queryByRole('link', { name: 'Usage Logs' })
    ).not.toBeInTheDocument()
  })

  it('keeps system settings reachable through contextual menus while retaining the admin area', async () => {
    await renderWorkspace('/system-settings/billing/model-pricing', {
      role: ROLE.SUPER_ADMIN,
    })
    expect(
      screen.getByRole('link', { name: 'Administration' })
    ).toHaveAttribute('aria-current', 'location')
    const trigger = screen.getByRole('button', { name: /Billing & Payment/ })
    trigger.focus()
    await userEvent.setup().keyboard('{Enter}')
    expect(
      await screen.findByRole('menuitem', { name: /Model Pricing/ })
    ).toHaveAttribute('href', '/system-settings/billing/model-pricing')
  })

  it('preserves the drawing log alias as the active task page on a narrow viewport', async () => {
    vi.stubGlobal('innerWidth', 390)
    await renderWorkspace('/usage-logs/drawing')
    expect(screen.getByRole('link', { name: 'Task Logs' })).toHaveAttribute(
      'aria-current',
      'page'
    )
    expect(
      screen.getByRole('navigation', { name: 'Workspace areas' })
    ).toHaveClass('overflow-x-auto')
    expect(
      screen.getByRole('navigation', { name: 'Workspace pages' })
    ).toHaveClass('overflow-x-auto')
  })
})
