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
import { Combobox } from '@/components/ui/combobox'
import { Field, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import type { TaskMatrixRow } from '@/features/pricing/lib/task-expr'
import { taskPriceLabel } from '@/features/pricing/lib/task-price-display'
import type { BillingUsageSchema } from '@/features/pricing/types'

const branchSchema = z.object({
  values: z.record(z.string(), z.string().trim().min(1).max(64)),
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
        fields.map(([field, definition]) => [
          field,
          definition.allowCustom ? '' : (definition.enum?.[0] ?? ''),
        ])
      ),
    },
  })
  const submit = form.handleSubmit(({ values }) => {
    const combination = values
    if (
      props.rows.some((row) =>
        fields.every(([field]) => row.combination[field] === combination[field])
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
      <form id={id} onSubmit={submit} className='space-y-3'>
        {fields.map(([field, definition]) => (
          <Field key={field}>
            <FieldLabel htmlFor={`${id}-${field}`}>
              {taskPriceLabel(definition.description, field, i18n.language)}
            </FieldLabel>
            {definition.allowCustom ? (
              <Input
                id={`${id}-${field}`}
                aria-label={t('{{field}} value', {
                  field: taskPriceLabel(
                    definition.description,
                    field,
                    i18n.language
                  ),
                })}
                maxLength={64}
                required
                {...form.register(`values.${field}`)}
              />
            ) : (
              <Controller
                control={form.control}
                name={`values.${field}`}
                render={({ field: input }) => (
                  <Combobox
                    id={`${id}-${field}`}
                    value={input.value}
                    options={(definition.enum ?? []).map((value) => ({
                      value,
                      label: value,
                    }))}
                    onValueChange={input.onChange}
                  />
                )}
              />
            )}
          </Field>
        ))}
        {form.formState.errors.root?.message && (
          <p role='alert' className='text-destructive text-xs'>
            {t(form.formState.errors.root.message)}
          </p>
        )}
      </form>
    </Dialog>
  )
}
