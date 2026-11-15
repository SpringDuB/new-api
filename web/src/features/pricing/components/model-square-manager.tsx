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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Dialog } from '@/components/dialog'
import { EmptyState } from '@/components/empty-state'
import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { ComboboxInput } from '@/components/ui/combobox-input'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  getServerErrorMessage,
  requireServerSuccess,
} from '@/lib/server-error-message'

import { getModelSquareDisplay, saveModelSquareDisplay } from '../api'
import {
  getModelCategoryLabels,
  getModelCategory,
  inferModelCategory,
  type ModelCategoryChoice,
  type ModelCategoryFilter as CategoryFilter,
} from '../lib/model-categories'
import {
  ALL_MODEL_FAMILIES,
  getModelFamilies,
  getModelFamilyKey,
  inferModelFamily,
} from '../lib/model-families'
import {
  applyModelSquareDisplay,
  EMPTY_MODEL_SQUARE_DISPLAY,
  reorderModelSquareDisplay,
  type ModelSquareDecoration,
  type ModelSquareDisplay,
} from '../lib/model-square-display'
import { modelSquareDisplaySchema } from '../lib/model-square-form'
import type { PricingModel } from '../types'
import { ModelCard } from './model-card'
import { ModelCategorySelect } from './model-category-select'
import { ModelSquareBulkCategoryEditor } from './model-square-bulk-category-editor'
import { ModelSquareDecorationEditor } from './model-square-decoration-editor'
import { ModelSquareOrderEditor } from './model-square-order-editor'
import { ModelSquareScopeFilter } from './model-square-scope-filter'

