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
import { describe, expect, it } from 'vitest'

import { SORT_OPTIONS } from '../constants'
import {
  extractAllTags,
  filterAndSortModels,
  filterBySearch,
  filterByTag,
  sortModels,
} from '../lib/filters'
import {
  applyModelSquareDisplay,
  reorderModelSquareDisplay,
} from '../lib/model-square-display'
import type { PricingModel } from '../types'

const models: PricingModel[] = ['alpha', 'beta', 'gamma'].map(
  (model_name, id) => ({
    id,
    model_name,
    quota_type: 1,
    model_price: id + 1,
    model_ratio: 1,
    completion_ratio: 1,
    enable_groups: ['default'],
    tags: 'original',
  })
)

describe('model square display', () => {
  it('applies a saved order and appends unconfigured models by name without changing the catalog', () => {
    const result = applyModelSquareDisplay(models, {
      order: ['gamma', 'missing', 'beta'],
      models: { gamma: { badge: 'Featured' } },
    })
    expect(result.map((model) => model.model_name)).toEqual([
      'gamma',
      'beta',
      'alpha',
    ])
    expect(models.map((model) => model.model_name)).toEqual([
      'alpha',
      'beta',
      'gamma',
    ])
    expect(result[0].square_display?.badge).toBe('Featured')
  })

  it('keeps recommended order after filtering but honors an explicit name or price sort', () => {
    const ordered = applyModelSquareDisplay(models, {
      order: ['gamma', 'beta', 'alpha'],
      models: {},
    })
    const result = filterAndSortModels(ordered, {
      search: 'a',
      vendor: 'all',
      group: 'all',
      quotaType: 'all',
      endpointType: 'all',
      tag: 'all',
      sortBy: SORT_OPTIONS.RECOMMENDED,
    })
    expect(result.map((model) => model.model_name)).toEqual([
      'gamma',
      'beta',
      'alpha',
    ])
    expect(
      sortModels(ordered, SORT_OPTIONS.NAME).map((model) => model.model_name)
    ).toEqual(['alpha', 'beta', 'gamma'])
    expect(
      sortModels(ordered, SORT_OPTIONS.REQUEST_PRICE_LOW).map(
        (model) => model.model_name
      )
    ).toEqual(['alpha', 'beta', 'gamma'])
  })

  it('merges additional tags without duplicates and makes them available to filtering', () => {
    const ordered = applyModelSquareDisplay(models, {
      order: [],
      models: { beta: { tags: ['original', 'featured'] } },
    })
    expect(ordered[1].tags).toEqual(['original', 'featured'])
    expect(models[1].tags).toBe('original')
    expect(
      filterAndSortModels(ordered, {
        search: '',
        vendor: 'all',
        group: 'all',
        quotaType: 'all',
        endpointType: 'all',
        tag: 'featured',
        sortBy: SORT_OPTIONS.RECOMMENDED,
      }).map((model) => model.model_name)
    ).toEqual(['beta'])
  })

  it('reorders only matching models and preserves unavailable entries and their decoration', () => {
    const display = {
      order: ['alpha', 'hidden', 'beta', 'gamma'],
      models: { hidden: { badge: 'Keep me' } },
    }
    expect(reorderModelSquareDisplay(display, ['gamma', 'alpha'])).toEqual({
      order: ['gamma', 'hidden', 'beta', 'alpha'],
      models: display.models,
    })
    expect(reorderModelSquareDisplay(display, ['new', 'beta']).order).toEqual([
      'alpha',
      'hidden',
      'new',
      'gamma',
      'beta',
    ])
  })

  it('preserves a compound additional tag as one filter option and matches the complete label', () => {
    const label = 'High quality; text|image,视频'
    const catalog = applyModelSquareDisplay([models[0]], {
      order: [],
      models: { alpha: { tags: [label] } },
    })

    expect(extractAllTags(catalog)).toEqual([label.toLowerCase(), 'original'])
    expect(filterByTag(catalog, label.toUpperCase())).toEqual(catalog)
    expect(filterByTag(catalog, 'High')).toEqual([])
    expect(filterBySearch(catalog, 'quality')).toEqual(catalog)
  })

  it('supports an empty catalog and a single model', () => {
    expect(applyModelSquareDisplay([])).toEqual([])
    expect(applyModelSquareDisplay([models[0]])[0].model_name).toBe('alpha')
  })
})
