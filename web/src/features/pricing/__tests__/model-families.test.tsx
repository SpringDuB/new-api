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
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { ModelCategoryFilter } from '../components/model-category-filter'
import { PricingSidebar } from '../components/pricing-sidebar'
import { SearchBar } from '../components/search-bar'
import { useFilters } from '../hooks/use-filters'
import {
  getModelFamilies,
  getModelFamily,
  getModelFamilyKey,
  inferModelFamily,
} from '../lib/model-families'
import { applyModelSquareDisplay } from '../lib/model-square-display'
import type { PricingModel } from '../types'

const models: PricingModel[] = [
  'first-seedance-2-5',
  'image-2',
  'old-seedance-2-0',
  'second-seedance2.5',
].map((model_name, id) => ({
  id,
  model_name,
  quota_type: 1,
  model_ratio: 1,
  completion_ratio: 1,
  enable_groups: ['default'],
  square_category: id === 1 ? 'image' : 'video',
}))

function FamilyCatalog() {
  const filters = useFilters(models)
  return (
    <>
      <SearchBar
        value={filters.searchInput}
        onChange={filters.setSearchInput}
        onClear={filters.clearSearch}
      />
      <ModelCategoryFilter
        value={filters.categoryFilter}
        counts={filters.categoryCounts}
        onChange={filters.setCategoryFilter}
      />
      <PricingSidebar
        familyFilter={filters.familyFilter}
        onFamilyChange={filters.setFamilyFilter}
        familyModels={filters.familyModels}
        quotaTypeFilter={filters.quotaTypeFilter}
        onQuotaTypeChange={filters.setQuotaTypeFilter}
        endpointTypeFilter={filters.endpointTypeFilter}
        onEndpointTypeChange={filters.setEndpointTypeFilter}
        vendorFilter={filters.vendorFilter}
        onVendorChange={filters.setVendorFilter}
        groupFilter={filters.groupFilter}
        onGroupChange={filters.setGroupFilter}
        tagFilter={filters.tagFilter}
        onTagChange={filters.setTagFilter}
        models={filters.sidebarModels}
        vendors={[]}
        groups={[]}
        tags={filters.availableTags}
        hasActiveFilters={filters.hasActiveFilters}
        onClearFilters={filters.clearFilters}
      />
      <ul aria-label='Visible models'>
        {filters.filteredModels.map((model) => (
          <li key={model.model_name}>{model.model_name}</li>
        ))}
      </ul>
    </>
  )
}

function renderCatalog(path = '/pricing/?category=video') {
  const root = createRootRoute()
  const route = createRoute({
    getParentRoute: () => root,
    path: '/pricing/',
    component: FamilyCatalog,
  })
  const router = createRouter({
    routeTree: root.addChildren([route]),
    history: createMemoryHistory({ initialEntries: [path] }),
  })
  render(<RouterProvider router={router} />)
}

describe('model series', () => {
  it.each([
    ['ac-seedance-2-5-full', 'Seedance 2.5'],
    ['Seedance2.5-fast', 'Seedance 2.5'],
    ['prefix-seedance_2_0-720p', 'Seedance 2.0'],
    ['seedance-2-50', 'Seedance 2.50'],
    ['seedance-2-9-301010', 'Seedance 2.9'],
    ['ac-seedream-5-0-pro', 'Seedream 5.0'],
    ['ad-minimax-h3-replace', 'MiniMax'],
    ['h3video-2k', 'MiniMax'],
    ['jl-wan-3-0-1055-720p', 'Wan'],
    ['ap-muse-video-720-10s', 'Muse Video'],
    ['muse-zr-700', 'Muse'],
    ['tinysnow-banana-3-1', 'Nano Banana'],
    ['nw-image-2-5', 'Image 2'],
    ['image2', 'Image 2'],
    ['sl-midjourney-niji', 'Midjourney'],
    ['mf-gemini-omni-1-2-flash', 'Gemini Omni'],
    ['unknown-model', undefined],
    ['notseedance-2-5', undefined],
  ])(
    'groups %s as %s without using provider prefixes or variant suffixes',
    (name, family) => {
      expect(inferModelFamily(name)).toBe(family)
    }
  )

  it('lets a custom series override automatic grouping and keeps custom sentinel-like names filterable', () => {
    const [custom] = applyModelSquareDisplay([models[0]], {
      order: [],
      models: { [models[0].model_name]: { family: 'all' } },
    })
    expect(getModelFamily(custom)).toBe('all')
    expect(getModelFamilyKey(custom)).toBe('series:all')
    const [automatic] = applyModelSquareDisplay([models[0]], {
      order: [],
      models: {},
    })
    expect(getModelFamily(automatic)).toBe('Seedance 2.5')
    expect(getModelFamilies([])).toEqual([])
    expect(
      getModelFamilies([...models, { ...models[0], model_name: 'unknown' }])
    ).toEqual([
      { value: 'series:Image 2', label: 'Image 2', count: 1 },
      { value: 'series:Seedance 2.0', label: 'Seedance 2.0', count: 1 },
      { value: 'series:Seedance 2.5', label: 'Seedance 2.5', count: 2 },
      { value: 'unassigned', label: undefined, count: 1 },
    ])
  })

  it('limits series to the category and resets its filters while retaining category and recommendation order', async () => {
    renderCatalog()
    const user = userEvent.setup()
    await screen.findByRole('button', { name: /^Videos\s*3$/ })
    expect(
      screen.queryByRole('button', { name: /^Image 2\s*1$/ })
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /^Seedance 2.5\s*2$/ }))
    expect(
      within(screen.getByRole('list', { name: 'Visible models' }))
        .getAllByRole('listitem')
        .map((el) => el.textContent)
    ).toEqual(['first-seedance-2-5', 'second-seedance2.5'])
    expect(
      screen.getByRole('button', { name: /^Seedance 2.0\s*1$/ })
    ).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Reset' }))
    expect(
      screen.getByRole('button', { name: /^Videos\s*3$/ })
    ).toHaveAttribute('aria-pressed', 'true')
    expect(
      within(screen.getByRole('list', { name: 'Visible models' })).getAllByRole(
        'listitem'
      )
    ).toHaveLength(3)
    await user.click(screen.getByRole('button', { name: /^Seedance 2.5\s*2$/ }))
    await user.click(screen.getByRole('button', { name: /^Images\s*1$/ }))
    expect(
      within(screen.getByRole('list', { name: 'Visible models' })).getAllByRole(
        'listitem'
      )
    ).toHaveLength(1)
    expect(
      screen.getByRole('button', { name: /^Image 2\s*1$/ })
    ).toHaveAttribute('aria-pressed', 'false')
  })

  it('loads a series query and updates its sibling counts with search', async () => {
    renderCatalog('/pricing/?category=video&family=series%3ASeedance%202.5')
    const user = userEvent.setup()
    const family = await screen.findByRole('button', {
      name: /^Seedance 2.5\s*2$/,
    })
    expect(family).toHaveAttribute('aria-pressed', 'true')
    await user.type(
      screen.getByRole('textbox', { name: 'Search models' }),
      'first'
    )
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: /^Seedance 2.5\s*1$/ })
      ).toBeVisible()
    )
    expect(
      screen.queryByRole('button', { name: /^Seedance 2.0/ })
    ).not.toBeInTheDocument()
    expect(
      within(screen.getByRole('list', { name: 'Visible models' })).getByRole(
        'listitem'
      )
    ).toHaveTextContent('first-seedance-2-5')
  })
})
