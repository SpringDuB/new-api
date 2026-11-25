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
import { act, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { SystemStatus } from '@/features/auth/types'
import { useIsSidebarModuleVisible } from '@/hooks/use-sidebar-config'
import { useStatus } from '@/hooks/use-status'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { Profile } from '..'
import { useProfile } from '../hooks'
import type { UserProfile } from '../types'

vi.mock('@/hooks/use-status', () => ({ useStatus: vi.fn() }))
vi.mock('@/hooks/use-sidebar-config', () => ({
  useIsSidebarModuleVisible: vi.fn(),
}))
vi.mock('../hooks', () => ({ useProfile: vi.fn() }))

const profile: UserProfile = {
  id: 21,
  username: 'mira',
  display_name: 'Mira Chen',
  email: 'mira@example.com',
  role: 1,
  group: 'studio',
  quota: 1000000,
  used_quota: 500000,
  request_count: 1250,
  status: 1,
  aff_count: 0,
  aff_quota: 0,
  aff_history_quota: 0,
  created_time: 0,
  setting: JSON.stringify({ notify_type: 'email' }),
}

let client: QueryClient | undefined

beforeEach(() => {
  vi.mocked(useProfile).mockReturnValue({
    profile,
    loading: false,
    updating: false,
    fetchProfile: vi.fn(),
    refreshProfile: vi.fn(),
    updateProfile: vi.fn(),
    updateSettings: vi.fn(),
  })
  vi.mocked(useStatus).mockReturnValue({
    status: null,
    loading: false,
    error: null,
  })
  vi.mocked(useIsSidebarModuleVisible).mockReturnValue(true)
  useAuthStore.getState().auth.setUser(profile)
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: profile },
  })
})

afterEach(() => {
  client?.clear()
  delete window.turnstile
  useAuthStore.getState().auth.setUser(null)
})

