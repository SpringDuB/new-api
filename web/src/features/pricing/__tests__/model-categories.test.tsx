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
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { Button } from '@/components/ui/button'

import { ModelCategoryFilter } from '../components/model-category-filter'
import { SearchBar } from '../components/search-bar'
import { useFilters } from '../hooks/use-filters'
import {
  countModelCategories,
  getModelCategory,
  inferModelCategory,
  type ModelCategory,
} from '../lib/model-categories'
import { applyModelSquareDisplay } from '../lib/model-square-display'
import type { PricingModel } from '../types'

const base: PricingModel = {
  id: 1,
  model_name: 'example',
  quota_type: 1,
  model_ratio: 1,
  completion_ratio: 1,
  enable_groups: ['default'],
}
const catalog: PricingModel[] = [
  {
    ...base,
    model_name: 'studio-video',
    square_category: 'video',
    supported_endpoint_types: ['openai'],
    tags: 'image',
  },
  {
    ...base,
    id: 2,
    model_name: 'studio-image',
    supported_endpoint_types: ['image-generation'],
  },
  {
    ...base,
    id: 3,
    model_name: 'chat-model',
    supported_endpoint_types: ['openai'],
    tags: 'chat',
  },
  { ...base, id: 4, model_name: 'unknown' },
]

function CategoryCatalog() {
  const filters = useFilters(catalog)
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
      <Button onClick={filters.clearFilters}>Reset</Button>
      <ul aria-label='Visible models'>
        {filters.filteredModels.map((model) => (
          <li key={model.model_name}>{model.model_name}</li>
        ))}
      </ul>
    </>
  )
}

function renderCatalog(path = '/pricing/') {
  const root = createRootRoute()
  const route = createRoute({
    getParentRoute: () => root,
    path: '/pricing/',
    component: CategoryCatalog,
  })
  const router = createRouter({
    routeTree: root.addChildren([route]),
    history: createMemoryHistory({ initialEntries: [path] }),
  })
  return render(<RouterProvider router={router} />)
}

describe('model categories', () => {
  it.each<{
    label: string
    data: Partial<PricingModel>
    expected: ModelCategory
  }>([
    {
      label: 'plugin video on generic chat endpoint with image input tags',
      data: {
        square_category: 'video',
        supported_endpoint_types: ['openai'],
        tags: 'image,audio',
      },
      expected: 'video',
    },
    {
      label: 'video endpoint',
      data: { supported_endpoint_types: ['openai-video'] },
      expected: 'video',
    },
    {
      label: 'image endpoint',
      data: { supported_endpoint_types: ['image-generation', 'openai'] },
      expected: 'image',
    },
    {
      label: 'image output from a multimodal endpoint',
      data: {
        output_modalities: ['text', 'image'],
        supported_endpoint_types: ['gemini'],
      },
      expected: 'image',
    },
    {
      label: 'chat with reference images and video',
      data: {
        input_modalities: ['image', 'video'],
        output_modalities: ['text'],
        tags: 'image,video',
      },
      expected: 'chat',
    },
    {
      label: 'audio output',
      data: { output_modalities: ['audio'] },
      expected: 'other',
    },
    {
      label: 'embeddings',
      data: { supported_endpoint_types: ['embeddings'] },
      expected: 'other',
    },
    {
      label: 'rerank',
      data: { supported_endpoint_types: ['jina-rerank'] },
      expected: 'other',
    },
    {
      label: 'explicit generation metadata',
      data: { tags: 'image-to-video', supported_endpoint_types: ['openai'] },
      expected: 'video',
    },
    {
      label: 'chat name containing video',
      data: {
        model_name: 'video-expert',
        supported_endpoint_types: ['openai'],
        tags: 'chat,video',
      },
      expected: 'chat',
    },
    { label: 'no metadata', data: {}, expected: 'other' },
    {
      label: 'generic endpoint without output metadata',
      data: { supported_endpoint_types: ['openai'] },
      expected: 'other',
    },
  ])('classifies $label as $expected', ({ data, expected }) => {
    expect(inferModelCategory({ ...base, ...data })).toBe(expected)
  })

  it('honors a manual category without changing automatic inference or using promotion tags', () => {
    const [manual] = applyModelSquareDisplay([catalog[0]], {
      order: [],
      models: {
        'studio-video': { category: 'other', tags: ['image-generation'] },
      },
    })
    expect(getModelCategory(manual)).toBe('other')
    expect(inferModelCategory(manual)).toBe('video')
    const [automatic] = applyModelSquareDisplay([base], {
      order: [],
      models: { example: { tags: ['video-generation'] } },
    })
    expect(getModelCategory(automatic)).toBe('other')
  })

  it('counts each model once, including empty and single-model catalogs', () => {
    expect(countModelCategories(catalog)).toEqual({
      all: 4,
      video: 1,
      image: 1,
      chat: 1,
      other: 1,
    })
    expect(countModelCategories([])).toEqual({
      all: 0,
      video: 0,
      image: 0,
      chat: 0,
      other: 0,
    })
    expect(countModelCategories([catalog[0]])).toEqual({
      all: 1,
      video: 1,
      image: 0,
      chat: 0,
      other: 0,
    })
  })

  it('filters from a category query, keeps the selected category pressed and resets only filters within that category', async () => {
    renderCatalog('/pricing/?category=video')
    const user = userEvent.setup()
    const video = await screen.findByRole('button', { name: /^Videos\s*1$/ })
    expect(video).toHaveAttribute('aria-pressed', 'true')
    expect(
      screen.getByRole('list', { name: 'Visible models' })
    ).toHaveTextContent('studio-video')
    expect(screen.queryByText('studio-image')).not.toBeInTheDocument()
    await user.click(video)
    expect(video).toHaveAttribute('aria-pressed', 'true')
    await user.click(screen.getByRole('button', { name: 'Reset' }))
    expect(video).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getAllByRole('listitem')).toHaveLength(1)
    await user.click(screen.getByRole('button', { name: /^All\s*4$/ }))
    expect(screen.getAllByRole('listitem')).toHaveLength(4)
  })

  it('updates counts with search and permits keyboard category selection with empty results', async () => {
    renderCatalog()
    const user = userEvent.setup()
    await user.type(
      await screen.findByRole('textbox', { name: 'Search models' }),
      'studio'
    )
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /^All\s*2$/ })).toBeVisible()
    )
    const chat = screen.getByRole('button', { name: /^Conversations\s*0$/ })
    chat.focus()
    await user.keyboard('{Enter}')
    expect(chat).toHaveAttribute('aria-pressed', 'true')
    expect(screen.queryAllByRole('listitem')).toHaveLength(0)
    await user.click(screen.getByRole('button', { name: /^Images\s*1$/ }))
    expect(screen.getAllByRole('listitem')).toHaveLength(1)
    expect(screen.getByRole('listitem')).toHaveTextContent('studio-image')
  })
})
