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

import { Button } from '@/components/ui/button'

import { ModelCategoryFilter } from '../components/model-category-filter'
import { PricingSidebar } from '../components/pricing-sidebar'
import { SearchBar } from '../components/search-bar'
import { useFilters } from '../hooks/use-filters'
import { sortModels } from '../lib/filters'
import type { ModelCategory } from '../lib/model-categories'
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

function FamilyCatalog(props: { models: PricingModel[] }) {
  const filters = useFilters(props.models)
  return (
    <>
      <Button onClick={() => filters.setSortBy('recommended')}>
        Recommended order
      </Button>
      <Button onClick={() => filters.setSortBy('name')}>Name order</Button>
      <Button onClick={() => filters.setSortBy('family')}>Series order</Button>
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
        groupFilter={filters.groupFilter}
        onGroupChange={filters.setGroupFilter}
        models={filters.sidebarModels}
        groups={[]}
      />
      <Button onClick={filters.clearFilters}>Reset</Button>
      <ul aria-label='Visible models'>
        {filters.filteredModels.map((model) => (
          <li key={model.model_name}>{model.model_name}</li>
        ))}
      </ul>
    </>
  )
}

function renderCatalog(
  path = '/pricing/?category=video',
  catalog: PricingModel[] = models
) {
  const root = createRootRoute()
  const route = createRoute({
    getParentRoute: () => root,
    path: '/pricing/',
    component: () => <FamilyCatalog models={catalog} />,
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
    ['jl-wan3.0-fast', 'Wan'],
    ['ap-muse-video-720-10s', 'Muse Video'],
    ['muse-zr-700', 'Muse'],
    ['tinysnow-banana-3-1', 'Nano Banana'],
    ['nw-image-2-5', 'Image 2'],
    ['image2', 'Image 2'],
    ['sl-midjourney-niji', 'Midjourney'],
    ['mf-gemini-omni-1-2-flash', 'Gemini Omni'],
    ['anthropic/claude-sonnet-4', 'Claude'],
    ['proxy-gpt-4o-mini', 'GPT'],
    ['proxy-deepseek-chat', 'DeepSeek'],
    ['Qwen/Qwen3-32B', 'Qwen'],
    ['proxy-glm-4.5', 'GLM'],
    ['proxy-grok-3-mini', 'Grok'],
    ['unknown-model', undefined],
    ['notseedance-2-5', undefined],
    ['notgpt-4o', undefined],
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
      { value: 'series:Seedance 2.5', label: 'Seedance 2.5', count: 2 },
      { value: 'series:Seedance 2.0', label: 'Seedance 2.0', count: 1 },
      { value: 'series:Image 2', label: 'Image 2', count: 1 },
      { value: 'unassigned', label: undefined, count: 1 },
    ])
  })

  it.each<{
    category: ModelCategory
    names: string[]
    expected: string[]
    families: string[]
  }>([
    {
      category: 'video',
      names: [
        'a-minimax-h3',
        'b-seedance-2-0',
        'c-wan-3-0',
        'd-seedance-2-5',
        'e-muse',
        'f-minimax-h3',
        'g-seedance-2-5',
        'h-seedance-2-0',
        'i-wan3.0-fast',
      ],
      expected: [
        'd-seedance-2-5',
        'g-seedance-2-5',
        'b-seedance-2-0',
        'h-seedance-2-0',
        'c-wan-3-0',
        'i-wan3.0-fast',
        'a-minimax-h3',
        'f-minimax-h3',
        'e-muse',
      ],
      families: ['Seedance 2.5', 'Seedance 2.0', 'Wan', 'MiniMax', 'Muse'],
    },
    {
      category: 'image',
      names: [
        'a-seedream-5-0',
        'b-image-2',
        'c-seedream-4-5',
        'd-image-2',
        'e-seedream-5-0',
      ],
      expected: [
        'b-image-2',
        'd-image-2',
        'c-seedream-4-5',
        'a-seedream-5-0',
        'e-seedream-5-0',
      ],
      families: ['Image 2', 'Seedream 4.5', 'Seedream 5.0'],
    },
    {
      category: 'chat',
      names: [
        'a-claude-sonnet-4',
        'b-gpt-4o',
        'c-deepseek-chat',
        'd-claude-opus-4',
        'e-gpt-4o-mini',
      ],
      expected: [
        'a-claude-sonnet-4',
        'd-claude-opus-4',
        'c-deepseek-chat',
        'b-gpt-4o',
        'e-gpt-4o-mini',
      ],
      families: ['Claude', 'DeepSeek', 'GPT'],
    },
  ])(
    'groups interleaved $category models by series by default and uses the same filter order',
    async ({ category, names, expected, families }) => {
      const catalog = names.map((model_name, id) => ({
        ...models[0],
        id,
        model_name,
        square_category: category,
      }))
      renderCatalog(`/pricing/?category=${category}`, catalog)
      const list = await screen.findByRole('list', { name: 'Visible models' })

      expect(
        within(list)
          .getAllByRole('listitem')
          .map((item) => item.textContent)
      ).toEqual(expected)
      expect(
        within(screen.getByRole('group', { name: 'Model series' }))
          .getAllByRole('button')
          .flatMap((option) => {
            const family = families.find((name) =>
              option.textContent?.startsWith(name)
            )
            return family ? [family] : []
          })
      ).toEqual(families)
      expect(catalog.map((model) => model.model_name)).toEqual(names)
    }
  )

  it('uses custom series for sorting, leaves unassigned models last, and preserves order within each series', () => {
    const catalog = applyModelSquareDisplay(
      ['a-seedance-2-5', 'b-unknown', 'c-minimax', 'd-custom', 'e-custom'].map(
        (model_name) => ({ ...models[0], model_name })
      ),
      {
        order: [
          'e-custom',
          'b-unknown',
          'c-minimax',
          'd-custom',
          'a-seedance-2-5',
        ],
        models: {
          'a-seedance-2-5': { family: 'Editorial' },
          'd-custom': { family: 'Seedance 2.5' },
          'e-custom': { family: 'Seedance 2.5' },
        },
      }
    )

    expect(
      sortModels(catalog, 'family').map((model) => model.model_name)
    ).toEqual([
      'e-custom',
      'd-custom',
      'c-minimax',
      'a-seedance-2-5',
      'b-unknown',
    ])
    expect(sortModels([], 'family')).toEqual([])
    expect(sortModels([catalog[0]], 'family')).toEqual([catalog[0]])
  })

  it('switches from series sorting to recommended and name order and can restore the default series order', async () => {
    const catalog = applyModelSquareDisplay(models, {
      order: ['old-seedance-2-0', 'second-seedance2.5', 'first-seedance-2-5'],
      models: {},
    })
    renderCatalog('/pricing/?category=video', catalog)
    const user = userEvent.setup()
    const list = await screen.findByRole('list', { name: 'Visible models' })
    expect(
      within(list)
        .getAllByRole('listitem')
        .map((item) => item.textContent)
    ).toEqual(['second-seedance2.5', 'first-seedance-2-5', 'old-seedance-2-0'])

    await user.click(screen.getByRole('button', { name: 'Recommended order' }))
    expect(
      within(list)
        .getAllByRole('listitem')
        .map((item) => item.textContent)
    ).toEqual(['old-seedance-2-0', 'second-seedance2.5', 'first-seedance-2-5'])
    await user.click(screen.getByRole('button', { name: 'Name order' }))
    expect(
      within(list)
        .getAllByRole('listitem')
        .map((item) => item.textContent)
    ).toEqual(['first-seedance-2-5', 'old-seedance-2-0', 'second-seedance2.5'])
    await user.click(screen.getByRole('button', { name: 'Series order' }))
    expect(
      within(list)
        .getAllByRole('listitem')
        .map((item) => item.textContent)
    ).toEqual(['second-seedance2.5', 'first-seedance-2-5', 'old-seedance-2-0'])
  })

  it('selects visible series choices and resets within the active category', async () => {
    renderCatalog()
    const user = userEvent.setup()
    await screen.findByRole('button', { name: /^Videos\s*3$/ })
    expect(
      screen.queryByRole('button', { name: /^Image 2/ })
    ).not.toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /^Seedance 2.5/ }))
    expect(
      within(screen.getByRole('list', { name: 'Visible models' })).getAllByRole(
        'listitem'
      )
    ).toHaveLength(2)
    await user.click(screen.getByRole('button', { name: 'Reset' }))
    expect(
      screen.getByRole('button', { name: /^Videos\s*3$/ })
    ).toHaveAttribute('aria-pressed', 'true')
    expect(
      within(screen.getByRole('list', { name: 'Visible models' })).getAllByRole(
        'listitem'
      )
    ).toHaveLength(3)
    await user.click(screen.getByRole('button', { name: /^Images\s*1$/ }))
    expect(screen.getByRole('button', { name: /^Image 2/ })).toBeVisible()
  })

  it('loads a series query and updates its count with search', async () => {
    renderCatalog('/pricing/?category=video&family=series%3ASeedance%202.5')
    const user = userEvent.setup()
    const family = await screen.findByRole('button', { name: /^Seedance 2.5/ })
    expect(family).toHaveAttribute('aria-pressed', 'true')
    await user.type(
      screen.getByRole('textbox', { name: 'Search models' }),
      'first'
    )
    await waitFor(() => expect(family).toHaveTextContent(/Seedance 2.5\s*1/))
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
