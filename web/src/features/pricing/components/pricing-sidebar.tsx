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
import { memo } from 'react'
import { useTranslation } from 'react-i18next'

import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { getLobeIcon, getLobeIconNames } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { FILTER_ALL } from '../constants'
import { getModelFamilies, getModelFamilyKey } from '../lib/model-families'
import type { PricingModel } from '../types'

const AVAILABLE_ICON_BASES = new Set(
  getLobeIconNames().map((name) => name.split('.')[0])
)

export interface PricingSidebarProps {
  familyFilter: string
  onFamilyChange: (value: string) => void
  familyModels?: PricingModel[]
  groupFilter: string
  onGroupChange: (value: string) => void
  groups: string[]
  models: PricingModel[]
  showGroups?: boolean
  className?: string
}

export const PricingSidebar = memo(function PricingSidebar(
  props: PricingSidebarProps
) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const models = props.familyModels ?? props.models
  const familyIcons = new Map<string, string>()
  for (const model of models) {
    const family = getModelFamilyKey(model)
    const icon = [model.icon?.trim(), model.vendor_icon?.trim()].find(
      (candidate) =>
        candidate && AVAILABLE_ICON_BASES.has(candidate.split('.')[0])
    )
    if (icon && !familyIcons.has(family)) familyIcons.set(family, icon)
  }
  const families = [
    { value: FILTER_ALL, label: t('All'), count: models.length },
    ...getModelFamilies(models).map((family) => ({
      ...family,
      label: family.label ?? t('Unassigned series'),
    })),
  ]
  if (!families.some((family) => family.value === props.familyFilter)) {
    families.push({
      value: props.familyFilter,
      label: props.familyFilter.replace(/^series:/, ''),
      count: 0,
    })
  }
  const groups = [...new Set([FILTER_ALL, ...props.groups])]
  if (!groups.includes(props.groupFilter)) groups.push(props.groupFilter)

  return (
    <div className={cn('stolas-catalog-filter-fields', props.className)}>
      <div className='stolas-catalog-filter-row'>
        <p className='stolas-catalog-filter-label'>{t('Model series')}</p>
        <ToggleGroup
          aria-label={t('Model series')}
          value={[props.familyFilter]}
          onValueChange={(values) => {
            if (values[0]) props.onFamilyChange(values[0])
          }}
          spacing={2}
          className='stolas-catalog-choices'
        >
          {families.map((family) => {
            const icon = familyIcons.get(family.value)
            return (
              <ToggleGroupItem
                key={family.value}
                value={family.value}
                className='stolas-catalog-choice'
              >
                {icon && (
                  <span
                    data-slot='series-icon'
                    aria-hidden='true'
                    className='shrink-0'
                  >
                    {getLobeIcon(icon, 18)}
                  </span>
                )}
                <span className='min-w-0 flex-1 text-start break-words whitespace-normal'>
                  {family.label}
                </span>
                <span className='text-muted-foreground shrink-0 text-xs tabular-nums'>
                  {formatNumber(family.count, locale)}
                </span>
              </ToggleGroupItem>
            )
          })}
        </ToggleGroup>
      </div>
      {props.showGroups && (
        <div className='stolas-catalog-filter-row'>
          <p className='stolas-catalog-filter-label'>{t('Groups')}</p>
          <ToggleGroup
            aria-label={t('Groups')}
            value={[props.groupFilter]}
            onValueChange={(values) => {
              if (values[0]) props.onGroupChange(values[0])
            }}
            spacing={2}
            className='stolas-catalog-choices stolas-catalog-group-choices'
          >
            {groups.map((group) => (
              <ToggleGroupItem
                key={group}
                value={group}
                className='stolas-catalog-choice'
              >
                {group === FILTER_ALL ? t('All Groups') : group}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </div>
      )}
    </div>
  )
})
