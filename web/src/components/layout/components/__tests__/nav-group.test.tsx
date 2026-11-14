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
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { SidebarProvider, useSidebar } from '@/components/ui/sidebar'
import { useSidebarData } from '@/hooks/use-sidebar-data'

import { NavGroup } from '../nav-group'

let client: QueryClient

afterEach(() => {
  cleanup()
  client?.clear()
  vi.unstubAllGlobals()
})

function Navigation() {
  const group = useSidebarData().navGroups.find((entry) => entry.id === 'chat')
  const { openMobile, setOpenMobile } = useSidebar()
  return (
    <>
      <button type='button' onClick={() => setOpenMobile(true)}>
        Open navigation
      </button>
      <output aria-label='Mobile navigation'>
        {openMobile ? 'open' : 'closed'}
      </output>
      {group && <NavGroup {...group} />}
    </>
  )
}

async function renderNavigation(collapsed = false) {
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], { chats: [] })
  const router = createRouter({
    routeTree: createRootRoute({
      component: () => (
        <SidebarProvider defaultOpen={!collapsed}>
          <Navigation />
        </SidebarProvider>
      ),
    }),
    history: createMemoryHistory({ initialEntries: ['/playground'] }),
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

describe('sidebar canvas link', () => {
  it.each([false, true])(
    'renders an accessible external link and preserves the active Playground when collapsed=%s',
    async (collapsed) => {
      await renderNavigation(collapsed)

      const canvas = await screen.findByRole('link', {
        name: 'Infinite Canvas',
      })
      expect(canvas).toHaveAttribute('href', 'https://canvas.stolas.cc')
      expect(canvas).toHaveAttribute('target', '_blank')
      expect(canvas).toHaveAttribute('rel', 'noopener noreferrer')
      expect(canvas).not.toHaveAttribute('data-active')
      expect(screen.getByRole('link', { name: 'Playground' })).toHaveAttribute(
        'data-active'
      )
      expect(screen.getByText('Online Use')).toBeInTheDocument()

      const user = userEvent.setup()
      await user.tab()
      await user.tab()
      await user.tab()
      expect(canvas).toHaveFocus()
    }
  )

  it('closes the open mobile navigation when the canvas link is activated', async () => {
    vi.stubGlobal('innerWidth', 375)
    const user = userEvent.setup()
    await renderNavigation()
    await user.click(
      await screen.findByRole('button', { name: 'Open navigation' })
    )
    expect(
      screen.getByRole('status', { name: 'Mobile navigation' })
    ).toHaveTextContent('open')

    await user.click(screen.getByRole('link', { name: 'Infinite Canvas' }))

    expect(
      screen.getByRole('status', { name: 'Mobile navigation' })
    ).toHaveTextContent('closed')
  })
})
