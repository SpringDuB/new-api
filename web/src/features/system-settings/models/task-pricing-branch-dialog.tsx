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
import { zodResolver } from '@hookform/resolvers/zod'
import { useId } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Combobox } from '@/components/ui/combobox'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import type { TaskMatrixRow } from '@/features/pricing/lib/task-expr'
import {
  taskEnumLabel,
  taskPriceLabel,
} from '@/features/pricing/lib/task-price-display'
import type { BillingUsageSchema } from '@/features/pricing/types'

const branchSchema = z.object({
  values: z
    .record(z.string(), z.string().trim().min(1, 'Required').max(64))
    .refine((values) => Object.keys(values).length > 0, {
      message: 'Select at least one condition.',
    }),
})

export function TaskPricingBranchDialog(props: {
  open: boolean
  onOpenChange: (open: boolean) => void
  schema: BillingUsageSchema
  rows: TaskMatrixRow[]
  onAdd: (combination: Record<string, string>) => void
}) {
  const { t, i18n } = useTranslation()
  const id = useId()
  const fields = Object.entries(props.schema).filter(
    ([, field]) => field.enum?.length
  )
  const form = useForm<z.infer<typeof branchSchema>>({
    resolver: zodResolver(branchSchema),
    defaultValues: {
      values: Object.fromEntries(
        fields
          .slice(0, 1)
          .map(([field, definition]) => [
            field,
            definition.allowCustom ? '' : (definition.enum?.[0] ?? ''),
          ])
      ),
    },
  })
  const submit = form.handleSubmit(({ values }) => {
    const combination = values
    const entries = Object.entries(combination)
    if (
      props.rows.some(
        (row) =>
          Object.keys(row.combination).length === entries.length &&
          entries.every(([field, value]) => row.combination[field] === value)
      )
    ) {
      form.setError('root', {
        message: 'A tier with these conditions already exists.',
      })
      return
    }
    props.onAdd(combination)
    props.onOpenChange(false)
    form.reset()
  })
  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Add pricing branch')}
      description={t('Choose one or more conditions for this branch.')}
      contentClassName='sm:max-w-md'
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button type='submit' form={id}>
            {t('Add tier')}
          </Button>
        </>
      }
    >
      <form id={id} onSubmit={submit}>
        <FieldGroup>
          <Controller
            control={form.control}
            name='values'
            render={({ field: input }) => (
              <FieldSet>
                <FieldLegend variant='label'>
                  {t('Tier conditions')}
                </FieldLegend>
                <FieldGroup>
                  {fields.map(([field, definition]) => {
                    const label = taskPriceLabel(
                      definition.description,
                      field,
                      i18n.language
                    )
                    const selected = Object.hasOwn(input.value, field)
                    const error = form.formState.errors.values?.[field]
                    const valueLabel = t('{{field}} value', { field: label })
                    return (
                      <Field key={field} data-invalid={Boolean(error)}>
                        <Field orientation='horizontal'>
                          <Checkbox
                            id={`${id}-${field}-selected`}
                            checked={selected}
                            onCheckedChange={(checked) => {
                              const values = { ...input.value }
                              if (checked) {
                                values[field] = definition.allowCustom
                                  ? ''
                                  : (definition.enum?.[0] ?? '')
                              } else {
                                delete values[field]
                              }
                              form.clearErrors()
                              input.onChange(values)
                            }}
                          />
                          <FieldLabel htmlFor={`${id}-${field}-selected`}>
                            {label}
                          </FieldLabel>
                        </Field>
                        {selected && (
                          <>
                            {definition.allowCustom ? (
                              <Input
                                id={`${id}-${field}`}
                                aria-label={valueLabel}
                                aria-invalid={Boolean(error)}
                                aria-describedby={
                                  error ? `${id}-${field}-error` : undefined
                                }
                                maxLength={64}
                                required
                                value={input.value[field]}
                                onChange={(event) =>
                                  input.onChange({
                                    ...input.value,
                                    [field]: event.target.value,
                                  })
                                }
                              />
                            ) : (
                              <Combobox
                                id={`${id}-${field}`}
                                aria-label={valueLabel}
                                value={input.value[field]}
                                options={(definition.enum ?? []).map(
                                  (value) => ({
                                    value,
                                    label: taskEnumLabel(
                                      definition,
                                      value,
                                      i18n.language
                                    ),
                                  })
                                )}
                                onValueChange={(value) =>
                                  input.onChange({
                                    ...input.value,
                                    [field]: value,
                                  })
                                }
                              />
                            )}
                            {error?.message && (
                              <FieldError id={`${id}-${field}-error`}>
                                {t(error.message)}
                              </FieldError>
                            )}
                          </>
                        )}
                      </Field>
                    )
                  })}
                </FieldGroup>
              </FieldSet>
            )}
          />
          {typeof form.formState.errors.values?.message === 'string' && (
            <FieldError>{t(form.formState.errors.values.message)}</FieldError>
          )}
          {form.formState.errors.root?.message && (
            <FieldError>{t(form.formState.errors.root.message)}</FieldError>
          )}
        </FieldGroup>
      </form>
    </Dialog>
  )
}
