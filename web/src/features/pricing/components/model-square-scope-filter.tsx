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
import { useTranslation } from 'react-i18next'

import { ComboboxInput } from '@/components/ui/combobox-input'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import {
  countModelCategories,
  getModelCategoryLabels,
  MODEL_CATEGORIES,
  type ModelCategoryFilter,
} from '../lib/model-categories'
import { ALL_MODEL_FAMILIES, getModelFamilies } from '../lib/model-families'
import type { PricingModel } from '../types'

export function ModelSquareScopeFilter(props: {
  models: PricingModel[]
  categoryModels: PricingModel[]
  category: ModelCategoryFilter
  family: string
  onCategoryChange: (category: ModelCategoryFilter) => void
  onFamilyChange: (family: string) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const counts = countModelCategories(props.models)
  const labels = getModelCategoryLabels(t)
  const families = getModelFamilies(props.categoryModels)
  if (
    props.family !== ALL_MODEL_FAMILIES &&
    !families.some((family) => family.value === props.family)
  ) {
    families.push({
      value: props.family,
      label: props.family.startsWith('series:')
        ? props.family.slice(7)
        : undefined,
      count: 0,
    })
  }
  const items = [...MODEL_CATEGORIES, 'all' as const].map((value) => ({
    value,
    label: `${labels[value]} · ${formatNumber(counts[value], locale)}`,
  }))
  return (
    <div className='space-y-2 rounded-xl border p-3'>
      <div className='grid min-w-0 gap-3 sm:grid-cols-[220px_minmax(0,1fr)]'>
        <Field>
          <FieldLabel htmlFor='square-scope-category'>
            {t('Ordering scope')}
          </FieldLabel>
          <Select
            items={items}
            value={props.category}
            onValueChange={(value) => {
              if (
                value === 'all' ||
                MODEL_CATEGORIES.some((category) => category === value)
              ) {
                props.onCategoryChange(value as ModelCategoryFilter)
              }
            }}
          >
            <SelectTrigger id='square-scope-category' className='w-full'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {items.map((item) => (
                <SelectItem key={item.value} value={item.value}>
                  {item.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </Field>
        <Field>
          <FieldLabel htmlFor='square-scope-family'>
            {t('Filter by model series')}
          </FieldLabel>
          <ComboboxInput
            id='square-scope-family'
            value={props.family}
            options={[
              { value: ALL_MODEL_FAMILIES, label: t('All') },
              ...families.map((family) => ({
                value: family.value,
                label: `${family.label ?? t('Unassigned series')} · ${formatNumber(family.count, locale)}`,
              })),
            ]}
            onValueChange={props.onFamilyChange}
            emptyText='No matching items'
            openOnFocus={false}
          />
        </Field>
      </div>
      <FieldDescription>
        {t(
          'Reordering only changes models in the current category and series. Other models keep their positions.'
        )}
      </FieldDescription>
    </div>
  )
}
