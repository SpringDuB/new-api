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

import zh from '@/i18n/locales/zh.json'

import { InfiniteCanvasCard } from '../components/infinite-canvas-card'
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
    'shows the canvas recommendation after the primary action when authenticated=%s',
    async (isAuthenticated) => {
      await renderHero(isAuthenticated)

      const action = await screen.findByRole('button', {
        name: isAuthenticated ? 'Go to Dashboard' : 'Get Started',
      })
      const heading = screen.getByRole('heading', {
        name: 'Want to generate videos online?',
      })
      expect(
        action.compareDocumentPosition(heading) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).not.toBe(0)
      const link = screen.getByRole('link', { name: 'Open Infinite Canvas' })
      expect(link).toHaveAttribute('href', 'https://canvas.stolas.cc')
      expect(link).toHaveAttribute('target', '_blank')
      expect(link).toHaveAttribute('rel', 'noopener noreferrer')
    }
  )

  it('keeps the canvas action keyboard accessible', async () => {
    const user = userEvent.setup()
    render(<InfiniteCanvasCard />)

    await user.tab()
    expect(
      screen.getByRole('link', { name: 'Open Infinite Canvas' })
    ).toHaveFocus()
  })

  it('updates the recommendation and action when the interface switches to Chinese', async () => {
    const i18n = createInstance()
    await i18n.init({
      lng: 'en',
      fallbackLng: 'en',
      resources: { en: { translation: {} }, zh },
    })
    render(
      <I18nextProvider i18n={i18n}>
        <InfiniteCanvasCard />
      </I18nextProvider>
    )
    expect(
      screen.getByRole('link', { name: 'Open Infinite Canvas' })
    ).toBeInTheDocument()

    await act(() => i18n.changeLanguage('zh'))

    expect(
      screen.getByRole('heading', {
        name: zh.translation['Want to generate videos online?'],
      })
    ).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: zh.translation['Open Infinite Canvas'] })
    ).toHaveAttribute('href', 'https://canvas.stolas.cc')
  })
})
