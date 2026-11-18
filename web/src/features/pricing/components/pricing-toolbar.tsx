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
import { Filter } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import { getSortLabels, type SortOption, type ViewMode } from '../constants'
import type { PricingModel, PricingVendor, TokenUnit } from '../types'
import { PricingDisplaySettings } from './pricing-display-settings'
import { PricingSidebar } from './pricing-sidebar'

export interface PricingToolbarProps {
  familyFilter: string
  onFamilyChange: (value: string) => void
  familyModels?: PricingModel[]
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
  quotaTypeFilter: string
  endpointTypeFilter: string
  vendorFilter: string
  groupFilter: string
  tagFilter: string
  onQuotaTypeChange: (value: string) => void
  onEndpointTypeChange: (value: string) => void
  onVendorChange: (value: string) => void
  onGroupChange: (value: string) => void
  onTagChange: (value: string) => void
  vendors: PricingVendor[]
  groups: string[]
  groupRatios?: Record<string, number>
  tags: string[]
  models: PricingModel[]
  hasActiveFilters: boolean
  activeFilterCount: number
  onClearFilters: () => void
}

export function PricingToolbar(props: PricingToolbarProps) {
  const { t, i18n } = useTranslation()
  const [filtersOpen, setFiltersOpen] = useState(false)
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const sortLabels = getSortLabels(t)
  const sortOptions = Object.entries(sortLabels).map(([value, label]) => ({
    value,
    label,
  }))

  return (
    <Sheet open={filtersOpen} onOpenChange={setFiltersOpen}>
      <div className='stolas-catalog-toolbar'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div className='text-muted-foreground flex items-baseline gap-1 text-sm'>
            <span className='text-foreground font-semibold tabular-nums'>
              {formatNumber(props.filteredCount, locale)}
            </span>
            <span>{props.filteredCount === 1 ? t('model') : t('models')}</span>
            {props.totalCount != null &&
              props.filteredCount !== props.totalCount && (
                <span className='text-muted-foreground/60 text-xs'>
                  / {formatNumber(props.totalCount, locale)}
                </span>
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
            <SheetTrigger
              render={<Button type='button' variant='ghost' size='sm' />}
            >
              <Filter data-icon='inline-start' />
              {t('Filters and display')}
              {props.activeFilterCount > 0 && (
                <span className='text-primary tabular-nums'>
                  ({formatNumber(props.activeFilterCount, locale)})
                </span>
              )}
            </SheetTrigger>
          </div>
        </div>

        <SheetContent
          side='right'
          className={sideDrawerContentClassName('sm:max-w-md')}
        >
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>{t('Filters and display')}</SheetTitle>
            <SheetDescription>
              {t('Filters apply to the current category.')}
            </SheetDescription>
          </SheetHeader>
          <Tabs defaultValue='filters' className='stolas-catalog-panel-tabs'>
            <TabsList variant='line' className='stolas-catalog-panel-tablist'>
              <TabsTrigger value='filters'>{t('Filter')}</TabsTrigger>
              <TabsTrigger value='display'>{t('Display')}</TabsTrigger>
            </TabsList>
            <TabsContent value='filters' className={sideDrawerFormClassName()}>
              <PricingSidebar
                familyFilter={props.familyFilter}
                onFamilyChange={props.onFamilyChange}
                familyModels={props.familyModels}
                quotaTypeFilter={props.quotaTypeFilter}
                endpointTypeFilter={props.endpointTypeFilter}
                vendorFilter={props.vendorFilter}
                groupFilter={props.groupFilter}
                tagFilter={props.tagFilter}
                onQuotaTypeChange={props.onQuotaTypeChange}
                onEndpointTypeChange={props.onEndpointTypeChange}
                onVendorChange={props.onVendorChange}
                onGroupChange={props.onGroupChange}
                onTagChange={props.onTagChange}
                vendors={props.vendors}
                groups={props.groups}
                groupRatios={props.groupRatios}
                tags={props.tags}
                models={props.models}
                hasActiveFilters={props.hasActiveFilters}
                onClearFilters={props.onClearFilters}
              />
            </TabsContent>
            <TabsContent value='display' className={sideDrawerFormClassName()}>
              <PricingDisplaySettings
                tokenUnit={props.tokenUnit}
                onTokenUnitChange={props.onTokenUnitChange}
                showRechargePrice={props.showRechargePrice}
                onRechargePriceChange={props.onRechargePriceChange}
                viewMode={props.viewMode}
                onViewModeChange={props.onViewModeChange}
              />
            </TabsContent>
          </Tabs>
        </SheetContent>
      </div>
    </Sheet>
  )
}
