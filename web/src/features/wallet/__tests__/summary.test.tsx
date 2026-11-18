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
import { render, screen, within } from '@testing-library/react'
import { expect, it } from 'vitest'

import { WalletStatsCard } from '../components/wallet-stats-card'

it('keeps an unknown wallet balance in a busy state instead of showing a zero balance', () => {
  render(<WalletStatsCard user={null} loading />)
  expect(
    screen.getByRole('region', { name: 'Balance overview' })
  ).toHaveAttribute('aria-busy', 'true')
  expect(screen.queryByText('$0')).not.toBeInTheDocument()
})

it('shows the shared balance separately from usage and request totals', () => {
  render(
    <WalletStatsCard
      user={{
        id: 1,
        username: 'studio',
        quota: 1000000,
        used_quota: 250000,
        request_count: 1234,
        aff_quota: 0,
        aff_history_quota: 0,
        aff_count: 0,
        group: 'default',
      }}
    />
  )
  const summary = screen.getByRole('region', { name: 'Balance overview' })
  expect(summary).toHaveAttribute('aria-busy', 'false')
  expect(within(summary).getByRole('heading', { name: '$2' })).toBeVisible()
  expect(within(summary).getByText('$0.5')).toBeVisible()
  expect(within(summary).getByText('1,234')).toBeVisible()
})
