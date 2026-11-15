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

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import {
  getModelCategoryLabels,
  MODEL_CATEGORIES,
  type ModelCategoryChoice,
} from '../lib/model-categories'

export function ModelCategorySelect(props: {
  id: string
  value: ModelCategoryChoice
  onChange: (category: ModelCategoryChoice) => void
}) {
  const { t } = useTranslation()
  const labels = getModelCategoryLabels(t)
  const items = [
    { value: 'auto', label: t('Automatic classification') },
    ...MODEL_CATEGORIES.map((category) => ({
      value: category,
      label: labels[category],
    })),
  ]
  return (
    <Select
      items={items}
      value={props.value}
      onValueChange={(category) => {
        if (
          category === 'auto' ||
          MODEL_CATEGORIES.some((value) => value === category)
        ) {
          props.onChange(category as ModelCategoryChoice)
        }
      }}
    >
      <SelectTrigger id={props.id} className='w-full'>
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
  )
}