async function renderProfile() {
  client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const router = createRouter({
    routeTree: createRootRoute({ component: Profile }),
    history: createMemoryHistory({ initialEntries: ['/profile'] }),
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  await screen.findByRole('heading', { name: 'Personal Settings' })
}

describe('profile preference layout', () => {
  it('shows account identity and labeled facts beside the preference workspace', async () => {
    const user = userEvent.setup()
    await renderProfile()

    const identity = screen.getByRole('complementary', {
      name: 'Account details',
    })
    expect(
      within(identity).getByRole('img', { name: 'Stolas' })
    ).toHaveAttribute('src', '/stolas/mark-black.png')
    expect(
      within(identity).getByRole('heading', { name: 'Mira Chen' })
    ).toBeVisible()
    expect(within(identity).getByText('mira@example.com')).toBeVisible()
    expect(
      within(identity).getByText('User ID').nextElementSibling
    ).toHaveTextContent('21')
    expect(
      within(identity).getByText('Current Balance').nextElementSibling
    ).toHaveTextContent('$2')
    expect(
      within(identity).getByRole('link', { name: 'Open wallet' })
    ).toHaveAttribute('href', '/wallet')
    expect(screen.getAllByRole('tabpanel')).toHaveLength(1)
    await user.click(
      within(identity).getByRole('button', {
        name: 'Copy to clipboard: User ID',
      })
    )
    expect(await navigator.clipboard.readText()).toBe('21')
  })

  it('switching tabs preserves notification drafts and exposes only the selected panel', async () => {
    const user = userEvent.setup()
    await renderProfile()
    await user.type(
      screen.getByRole('textbox', { name: 'Notification Email' }),
      'draft@example.com'
    )
    await user.click(screen.getByRole('tab', { name: 'Language Preferences' }))

    expect(
      screen.getByRole('tab', { name: 'Language Preferences' })
    ).toHaveAttribute('aria-selected', 'true')
    expect(screen.getAllByRole('tabpanel')).toHaveLength(1)
    expect(
      screen.getByRole('combobox', { name: 'Interface Language' })
    ).toBeVisible()
    expect(
      screen.queryByRole('textbox', { name: 'Notification Email' })
    ).not.toBeInTheDocument()

    await user.click(screen.getByRole('tab', { name: 'Notifications' }))
    expect(
      screen.getByRole('textbox', { name: 'Notification Email' })
    ).toHaveValue('draft@example.com')
  })

  it('arrow keys move between preference tabs with matching selection', async () => {
    const user = userEvent.setup()
    await renderProfile()
    screen.getByRole('tab', { name: 'Notifications' }).focus()

    await user.keyboard('{ArrowRight}')

    const language = screen.getByRole('tab', { name: 'Language Preferences' })
    expect(language).toHaveFocus()
    expect(language).toHaveAttribute('aria-selected', 'true')
  })

  it('restricted sidebar preferences and disabled check-in have no tabs or content', async () => {
    useAuthStore.getState().auth.setUser({
      ...profile,
      permissions: { sidebar_settings: false },
    })
    await renderProfile()

    expect(
      screen.queryByRole('tab', { name: 'Navigation preferences' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('tab', { name: 'Daily Check-in' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByText('Customize sidebar display content')
    ).not.toBeInTheDocument()
  })

  it('hidden wallet modules do not expose a wallet shortcut', async () => {
    vi.mocked(useIsSidebarModuleVisible).mockReturnValue(false)
    await renderProfile()

    expect(
      screen.queryByRole('link', { name: 'Open wallet' })
    ).not.toBeInTheDocument()
    expect(screen.getByText('Current Balance')).toBeVisible()
  })

  it('revoking navigation access returns the selected panel to notifications', async () => {
    const user = userEvent.setup()
    await renderProfile()
    await user.click(
      screen.getByRole('tab', { name: 'Navigation preferences' })
    )

    act(() =>
      useAuthStore
        .getState()
        .auth.setUser({ ...profile, permissions: { sidebar_settings: false } })
    )

    expect(
      screen.queryByRole('tab', { name: 'Navigation preferences' })
    ).not.toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Notifications' })).toHaveAttribute(
      'aria-selected',
      'true'
    )
    expect(
      screen.getByRole('textbox', { name: 'Notification Email' })
    ).toBeVisible()
  })

  it('navigation preferences show allowed modules and disable children with their section', async () => {
    const user = userEvent.setup()
    vi.mocked(useStatus).mockReturnValue({
      status: {
        SidebarModulesAdmin: JSON.stringify({
          chat: { enabled: false },
          console: { enabled: true, token: true, detail: false },
          personal: { enabled: false },
        }),
      } as SystemStatus,
      loading: false,
      error: null,
    })
    await renderProfile()
    await user.click(
      screen.getByRole('tab', { name: 'Navigation preferences' })
    )

    const tokens = screen.getByRole('switch', { name: 'Token Management' })
    expect(tokens).not.toHaveAttribute('aria-disabled', 'true')
    expect(
      screen.queryByRole('switch', { name: 'Dashboard' })
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('switch', { name: 'Online Use' })
    ).not.toBeInTheDocument()

    await user.click(screen.getByRole('switch', { name: 'Console Area' }))

    expect(tokens).toHaveAttribute('aria-disabled', 'true')
    await user.click(tokens)
    expect(tokens).toBeChecked()
  })

  it('enabled check-in keeps its history control and completes a requested Turnstile challenge', async () => {
    const user = userEvent.setup()
    let verifyChallenge: ((token: string) => void) | undefined
    window.turnstile = {
      render: vi.fn((_element, options) => {
        verifyChallenge = options.callback as (token: string) => void
      }),
    }
    vi.mocked(useStatus).mockReturnValue({
      status: {
        checkin_enabled: true,
        turnstile_check: true,
        turnstile_site_key: 'test-site-key',
      } as SystemStatus,
      loading: false,
      error: null,
    })
    vi.mocked(api.get).mockImplementation(async (url) => ({
      data: {
        success: true,
        data: url.startsWith('/api/user/checkin')
          ? {
              enabled: true,
              stats: {
                checked_in_today: false,
                total_checkins: 0,
                total_quota: 0,
                checkin_count: 0,
                records: [],
              },
            }
          : profile,
      },
    }))
    const post = vi
      .spyOn(api, 'post')
      .mockResolvedValueOnce({
        data: { success: false, message: 'Turnstile required' },
      })
      .mockResolvedValueOnce({
        data: { success: true, data: { quota_awarded: 1000 } },
      })
    await renderProfile()
    await user.click(screen.getByRole('tab', { name: 'Daily Check-in' }))
    const history = await screen.findByRole('button', {
      name: 'Check-in history',
    })
    expect(history).toHaveAttribute('aria-expanded', 'true')
    await user.click(history)
    expect(history).toHaveAttribute('aria-expanded', 'false')
    expect(
      screen.queryByRole('button', { name: 'Previous month' })
    ).not.toBeInTheDocument()
    await user.click(history)
    expect(screen.getByRole('button', { name: 'Previous month' })).toBeVisible()

    await user.click(screen.getByRole('button', { name: 'Check in now' }))

    expect(
      await screen.findByRole('dialog', { name: 'Security Check' })
    ).toBeVisible()
    await waitFor(() => expect(verifyChallenge).toBeTypeOf('function'))
    await act(async () => verifyChallenge?.('verified/token'))
    await waitFor(() =>
      expect(
        screen.queryByRole('dialog', { name: 'Security Check' })
      ).not.toBeInTheDocument()
    )
    expect(post).toHaveBeenNthCalledWith(1, '/api/user/checkin')
    expect(post).toHaveBeenNthCalledWith(
      2,
      '/api/user/checkin?turnstile=verified%2Ftoken'
    )
  })
})