export default function ModelSquareManager(props: {
  models: PricingModel[]
  onClose: () => void
  priceRate: number
  usdExchangeRate: number
  initialCategory?: CategoryFilter
  initialFamily?: string
}) {
  const { t } = useTranslation()
  const categoryLabels = getModelCategoryLabels(t)
  const queryClient = useQueryClient()
  const [discardOpen, setDiscardOpen] = useState(false)
  const [selectedModelName, setSelectedName] = useState('')
  const [categoryScope, setCategoryScope] = useState<CategoryFilter>(
    props.initialCategory ?? 'all'
  )
  const [familyScope, setFamilyScope] = useState(
    props.initialFamily ?? ALL_MODEL_FAMILIES
  )
  const [position, setPosition] = useState('')
  const query = useQuery({
    queryKey: ['model-square-display'],
    queryFn: async () => requireServerSuccess(await getModelSquareDisplay()),
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  })
  const form = useForm<ModelSquareDisplay>({
    resolver: zodResolver(modelSquareDisplaySchema),
    defaultValues: EMPTY_MODEL_SQUARE_DISPLAY,
    values: query.data?.data ?? EMPTY_MODEL_SQUARE_DISPLAY,
    // Reopening uses cached settings while a fresh request is still in flight.
    resetOptions: { keepDirtyValues: true, keepDirty: true },
    mode: 'onChange',
  })
  const display = form.watch()
  const models = useMemo(
    () => applyModelSquareDisplay(props.models, display),
    [props.models, display]
  )
  const categoryModels = models.filter(
    (model) =>
      categoryScope === 'all' || getModelCategory(model) === categoryScope
  )
  const scopedModels = categoryModels.filter(
    (model) =>
      familyScope === ALL_MODEL_FAMILIES ||
      getModelFamilyKey(model) === familyScope
  )
  const names = scopedModels.map((model) => model.model_name)
  const selectedModel =
    models.find((model) => model.model_name === selectedModelName) ??
    scopedModels[0]
  const selectedName = selectedModel?.model_name ?? ''
  const familyOptions = getModelFamilies(models).flatMap((family) =>
    family.label ? [{ value: family.label, label: family.label }] : []
  )
  const selectedIndex = names.indexOf(selectedName)
  const save = useMutation({
    mutationFn: async (value: ModelSquareDisplay) =>
      requireServerSuccess(await saveModelSquareDisplay(value)),
    onSuccess: async (response) => {
      queryClient.setQueryData(['model-square-display'], response)
      await queryClient.invalidateQueries({ queryKey: ['pricing'] })
      toast.success(t('Model square display saved'))
      props.onClose()
    },
  })
  const reorder = (order: string[]) => {
    setSelectedName(selectedName)
    const currentOrder = reorderModelSquareDisplay(
      display,
      models.map((model) => model.model_name)
    )
    form.setValue(
      'order',
      reorderModelSquareDisplay(currentOrder, order).order,
      {
        shouldDirty: true,
        shouldValidate: true,
      }
    )
  }
  const updateDecoration = (decoration: ModelSquareDecoration) => {
    setSelectedName(selectedName)
    form.setValue(
      'models',
      { ...display.models, [selectedName]: decoration },
      { shouldDirty: true, shouldValidate: true }
    )
  }
  const updateCategories = (names: string[], category: ModelCategoryChoice) => {
    setSelectedName(selectedName)
    const decorations = { ...display.models }
    for (const name of names) {
      const decoration = { ...decorations[name] }
      if (category === 'auto') delete decoration.category
      else decoration.category = category
      decorations[name] = decoration
    }
    form.setValue('models', decorations, {
      shouldDirty: true,
      shouldValidate: true,
    })
  }
  const close = () => {
    if (save.isPending) return
    if (form.formState.isDirty) setDiscardOpen(true)
    else props.onClose()
  }
  const moveToPosition = () => {
    const target = Number(position)
    if (
      !Number.isInteger(target) ||
      target < 1 ||
      target > names.length ||
      selectedIndex < 0
    ) {
      return
    }
    const next = names.filter((name) => name !== selectedName)
    next.splice(target - 1, 0, selectedName)
    reorder(next)
    setPosition('')
  }

  return (
    <>
      <Dialog
        open
        onOpenChange={(open) => {
          if (!open) close()
        }}
        title={t('Manage model square display')}
        description={t(
          'Set the default order and customize model cards for everyone. Changes go live only after saving.'
        )}
        contentClassName='sm:max-w-6xl'
        contentHeight='min(70dvh, 800px)'
        footer={
          <>
            <Button variant='outline' onClick={close} disabled={save.isPending}>
              {t('Cancel')}
            </Button>
            <Button
              onClick={form.handleSubmit((value) => save.mutate(value))}
              disabled={
                !query.isSuccess ||
                !form.formState.isDirty ||
                !form.formState.isValid ||
                save.isPending
              }
            >
              {save.isPending ? t('Saving...') : t('Save changes')}
            </Button>
          </>
        }
      >
        {query.isPending && <LoadingState />}
        {query.isError && <ErrorState onRetry={() => query.refetch()} />}
        {save.isError && (
          <Alert variant='destructive'>
            <AlertDescription>
              {getServerErrorMessage(save.error, t('Operation failed'))}
            </AlertDescription>
          </Alert>
        )}
        {query.isSuccess && (
          <fieldset disabled={save.isPending} className='min-w-0 space-y-4'>
            <ModelSquareScopeFilter
              models={models}
              categoryModels={categoryModels}
              category={categoryScope}
              family={familyScope}
              onCategoryChange={(category) => {
                setCategoryScope(category)
                setFamilyScope(ALL_MODEL_FAMILIES)
                setSelectedName('')
                setPosition('')
              }}
              onFamilyChange={(family) => {
                setFamilyScope(family)
                setSelectedName('')
                setPosition('')
              }}
            />
            <ModelSquareBulkCategoryEditor
              key={`${categoryScope}:${familyScope}`}
              models={scopedModels}
              onApply={updateCategories}
              onApplyFamily={(names, family) => {
                setSelectedName(selectedName)
                const decorations = { ...display.models }
                for (const name of names) {
                  const decoration = { ...decorations[name] }
                  if (family.trim()) decoration.family = family.trim()
                  else delete decoration.family
                  decorations[name] = decoration
                }
                form.setValue('models', decorations, {
                  shouldDirty: true,
                  shouldValidate: true,
                })
              }}
            />
            {!form.formState.isValid && form.formState.isDirty && (
              <Alert variant='destructive'>
                <AlertDescription>
                  {t(
                    'Use up to 64 characters for series, 24 for badges, 120 for notices, and 10 tags of up to 32 characters each.'
                  )}
                </AlertDescription>
              </Alert>
            )}
            <div className='grid min-w-0 gap-5 lg:grid-cols-[minmax(230px,0.9fr)_minmax(0,1fr)_minmax(0,1fr)]'>
              <ModelSquareOrderEditor
                key={`${categoryScope}:${familyScope}`}
                names={names}
                selectedName={selectedName}
                onSelect={(name) => {
                  setSelectedName(name)
                  setPosition('')
                }}
                onReorder={reorder}
              />
              {selectedModel && (
                <>
                  <div className='min-w-0 space-y-4'>
                    <p className='font-mono text-sm font-semibold break-all'>
                      {selectedName}
                    </p>
                    <Field>
                      <FieldLabel htmlFor='square-category'>
                        {t('Model category')}
                      </FieldLabel>
                      <ModelCategorySelect
                        id='square-category'
                        value={display.models[selectedName]?.category ?? 'auto'}
                        onChange={(category) =>
                          updateCategories([selectedName], category)
                        }
                      />
                      <FieldDescription>
                        {t('Automatically detected: {{category}}', {
                          category:
                            categoryLabels[inferModelCategory(selectedModel)],
                        })}
                      </FieldDescription>
                    </Field>
                    <Field>
                      <FieldLabel htmlFor='square-family'>
                        {t('Model series')}
                      </FieldLabel>
                      <ComboboxInput
                        id='square-family'
                        value={display.models[selectedName]?.family ?? ''}
                        options={[
                          { value: '', label: t('Automatic grouping') },
                          ...familyOptions,
                        ]}
                        allowCustomValue
                        openOnFocus={false}
                        placeholder={t('Automatic grouping')}
                        emptyText='No matching items'
                        aria-invalid={Boolean(
                          form.formState.errors.models?.[selectedName]?.family
                        )}
                        onValueChange={(family) =>
                          updateDecoration({
                            ...display.models[selectedName],
                            family: family || undefined,
                          })
                        }
                      />
                      <FieldDescription>
                        {t('Automatically grouped as: {{series}}', {
                          series:
                            inferModelFamily(selectedName) ??
                            t('Unassigned series'),
                        })}
                      </FieldDescription>
                      <FieldDescription>
                        {t(
                          'Choose an existing series or enter a custom name. Clear the value to restore automatic grouping.'
                        )}
                      </FieldDescription>
                    </Field>
                    <div className='space-y-2 rounded-lg border p-3'>
                      <Label htmlFor='square-position'>
                        {t('Move to position')}
                      </Label>
                      <div className='flex items-center gap-2'>
                        <Input
                          id='square-position'
                          type='number'
                          min={1}
                          max={names.length}
                          step={1}
                          disabled={selectedIndex < 0}
                          placeholder={
                            selectedIndex < 0 ? '' : String(selectedIndex + 1)
                          }
                          value={position}
                          onChange={(event) => setPosition(event.target.value)}
                        />
                        <Button
                          variant='outline'
                          onClick={moveToPosition}
                          disabled={
                            selectedIndex < 0 ||
                            !Number.isInteger(Number(position)) ||
                            Number(position) < 1 ||
                            Number(position) > names.length
                          }
                        >
                          {t('Move')}
                        </Button>
                        <Button
                          variant='outline'
                          disabled={selectedIndex <= 0}
                          onClick={() =>
                            reorder([
                              selectedName,
                              ...names.filter((name) => name !== selectedName),
                            ])
                          }
                        >
                          {t('Move to top')}
                        </Button>
                      </div>
                      {selectedIndex < 0 && (
                        <FieldDescription>
                          {t(
                            'This model is outside the current ordering scope.'
                          )}
                        </FieldDescription>
                      )}
                    </div>
                    <ModelSquareDecorationEditor
                      value={display.models[selectedName] ?? {}}
                      onChange={updateDecoration}
                    />
                  </div>
                  <div className='min-w-0 space-y-3 lg:sticky lg:top-0 lg:self-start'>
                    <p className='text-sm font-semibold'>{t('Live preview')}</p>
                    <div className='pointer-events-none p-1' inert>
                      <ModelCard
                        model={selectedModel}
                        onClick={() => undefined}
                        priceRate={props.priceRate}
                        usdExchangeRate={props.usdExchangeRate}
                      />
                    </div>
                    <p className='text-muted-foreground text-xs'>
                      {t(
                        'Visitors can still choose name or price sorting. New models appear after your saved order.'
                      )}
                    </p>
                    <Button
                      variant='ghost'
                      onClick={() =>
                        updateDecoration({
                          category: display.models[selectedName]?.category,
                          family: display.models[selectedName]?.family,
                        })
                      }
                    >
                      {t('Clear card decoration')}
                    </Button>
                  </div>
                </>
              )}
            </div>
            {scopedModels.length === 0 && (
              <EmptyState title={t('No models found')} />
            )}
          </fieldset>
        )}
      </Dialog>
      <ConfirmDialog
        open={discardOpen}
        onOpenChange={setDiscardOpen}
        title={t('Discard changes?')}
        desc={t('Your unsaved model square changes will be lost.')}
        confirmText={t('Discard changes')}
        handleConfirm={props.onClose}
      />
    </>
  )
}
