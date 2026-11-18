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
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Field,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from '@/components/ui/field'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'

import type { ViewMode } from '../constants'
import type { TokenUnit } from '../types'

export interface PricingDisplaySettingsProps {
  tokenUnit: TokenUnit
  onTokenUnitChange: (value: TokenUnit) => void
  showRechargePrice: boolean
  onRechargePriceChange: (value: boolean) => void
  viewMode: ViewMode
  onViewModeChange: (value: ViewMode) => void
}

export function PricingDisplaySettings(props: PricingDisplaySettingsProps) {
  const { t } = useTranslation()
  const id = useId()
  const fields = [
    {
      name: 'price',
      label: t('Price display mode'),
      value: props.showRechargePrice ? 'recharge' : 'standard',
      onChange: (value: string) =>
        props.onRechargePriceChange(value === 'recharge'),
      options: [
        { value: 'standard', label: t('Standard') },
        { value: 'recharge', label: t('Recharge') },
      ],
    },
    {
      name: 'unit',
      label: t('Token unit'),
      value: props.tokenUnit,
      onChange: (value: string) => {
        if (value === 'M' || value === 'K') props.onTokenUnitChange(value)
      },
      options: [
        { value: 'M', label: '/1M' },
        { value: 'K', label: '/1K' },
      ],
    },
    {
      name: 'layout',
      label: t('Layout'),
      value: props.viewMode,
      onChange: (value: string) => {
        if (value === 'card' || value === 'table') props.onViewModeChange(value)
      },
      options: [
        { value: 'card', label: t('List view') },
        { value: 'table', label: t('Table view') },
      ],
    },
  ]
  return (
    <FieldGroup className='stolas-catalog-display-settings'>
      {fields.map((field) => (
        <FieldSet key={field.name}>
          <FieldLegend variant='label' id={`${id}-${field.name}`}>
            {field.label}
          </FieldLegend>
          <RadioGroup
            aria-labelledby={`${id}-${field.name}`}
            value={field.value}
            onValueChange={field.onChange}
          >
            {field.options.map((option) => (
              <Field key={option.value} orientation='horizontal'>
                <RadioGroupItem
                  value={option.value}
                  id={`${id}-${field.name}-${option.value}`}
                />
                <FieldLabel htmlFor={`${id}-${field.name}-${option.value}`}>
                  {option.label}
                </FieldLabel>
              </Field>
            ))}
          </RadioGroup>
        </FieldSet>
      ))}
    </FieldGroup>
  )
}
