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
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { afterEach, describe, expect, it } from 'vitest'

import { CanvasEntry } from '@/components/canvas-entry'
import zh from '@/i18n/locales/zh.json'

import { Hero } from '../components/sections/hero'

let client: QueryClient | undefined

afterEach(() => {
  cleanup()
  client?.clear()
})

async function renderHero(isAuthenticated: boolean) {
  client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  })
  client.setQueryData(['status'], {})
  const router = createRouter({
    routeTree: createRootRoute({
      component: () => <Hero isAuthenticated={isAuthenticated} />,
    }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}

describe('home canvas recommendation', () => {
  it.each([false, true])(
    'offers creators and developers working entry points when authenticated=%s',
    async (isAuthenticated) => {
      await renderHero(isAuthenticated)

      const action = await screen.findByRole('button', {
        name: 'Build with the API',
      })
      const heading = screen.getByRole('heading', {
        name: 'What would you like to make?',
      })
      expect(
        action.compareDocumentPosition(heading) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).not.toBe(0)
      expect(action).toHaveAttribute(
        'href',
        isAuthenticated ? '/keys' : '/sign-up?redirect=%2Fkeys'
      )
      const links = screen.getAllByRole('link', {
        name: 'Open Infinite Canvas',
      })
      expect(links).toHaveLength(2)
      for (const link of links) {
        expect(link).toHaveAttribute(
          'href',
          'https://canvas.stolas.cc/api/auth/stolas/start'
        )
      }
    }
  )

  it('keeps the canvas action keyboard accessible', async () => {
    const user = userEvent.setup()
    client = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    })
    client.setQueryData(['status'], {})
    render(
      <QueryClientProvider client={client}>
        <CanvasEntry />
      </QueryClientProvider>
    )

    await user.tab()
    expect(
      screen.getByRole('link', { name: 'Open Infinite Canvas' })
    ).toHaveFocus()
  })

  it('disables the creator entry when the service explicitly disables the bridge', () => {
    client = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    })
    client.setQueryData(['status'], { canvas_enabled: false })
    render(
      <QueryClientProvider client={client}>
        <CanvasEntry />
      </QueryClientProvider>
    )

    expect(
      screen.getByRole('button', { name: 'Service unavailable' })
    ).toBeDisabled()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })

  it('updates the recommendation and action when the interface switches to Chinese', async () => {
    const i18n = createInstance()
    await i18n.init({
      lng: 'en',
      fallbackLng: 'en',
      resources: { en: { translation: {} }, zh },
    })
    client = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    })
    client.setQueryData(['status'], {})
    render(
      <QueryClientProvider client={client}>
        <I18nextProvider i18n={i18n}>
          <CanvasEntry />
        </I18nextProvider>
      </QueryClientProvider>
    )
    expect(
      screen.getByRole('link', { name: 'Open Infinite Canvas' })
    ).toBeInTheDocument()

    await act(() => i18n.changeLanguage('zh'))

    expect(
      screen.getByRole('link', { name: zh.translation['Open Infinite Canvas'] })
    ).toHaveAttribute('href', 'https://canvas.stolas.cc/api/auth/stolas/start')
  })
})
