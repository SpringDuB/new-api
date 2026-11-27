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
    walletQuota?: number
  } = {}
) {
  vi.spyOn(api, 'get').mockImplementation(async (url) => ({
    data: {
      success: true,
      data:
        url === '/api/user/self' ? { id: 1, quota: options.walletQuota } : [],
    },
  }))
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], {
    system_name: 'New API',
    canvas_enabled: options.canvasEnabled ?? true,
    canvas_url: 'https://canvas.example.test',
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
  it.each([390, 1440])(
    'places the canvas shortcut between the model library and wallet balance at %ipx',
    async (width) => {
      vi.stubGlobal('innerWidth', width)
      await renderWorkspace('/keys', { walletQuota: 1250000 })
      const header = screen.getByRole('banner')
      const balance = await within(header).findByRole('group', {
        name: 'Current Balance',
      })
      const library = within(header).getByRole('link', {
        name: 'Model library',
      })
      expect(library).toHaveAttribute('href', '/pricing')
      const canvas = within(header).getByRole('link', {
        name: 'Infinite Canvas',
      })
      expect(canvas).toHaveAttribute(
        'href',
        'https://canvas.example.test/api/auth/stolas/start'
      )
      expect(canvas).toHaveAttribute('target', '_blank')
      expect(canvas).toHaveAttribute('rel', 'noopener noreferrer')
      expect(
        library.compareDocumentPosition(canvas) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy()
      expect(
        canvas.compareDocumentPosition(balance) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy()
      expect(await within(balance).findByText('$2.5')).toBeVisible()
      const explore = within(header).getByRole('button', {
        name: 'Explore Stolas',
      })
      expect(
        balance.compareDocumentPosition(explore) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy()
    }
  )

  it('previews on hover and navigates directly when the model library label is clicked', async () => {
    const user = userEvent.setup()
    const router = await renderWorkspace('/dashboard/overview')
    const library = within(screen.getByRole('banner')).getByRole('link', {
      name: 'Model library',
    })
    await user.hover(library)
    const menu = await screen.findByRole('dialog', { name: 'Model library' })
    await user.hover(within(menu).getByRole('link', { name: 'All models' }))
    expect(menu).toBeVisible()
    await user.click(library)
    await waitFor(() => expect(router.state.location.href).toBe('/pricing'))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('marks the current page and keeps links reachable with the rail collapsed', async () => {
    await renderWorkspace('/keys')
    const nav = within(
      screen.getByRole('navigation', { name: 'Workspace pages' })
    )
    expect(nav.getByRole('link', { name: 'API Keys' })).toHaveAttribute(
      'aria-current',
      'page'
    )
    expect(nav.getByRole('link', { name: 'Wallet' })).toHaveAttribute(
      'href',
      '/wallet'
    )
    expect(
      nav.queryByRole('link', { name: 'Channels' })
    ).not.toBeInTheDocument()
    const toggle = screen.getByRole('button', { name: 'Close menu' })
    toggle.focus()
    await userEvent.setup().keyboard('{Enter}')
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(nav.getByRole('link', { name: 'API Keys' })).toHaveAttribute(
      'href',
      '/keys'
    )
    await userEvent.setup().keyboard('{Enter}')
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
  })
  it('groups variants by series and links to the matching catalog filter', async () => {
    await renderWorkspace('/dashboard/overview')
    client.setQueryData(['pricing'], {
      vendors: [],
      data: [
        { id: 1, model_name: 'seedance-2-5', output_modalities: ['video'] },
        { id: 2, model_name: 'seedream-5-0', output_modalities: ['image'] },
        {
          id: 3,
          model_name: '59-seedance-2-5-flash',
          output_modalities: ['video'],
        },
      ],
    })
    const menu = screen.getByRole('button', { name: 'Model categories' })
    menu.focus()
    await userEvent.setup().keyboard('{Enter}')
    const family = await screen.findByRole('link', {
      name: /Seedance 2.5.*2 models/,
    })
    const href = family.getAttribute('href')
    if (!href) throw new Error('The model series link has no destination')
    const familyUrl = new URL(href, 'https://stolas.test')
    expect(familyUrl.pathname).toBe('/pricing')
    expect(familyUrl.searchParams.get('category')).toBe('video')
    expect(familyUrl.searchParams.get('family')).toBe('series:Seedance 2.5')
    expect(screen.getAllByRole('link', { name: /Seedance 2.5/ })).toHaveLength(
      1
    )
    const imagesCategory = screen.getByRole('link', { name: /^Images/ })
    expect(imagesCategory).toHaveAttribute('href', '/pricing?category=image')
    const videoCover = family.querySelector('.stolas-library-cover')
    expect(videoCover).toHaveAttribute('src')
    await act(async () => imagesCategory.focus())
    const imagePreview = await screen.findByRole('region', { name: 'Images' })
    const imageCover = within(imagePreview)
      .getByRole('link', { name: /Seedream/ })
      .querySelector('.stolas-library-cover')
    expect(imageCover).toHaveAttribute('src')
    expect(imageCover?.getAttribute('src')).not.toBe(
      videoCover?.getAttribute('src')
    )
    expect(screen.getByRole('link', { name: 'All models' })).toHaveAttribute(
      'href',
      '/pricing'
    )
    await userEvent.setup().keyboard('{Escape}')
    expect(menu).toHaveAttribute('aria-expanded', 'false')
  })
  it('shows permitted administrator links and respects user module visibility', async () => {
    await renderWorkspace('/keys', {
      role: ROLE.SUPER_ADMIN,
      userModules: { console: { log: false, audit: false } },
    })
    const nav = within(
      screen.getByRole('navigation', { name: 'Workspace pages' })
    )
    expect(nav.getByRole('link', { name: 'Channels' })).toHaveAttribute(
      'href',
      '/channels'
    )
    expect(nav.getByRole('link', { name: 'System Settings' })).toBeVisible()
    expect(
      nav.queryByRole('link', { name: 'Usage Logs' })
    ).not.toBeInTheDocument()
  })
  it('shows contextual settings and provides a way back to the console', async () => {
    await renderWorkspace('/system-settings/billing/model-pricing', {
      role: ROLE.SUPER_ADMIN,
    })
    expect(screen.getByRole('link', { name: 'Model Pricing' })).toHaveAttribute(
      'aria-current',
      'page'
    )
    expect(
      screen.getByRole('link', { name: 'Back to Dashboard' })
    ).toHaveAttribute('href')
  })
  it('preserves the drawing alias and external canvas link', async () => {
    await renderWorkspace('/usage-logs/drawing')
    expect(screen.getByRole('link', { name: 'Task Logs' })).toHaveAttribute(
      'aria-current',
      'page'
    )
    expect(
      within(
        screen.getByRole('navigation', { name: 'Workspace pages' })
      ).getByRole('link', { name: 'Infinite Canvas' })
    ).toHaveAttribute('target', '_blank')
  })
  it('opens a mobile drawer and dismisses it on Escape or navigation', async () => {
    vi.stubGlobal('innerWidth', 390)
    await renderWorkspace('/keys')
    const user = userEvent.setup()
    const toggle = screen.getByRole('button', { name: 'Open menu' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    await user.click(toggle)
    const drawer = await screen.findByRole('dialog')
    expect(
      within(drawer).getByRole('link', { name: 'API Keys' })
    ).toHaveAttribute('aria-current', 'page')
    expect(
      within(drawer).queryByRole('link', { name: 'Model library' })
    ).not.toBeInTheDocument()
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    await user.click(toggle)
    await user.click(screen.getByRole('link', { name: 'Wallet' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it.each([
    { canvasEnabled: false },
    { adminModules: { chat: { canvas: false } } },
    { userModules: { chat: { canvas: false } } },
  ])(
    'hides disabled canvas entries including in the header (%j)',
    async (options) => {
      await renderWorkspace('/keys', options)
      expect(
        screen.queryByRole('link', { name: /Infinite Canvas/ })
      ).not.toBeInTheDocument()
    }
  )
})
