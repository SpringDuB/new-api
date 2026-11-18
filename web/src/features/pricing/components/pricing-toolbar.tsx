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
import { ChevronDown, RotateCcw, SlidersHorizontal } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import { getSortLabels, type SortOption, type ViewMode } from '../constants'
import type {
  ModelCategoryCounts,
  ModelCategoryFilter as CategoryFilter,
} from '../lib/model-categories'
import type { PricingModel, TokenUnit } from '../types'
import { ModelCategoryFilter } from './model-category-filter'
import { PricingDisplaySettings } from './pricing-display-settings'
import { PricingSidebar, type PricingSidebarProps } from './pricing-sidebar'

export interface PricingToolbarProps extends Omit<
  PricingSidebarProps,
  'showGroups' | 'className'
> {
  categoryFilter: CategoryFilter
  categoryCounts: ModelCategoryCounts
  onCategoryChange: (value: CategoryFilter) => void
  search?: React.ReactNode
  filteredCount: number
  totalCount?: number
  sortBy: string
  onSortChange: (value: string) => void
  tokenUnit: TokenUnit
  onTokenUnitChange: (value: TokenUnit) => void
  showRechargePrice: boolean
  onRechargePriceChange: (value: boolean) => void
  viewMode: ViewMode
  onViewModeChange: (value: ViewMode) => void
  hasActiveFilters: boolean
  activeFilterCount: number
  onClearFilters: () => void
  familyModels?: PricingModel[]
}

export function PricingToolbar(props: PricingToolbarProps) {
  const { t, i18n } = useTranslation()
  const [filtersExpanded, setFiltersExpanded] = useState(
    props.groupFilter !== 'all' || props.familyFilter !== 'all'
  )
  const [displayOpen, setDisplayOpen] = useState(false)
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const sortLabels = getSortLabels(t)
  const sortOptions = Object.entries(sortLabels).map(([value, label]) => ({
    value,
    label,
  }))
  const showGroups = filtersExpanded

  return (
    <Collapsible
      open={filtersExpanded}
      onOpenChange={setFiltersExpanded}
      className='stolas-catalog-toolbar'
    >
      <div className='stolas-catalog-discovery'>
        <ModelCategoryFilter
          value={props.categoryFilter}
          counts={props.categoryCounts}
          onChange={props.onCategoryChange}
        />
        <div className='stolas-catalog-discovery-search'>{props.search}</div>
        <CollapsibleTrigger
          render={<Button type='button' variant='ghost' size='sm' />}
        >
          {showGroups ? t('Fewer filters') : t('More filters')}
          {props.activeFilterCount > 0 && (
            <span className='text-primary'>({props.activeFilterCount})</span>
          )}
          {props.groupFilter !== 'all' && (
            <span className='stolas-catalog-active-group'>
              {props.groupFilter}
            </span>
          )}
          <ChevronDown
            aria-hidden='true'
            className={showGroups ? 'rotate-180' : undefined}
          />
        </CollapsibleTrigger>
      </div>
      <CollapsibleContent>
        <PricingSidebar
          familyFilter={props.familyFilter}
          onFamilyChange={props.onFamilyChange}
          familyModels={props.familyModels}
          groupFilter={props.groupFilter}
          onGroupChange={props.onGroupChange}
          groups={props.groups}
          models={props.models}
          showGroups
        />
      </CollapsibleContent>
      <div className='stolas-catalog-toolbar-bottom'>
        <div className='flex flex-wrap items-center gap-3'>
          <p className='text-muted-foreground text-sm' aria-live='polite'>
            <span className='text-foreground font-medium tabular-nums'>
              {formatNumber(props.filteredCount, locale)}
            </span>{' '}
            {props.filteredCount === 1 ? t('model') : t('models')}
          </p>
          {props.hasActiveFilters && (
            <Button
              type='button'
              variant='ghost'
              size='sm'
              onClick={props.onClearFilters}
            >
              <RotateCcw aria-hidden='true' />
              {t('Reset')}
            </Button>
          )}
        </div>
        <div className='stolas-catalog-toolbar-actions'>
          <Select
            items={sortOptions}
            value={props.sortBy}
            onValueChange={(value) => {
              if (value) props.onSortChange(value)
            }}
          >
            <SelectTrigger
              aria-label={t('Sort')}
              className='stolas-catalog-sort'
            >
              <SelectValue>
                {sortLabels[props.sortBy as SortOption] || t('Sort')}
              </SelectValue>
            </SelectTrigger>
            <SelectContent
              align='end'
              alignItemWithTrigger={false}
              className='min-w-56'
            >
              <SelectGroup>
                {sortOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <Button
            type='button'
            variant='ghost'
            size='sm'
            onClick={() => setDisplayOpen(true)}
          >
            <SlidersHorizontal aria-hidden='true' />
            {t('Display')}
          </Button>
        </div>
      </div>
      <Dialog
        open={displayOpen}
        onOpenChange={setDisplayOpen}
        title={t('Display')}
        contentClassName='sm:max-w-md'
      >
        <PricingDisplaySettings
          tokenUnit={props.tokenUnit}
          onTokenUnitChange={props.onTokenUnitChange}
          showRechargePrice={props.showRechargePrice}
          onRechargePriceChange={props.onRechargePriceChange}
          viewMode={props.viewMode}
          onViewModeChange={props.onViewModeChange}
        />
      </Dialog>
    </Collapsible>
  )
}
