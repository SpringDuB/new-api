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
import { act, cleanup, render, screen } from '@testing-library/react'
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
  it('keeps destinations in one top navigation and marks the current API page in its menu', async () => {
    await renderWorkspace('/keys')
    expect(screen.getByRole('link', { name: 'Model library' })).toHaveAttribute(
      'href',
      '/pricing'
    )
    expect(
      screen.queryByRole('button', { name: 'Toggle Sidebar' })
    ).not.toBeInTheDocument()
    const menu = screen.getByRole('button', { name: /API Keys/ })
    menu.focus()
    await userEvent.setup().keyboard('{Enter}')
    expect(
      await screen.findByRole('menuitem', { name: 'API Keys' })
    ).toHaveAttribute('href', '/keys')
    expect(screen.getByRole('menuitem', { name: 'API Keys' })).toHaveAttribute(
      'aria-current',
      'page'
    )
  })
  it('keeps administrator destinations out of a regular user workspace', async () => {
    await renderWorkspace('/wallet')
    expect(
      screen.queryByRole('button', { name: /Admin/ })
    ).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Wallet/ })).toBeVisible()
  })
  it('honors module visibility within the API menu', async () => {
    await renderWorkspace('/keys', {
      userModules: { console: { log: false, audit: false } },
    })
    await userEvent
      .setup()
      .click(screen.getByRole('button', { name: /API Keys/ }))
    expect(
      await screen.findByRole('menuitem', { name: 'API Keys' })
    ).toBeVisible()
    expect(
      screen.queryByRole('menuitem', { name: 'Usage Logs' })
    ).not.toBeInTheDocument()
  })
  it('keeps system settings reachable from the top navigation', async () => {
    await renderWorkspace('/system-settings/billing/model-pricing', {
      role: ROLE.SUPER_ADMIN,
    })
    await userEvent
      .setup()
      .click(screen.getByRole('button', { name: /System Settings/ }))
    expect(
      await screen.findByRole('menuitem', { name: 'Model Pricing' })
    ).toHaveAttribute('href', '/system-settings/billing/model-pricing')
  })
  it('preserves the drawing alias as the current task page and the canvas entry', async () => {
    await renderWorkspace('/usage-logs/drawing')
    expect(screen.getByRole('button', { name: /Task Logs/ })).toBeVisible()
    expect(
      screen.getByRole('link', { name: /Infinite Canvas/ })
    ).toHaveAttribute('target', '_blank')
    expect(
      screen.getByRole('navigation', { name: 'Workspace pages' })
    ).not.toHaveClass('overflow-x-auto')
  })
  it('uses a compact current-page menu on mobile and retains every destination', async () => {
    vi.stubGlobal('innerWidth', 390)
    await renderWorkspace('/keys')
    const menu = screen.getByRole('button', { name: 'API Keys' })
    expect(menu).toHaveAttribute('aria-expanded', 'false')
    await userEvent.setup().click(menu)
    expect(
      await screen.findByRole('menuitem', { name: 'API Keys' })
    ).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('menuitem', { name: 'Wallet' })).toBeVisible()
    expect(
      screen.getByRole('menuitem', { name: 'Infinite Canvas' })
    ).toHaveAttribute('target', '_blank')
  })
  it('uses the compact menu on narrower desktops with all destinations reachable', async () => {
    const originalMatchMedia = window.matchMedia
    vi.spyOn(window, 'matchMedia').mockImplementation((query) => ({
      ...originalMatchMedia(query),
      matches: query === '(max-width: 1279px)',
    }))
    await renderWorkspace('/keys')
    await userEvent
      .setup()
      .click(screen.getByRole('button', { name: 'API Keys' }))
    expect(
      await screen.findByRole('menuitem', { name: 'Model library' })
    ).toBeVisible()
    expect(screen.getByRole('menuitem', { name: 'Wallet' })).toBeVisible()
  })

  it('hides the canvas entry when integration is disabled', async () => {
    await renderWorkspace('/keys', { canvasEnabled: false })
    expect(
      screen.queryByRole('link', { name: /Infinite Canvas/ })
    ).not.toBeInTheDocument()
  })
})
