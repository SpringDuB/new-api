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
import { RotateCcw } from 'lucide-react'
import { memo, useId, useMemo, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { ComboboxInput } from '@/components/ui/combobox-input'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
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
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import {
  ENDPOINT_TYPES,
  FILTER_ALL,
  QUOTA_TYPES,
  getEndpointTypeLabels,
  getQuotaTypeLabels,
} from '../constants'
import { getModelQuotaType, parseTags } from '../lib/filters'
import { getModelFamilies } from '../lib/model-families'
import type { PricingModel, PricingVendor } from '../types'

type FilterOption = {
  value: string
  label: string
  count?: number
  suffix?: string
  icon?: ReactNode
}

type FilterFieldProps = {
  title: string
  value: string
  options: FilterOption[]
  onChange: (value: string) => void
  searchable?: boolean
}

export interface PricingSidebarProps {
  familyFilter: string
  onFamilyChange: (value: string) => void
  familyModels?: PricingModel[]
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
  onClearFilters: () => void
  className?: string
}

function formatGroupRatio(ratio: number | undefined): string | undefined {
  if (ratio == null) return undefined
  const formatted = Number.isInteger(ratio)
    ? ratio.toString()
    : ratio.toFixed(3).replace(/0+$/, '').replace(/\.$/, '')
  return `x${formatted}`
}

function FilterField(props: FilterFieldProps) {
  const { t, i18n } = useTranslation()
  const id = useId()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const options = props.options.map((option) => {
    const detail =
      option.suffix ??
      (option.count == null ? undefined : formatNumber(option.count, locale))
    return {
      ...option,
      label: detail ? `${option.label} · ${detail}` : option.label,
    }
  })
  if (!options.some((option) => option.value === props.value)) {
    let label = props.value
    if (props.value.startsWith('series:')) label = props.value.slice(7)
    if (props.value === 'unassigned') label = t('Unassigned series')
    options.push({ value: props.value, label })
  }
  return (
    <Field>
      <FieldLabel htmlFor={id}>{props.title}</FieldLabel>
      {props.searchable ? (
        <ComboboxInput
          id={id}
          value={props.value}
          options={options}
          onValueChange={props.onChange}
          placeholder={t('Search...')}
          emptyText='No matching items'
          openOnFocus={false}
        />
      ) : (
        <Select
          items={options}
          value={props.value}
          onValueChange={(value) => {
            if (value != null) props.onChange(value)
          }}
        >
          <SelectTrigger id={id} className='w-full'>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {options.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      )}
    </Field>
  )
}

export const PricingSidebar = memo(function PricingSidebar(
  props: PricingSidebarProps
) {
  const { t } = useTranslation()
  const counts = useMemo(() => {
    const vendors = new Map<string, number>()
    const tags = new Map<string, number>()
    const endpoints = new Map<string, number>()
    const quotas = { token: 0, request: 0, task: 0 }
    for (const model of props.models) {
      if (model.vendor_name) {
        vendors.set(
          model.vendor_name,
          (vendors.get(model.vendor_name) ?? 0) + 1
        )
      }
      for (const tag of new Set(
        parseTags(model.tags).map((tag) => tag.toLowerCase())
      )) {
        tags.set(tag, (tags.get(tag) ?? 0) + 1)
      }
      for (const endpoint of new Set(model.supported_endpoint_types ?? [])) {
        endpoints.set(endpoint, (endpoints.get(endpoint) ?? 0) + 1)
      }
      const quotaType = getModelQuotaType(model)
      if (quotaType) quotas[quotaType]++
    }
    return { vendors, tags, endpoints, quotas }
  }, [props.models])
  const quotaTypeLabels = getQuotaTypeLabels(t)
  const endpointTypeLabels = getEndpointTypeLabels(t)
  const familyModels = props.familyModels ?? props.models
  const familyOptions: FilterOption[] = [
    { value: FILTER_ALL, label: t('All'), count: familyModels.length },
    ...getModelFamilies(familyModels).map((family) => ({
      ...family,
      label: family.label ?? t('Unassigned series'),
    })),
  ]

  const vendorOptions: FilterOption[] = [
    {
      value: FILTER_ALL,
      label: t('All Vendors'),
      count: props.models.length,
    },
    ...props.vendors
      .map((vendor) => ({
        value: vendor.name,
        label: vendor.name,
        count: counts.vendors.get(vendor.name) ?? 0,
        icon: vendor.icon ? getLobeIcon(vendor.icon, 14) : undefined,
      }))
      .filter((vendor) => vendor.count > 0),
  ]

  const groupOptions: FilterOption[] = [
    {
      value: FILTER_ALL,
      label: t('All Groups'),
    },
    ...props.groups.map((group) => ({
      value: group,
      label: group,
      suffix: formatGroupRatio(props.groupRatios?.[group]),
    })),
  ]

  const quotaOptions: FilterOption[] = [
    {
      value: QUOTA_TYPES.ALL,
      label: quotaTypeLabels[QUOTA_TYPES.ALL],
      count: props.models.length,
    },
    {
      value: QUOTA_TYPES.TOKEN,
      label: quotaTypeLabels[QUOTA_TYPES.TOKEN],
      count: counts.quotas.token,
    },
    {
      value: QUOTA_TYPES.REQUEST,
      label: quotaTypeLabels[QUOTA_TYPES.REQUEST],
      count: counts.quotas.request,
    },
    {
      value: QUOTA_TYPES.TASK,
      label: quotaTypeLabels[QUOTA_TYPES.TASK],
      count: counts.quotas.task,
    },
  ]

  const tagOptions: FilterOption[] = [
    {
      value: FILTER_ALL,
      label: t('All Tags'),
      count: props.models.length,
    },
    ...props.tags.map((tag) => ({
      value: tag,
      label: tag,
      count: counts.tags.get(tag.toLowerCase()) ?? 0,
    })),
  ]

  const endpointOptions: FilterOption[] = [
    {
      value: ENDPOINT_TYPES.ALL,
      label: endpointTypeLabels[ENDPOINT_TYPES.ALL],
      count: props.models.length,
    },
    ...Object.entries(endpointTypeLabels)
      .filter(([value]) => value !== ENDPOINT_TYPES.ALL)
      .map(([value, label]) => ({
        value,
        label,
        count: counts.endpoints.get(value) ?? 0,
      })),
  ]

  return (
    <section className={cn('flex min-w-0 flex-col gap-6', props.className)}>
      <FieldGroup>
        <FilterField
          title={t('Model series')}
          value={props.familyFilter}
          options={familyOptions}
          onChange={props.onFamilyChange}
          searchable
        />
        <FilterField
          title={t('Vendor')}
          value={props.vendorFilter}
          options={vendorOptions}
          onChange={props.onVendorChange}
          searchable
        />
        <FilterField
          title={t('Groups')}
          value={props.groupFilter}
          options={groupOptions}
          onChange={props.onGroupChange}
          searchable
        />
        <FilterField
          title={t('Model Tags')}
          value={props.tagFilter}
          options={tagOptions}
          onChange={props.onTagChange}
          searchable
        />
        <FilterField
          title={t('Pricing Type')}
          value={props.quotaTypeFilter}
          options={quotaOptions}
          onChange={props.onQuotaTypeChange}
        />
        <FilterField
          title={t('Endpoint Type')}
          value={props.endpointTypeFilter}
          options={endpointOptions}
          onChange={props.onEndpointTypeChange}
        />
      </FieldGroup>
      <Button
        type='button'
        variant='ghost'
        size='sm'
        onClick={props.onClearFilters}
        disabled={!props.hasActiveFilters}
        className='self-start'
      >
        <RotateCcw data-icon='inline-start' />
        {t('Reset')}
      </Button>
    </section>
  )
})
