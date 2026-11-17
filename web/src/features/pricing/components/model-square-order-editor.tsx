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
import { motion, Reorder } from 'motion/react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { AutoGroupOrderItem } from '@/components/auto-group-order-item'
import { EmptyState } from '@/components/empty-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import { getSortLabels, SORT_OPTIONS, type SortOption } from '../constants'
import {
  ALL_MODEL_FAMILIES,
  getModelFamilies,
  getModelFamilyKey,
} from '../lib/model-families'
import type { PricingModel } from '../types'

export function ModelSquareOrderEditor(props: {
  models: PricingModel[]
  sortBy: SortOption
  onSortChange: (sortBy: SortOption) => void
  selectedName: string
  onSelect: (name: string) => void
  onReorder: (names: string[]) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const [search, setSearch] = useState('')
  const sortLabels = getSortLabels(t)
  const sortOptions = [SORT_OPTIONS.FAMILY, SORT_OPTIONS.RECOMMENDED].map(
    (value) => ({ value, label: sortLabels[value] })
  )
  const groups =
    props.sortBy === SORT_OPTIONS.FAMILY
      ? getModelFamilies(props.models).map((family) => ({
          value: family.value,
          label: family.label ?? t('Unassigned series'),
          names: props.models
            .filter((model) => getModelFamilyKey(model) === family.value)
            .map((model) => model.model_name),
        }))
      : [
          {
            value: ALL_MODEL_FAMILIES,
            label: undefined,
            names: props.models.map((model) => model.model_name),
          },
        ]
  const visibleGroups = groups
    .map((group) => ({
      ...group,
      matchingNames: group.names.filter((name) =>
        name.toLowerCase().includes(search.trim().toLowerCase())
      ),
    }))
    .filter((group) => group.matchingNames.length > 0)
  const move = (names: string[], index: number, direction: 'up' | 'down') => {
    const next = [...names]
    const target = index + (direction === 'up' ? -1 : 1)
    if (target < 0 || target >= next.length) return
    const name = next[index]
    next[index] = next[target]
    next[target] = name
    props.onReorder(next)
  }

  return (
    <div className='min-w-0 space-y-3'>
      <div className='space-y-2'>
        <Label htmlFor='square-sort'>{t('Sort')}</Label>
        <Select
          items={sortOptions}
          value={props.sortBy}
          onValueChange={(value) => {
            if (
              value === SORT_OPTIONS.FAMILY ||
              value === SORT_OPTIONS.RECOMMENDED
            ) {
              props.onSortChange(value)
            }
          }}
        >
          <SelectTrigger id='square-sort' className='w-full'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {sortOptions.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <Label htmlFor='square-search'>{t('Model order')}</Label>
      <Input
        id='square-search'
        placeholder={t('Search models...')}
        value={search}
        onChange={(event) => setSearch(event.target.value)}
      />
      <p className='text-muted-foreground text-xs'>
        {t(
          'Drag models or use the arrow buttons to reorder. Search preserves the positions of other models.'
        )}
      </p>
      <motion.div
        layoutScroll
        role='region'
        aria-label={t('Model order')}
        tabIndex={0}
        className='max-h-[48dvh] space-y-3 overflow-y-auto p-1'
      >
        {visibleGroups.map((group) => (
          <div key={group.value} className='space-y-2'>
            {group.label && (
              <h3 className='text-muted-foreground text-xs font-medium'>
                {group.label}
              </h3>
            )}
            <Reorder.Group
              axis='y'
              values={group.matchingNames}
              onReorder={props.onReorder}
              aria-label={
                group.label
                  ? `${t('Model order')}: ${group.label}`
                  : t('Model order')
              }
              className='space-y-2'
            >
              {group.matchingNames.map((name, index) => (
                <AutoGroupOrderItem
                  key={name}
                  group={name}
                  index={index}
                  count={group.matchingNames.length}
                  onMove={(index, direction) =>
                    move(group.matchingNames, index, direction)
                  }
                  leading={
                    <span className='text-muted-foreground w-6 shrink-0 text-center text-xs tabular-nums'>
                      {formatNumber(group.names.indexOf(name) + 1, locale)}
                    </span>
                  }
                >
                  <Button
                    variant={
                      name === props.selectedName ? 'secondary' : 'ghost'
                    }
                    size='sm'
                    aria-label={t('Edit {{name}}', { name })}
                    aria-pressed={name === props.selectedName}
                    onClick={() => props.onSelect(name)}
                  >
                    {t('Edit')}
                  </Button>
                </AutoGroupOrderItem>
              ))}
            </Reorder.Group>
          </div>
        ))}
      </motion.div>
      {visibleGroups.length === 0 && props.models.length > 0 && (
        <EmptyState title={t('No models found')} />
      )}
    </div>
  )
}
