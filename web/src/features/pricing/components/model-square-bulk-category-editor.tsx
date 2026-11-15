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
import { ChevronDown } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { MultiSelect } from '@/components/multi-select'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { ComboboxInput } from '@/components/ui/combobox-input'
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'

import {
  getModelCategory,
  getModelCategoryLabels,
  type ModelCategoryChoice,
} from '../lib/model-categories'
import { getModelFamilies } from '../lib/model-families'
import type { PricingModel } from '../types'
import { ModelCategorySelect } from './model-category-select'

export function ModelSquareBulkCategoryEditor(props: {
  models: PricingModel[]
  onApply: (names: string[], category: ModelCategoryChoice) => void
  onApplyFamily: (names: string[], family: string) => void
}) {
  const { t } = useTranslation()
  const [selected, setSelected] = useState<string[]>([])
  const [category, setCategory] = useState<ModelCategoryChoice>('auto')
  const [family, setFamily] = useState('')
  const labels = getModelCategoryLabels(t)
  const available = new Set(props.models.map((model) => model.model_name))
  const selectedNames = selected.filter((name) => available.has(name))

  return (
    <Collapsible className='rounded-xl border'>
      <CollapsibleTrigger
        render={
          <Button
            variant='ghost'
            className='group w-full justify-between rounded-xl px-3'
          />
        }
      >
        {t('Batch classification')}
        <ChevronDown
          className='size-4 transition-transform group-data-[panel-open]:rotate-180'
          aria-hidden='true'
        />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <FieldGroup className='p-3'>
          <div className='grid min-w-0 gap-3 sm:grid-cols-[minmax(0,1fr)_180px_auto] sm:items-end'>
            <Field className='min-w-0'>
              <FieldLabel htmlFor='square-bulk-models'>
                {t('Models')}
              </FieldLabel>
              <MultiSelect
                id='square-bulk-models'
                options={props.models.map((model) => ({
                  value: model.model_name,
                  label: model.model_name,
                  hint: labels[getModelCategory(model)],
                }))}
                selected={selectedNames}
                onChange={setSelected}
                placeholder={t('Search and select models...')}
                maxVisibleChips={2}
              />
            </Field>
            <Field>
              <FieldLabel htmlFor='square-bulk-category'>
                {t('Assign category')}
              </FieldLabel>
              <ModelCategorySelect
                id='square-bulk-category'
                value={category}
                onChange={setCategory}
              />
            </Field>
            <Button
              variant='outline'
              disabled={selectedNames.length === 0}
              onClick={() => props.onApply(selectedNames, category)}
            >
              {t('Apply to selected')}
            </Button>
          </div>
          <div className='grid min-w-0 gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end'>
            <Field>
              <FieldLabel htmlFor='square-bulk-family'>
                {t('Assign series')}
              </FieldLabel>
              <ComboboxInput
                id='square-bulk-family'
                value={family}
                options={[
                  { value: '', label: t('Automatic grouping') },
                  ...getModelFamilies(props.models).flatMap((item) =>
                    item.label ? [{ value: item.label, label: item.label }] : []
                  ),
                ]}
                allowCustomValue
                openOnFocus={false}
                placeholder={t('Automatic grouping')}
                emptyText='No matching items'
                aria-invalid={
                  [...family.trim()].length > 64 || /[\n\r\t]/.test(family)
                }
                onValueChange={setFamily}
              />
            </Field>
            <Button
              variant='outline'
              disabled={
                selectedNames.length === 0 ||
                [...family.trim()].length > 64 ||
                /[\n\r\t]/.test(family)
              }
              onClick={() => props.onApplyFamily(selectedNames, family)}
            >
              {t('Apply series to selected')}
            </Button>
          </div>
          <FieldDescription>
            {t(
              'Search and select multiple models, assign a category or series, then save changes to publish.'
            )}
          </FieldDescription>
        </FieldGroup>
      </CollapsibleContent>
    </Collapsible>
  )
}
