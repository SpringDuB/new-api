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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  getServerErrorMessage,
  requireServerSuccess,
} from '@/lib/server-error-message'

import { getModelSquareDisplay, saveModelSquareDisplay } from '../api'
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
import { ModelSquareDecorationEditor } from './model-square-decoration-editor'
import { ModelSquareOrderEditor } from './model-square-order-editor'

export default function ModelSquareManager(props: {
  models: PricingModel[]
  onClose: () => void
  priceRate: number
  usdExchangeRate: number
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [discardOpen, setDiscardOpen] = useState(false)
  const [selectedName, setSelectedName] = useState(
    props.models[0]?.model_name ?? ''
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
  const names = models.map((model) => model.model_name)
  const selectedModel = models.find(
    (model) => model.model_name === selectedName
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
    const currentOrder = reorderModelSquareDisplay(display, names)
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
    form.setValue(
      'models',
      { ...display.models, [selectedName]: decoration },
      { shouldDirty: true, shouldValidate: true }
    )
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
            {!form.formState.isValid && form.formState.isDirty && (
              <Alert variant='destructive'>
                <AlertDescription>
                  {t(
                    'Use up to 24 characters for badges, 120 for notices, and 10 tags of up to 32 characters each.'
                  )}
                </AlertDescription>
              </Alert>
            )}
            <div className='grid min-w-0 gap-5 lg:grid-cols-[minmax(230px,0.9fr)_minmax(0,1fr)_minmax(0,1fr)]'>
              <ModelSquareOrderEditor
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
                          placeholder={String(selectedIndex + 1)}
                          value={position}
                          onChange={(event) => setPosition(event.target.value)}
                        />
                        <Button
                          variant='outline'
                          onClick={moveToPosition}
                          disabled={
                            !Number.isInteger(Number(position)) ||
                            Number(position) < 1 ||
                            Number(position) > names.length
                          }
                        >
                          {t('Move')}
                        </Button>
                        <Button
                          variant='outline'
                          disabled={selectedIndex === 0}
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
                      onClick={() => updateDecoration({})}
                    >
                      {t('Clear card decoration')}
                    </Button>
                  </div>
                </>
              )}
            </div>
            {models.length === 0 && <EmptyState title={t('No models found')} />}
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
