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
  Clapperboard,
  Image,
  LayoutGrid,
  MessageSquare,
  Shapes,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import {
  getModelCategoryLabels,
  MODEL_CATEGORIES,
  type ModelCategoryCounts,
  type ModelCategoryFilter as CategoryFilter,
} from '../lib/model-categories'

const categoryIcons = {
  video: Clapperboard,
  image: Image,
  chat: MessageSquare,
  other: Shapes,
  all: LayoutGrid,
}

export function ModelCategoryFilter(props: {
  value: CategoryFilter
  counts: ModelCategoryCounts
  onChange: (category: CategoryFilter) => void
}) {
  const { t, i18n } = useTranslation()
  const labels = getModelCategoryLabels(t)
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)

  return (
    <nav
      aria-label={t('Model categories')}
      className='stolas-catalog-categories'
    >
      <ToggleGroup
        value={[props.value]}
        onValueChange={(values) => {
          const category = values[0]
          if (
            category === 'all' ||
            MODEL_CATEGORIES.some((value) => value === category)
          ) {
            props.onChange(category as CategoryFilter)
          }
        }}
        spacing={1}
        aria-label={t('Model categories')}
        className='stolas-catalog-category-list'
      >
        {['all' as const, ...MODEL_CATEGORIES].map((category) => {
          const Icon = categoryIcons[category]
          return (
            <ToggleGroupItem
              key={category}
              value={category}
              className='stolas-catalog-category'
            >
              <Icon data-icon='inline-start' aria-hidden='true' />
              {labels[category]}
              <span className='text-muted-foreground ml-0.5 text-xs tabular-nums sm:text-sm'>
                {formatNumber(props.counts[category], locale)}
              </span>
            </ToggleGroupItem>
          )
        })}
      </ToggleGroup>
    </nav>
  )
}
