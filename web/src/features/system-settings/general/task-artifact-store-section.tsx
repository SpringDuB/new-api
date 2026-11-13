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
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ErrorState } from '@/components/error-state'
import { LoadingState } from '@/components/loading-state'
import { PasswordInput } from '@/components/password-input'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { api } from '@/lib/api'
import {
  getServerErrorMessage,
  requireServerSuccess,
} from '@/lib/server-error-message'

import {
  SettingsForm,
  SettingsSwitchField,
} from '../components/settings-form-layout'
import { SettingsSection } from '../components/settings-section'
import { storeSchema, type StoreForm } from '../lib/task-artifact-store-schema'

interface StoreSettings {
  mode: 'upstream' | 'oss'
  endpoint: string
  bucket: string
  region: string
  prefix: string
  presign_ttl: number
  max_mb: number
  credentials_configured: boolean
}
const settingsKey = ['settings', 'task-artifact-store']

export function TaskArtifactStoreSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [testPassed, setTestPassed] = useState(false)
  const settings = useQuery({
    queryKey: settingsKey,
    queryFn: async () =>
      requireServerSuccess(
        (
          await api.get<{ success: boolean; data: StoreSettings }>(
            '/api/option/task_artifact_store'
          )
        ).data
      ).data,
  })
  const form = useForm<StoreForm>({
    resolver: zodResolver(storeSchema),
    defaultValues: {
      mode: 'upstream',
      endpoint: '',
      bucket: '',
      region: '',
      prefix: 'tasks/v1',
      presign_ttl: 900,
      max_mb: 512,
      access_key: '',
      secret_key: '',
      credentials_configured: false,
    },
  })
  const isDirty = form.formState.isDirty
  useEffect(() => {
    if (settings.data) {
      form.reset({ ...settings.data, access_key: '', secret_key: '' })
    }
  }, [settings.data, form])
  const save = useMutation({
    mutationFn: async (value: StoreForm) =>
      requireServerSuccess(
        (
          await api.put<{ success: boolean; data: StoreSettings }>(
            '/api/option/task_artifact_store',
            value
          )
        ).data
      ).data,
    onSuccess: (data) => {
      queryClient.setQueryData(settingsKey, data)
      form.reset({ ...data, access_key: '', secret_key: '' })
      setTestPassed(false)
      probe.reset()
      toast.success(t('Setting updated successfully'))
    },
  })
  const probe = useMutation({
    meta: { errorToast: false },
    mutationFn: async () =>
      requireServerSuccess(
        (await api.post('/api/option/task_artifact_store/test')).data
      ),
    onSuccess: () => setTestPassed(true),
  })
  const enabled = form.watch('mode') === 'oss'
  if (settings.isPending) return <LoadingState />
  if (settings.isError) {
    return <ErrorState onRetry={() => void settings.refetch()} />
  }

  return (
    <SettingsSection title={t('Task result storage')}>
      <p className='text-muted-foreground text-sm'>
        {t(
          'Save generated videos and images to private Aliyun OSS objects and return temporary signed links'
        )}
      </p>
      <Form {...form}>
        <SettingsForm
          onSubmit={form.handleSubmit((value) => save.mutate(value))}
        >
          <SettingsSwitchField
            controlId='task-oss-enabled'
            checked={enabled}
            onCheckedChange={(checked) =>
              form.setValue('mode', checked ? 'oss' : 'upstream', {
                shouldDirty: true,
              })
            }
            label={t('Enable Aliyun OSS storage')}
          />
          {(
            [
              ['endpoint', 'OSS endpoint'],
              ['bucket', 'Bucket'],
              ['region', 'OSS region'],
              ['prefix', 'Object key prefix'],
            ] as const
          ).map(([name, label]) => (
            <FormField
              key={name}
              control={form.control}
              name={name}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t(label)}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      disabled={!enabled || save.isPending}
                      placeholder={
                        name === 'endpoint'
                          ? 'https://oss-cn-hangzhou.aliyuncs.com'
                          : undefined
                      }
                    />
                  </FormControl>
                  {name === 'region' ? (
                    <FormDescription>
                      {t(
                        'Use a region ID such as cn-shanghai, without the oss- prefix'
                      )}
                    </FormDescription>
                  ) : null}
                  <FormMessage />
                </FormItem>
              )}
            />
          ))}
          {settings.data?.credentials_configured ? (
            <p className='text-muted-foreground text-xs'>
              {t('Credentials configured; leave blank to keep them')}
            </p>
          ) : null}
          {(
            [
              ['access_key', 'AccessKey ID'],
              ['secret_key', 'AccessKey Secret'],
            ] as const
          ).map(([name, label]) => (
            <FormField
              key={name}
              control={form.control}
              name={name}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t(label)}</FormLabel>
                  <FormControl>
                    <PasswordInput
                      {...field}
                      autoComplete='new-password'
                      disabled={!enabled || save.isPending}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          ))}
          {(
            [
              ['presign_ttl', 'Signed link lifetime (seconds)', 1, 604800],
              ['max_mb', 'Maximum object size (MiB)', 1, 4096],
            ] as const
          ).map(([name, label, min, max]) => (
            <FormField
              key={name}
              control={form.control}
              name={name}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t(label)}</FormLabel>
                  <FormControl>
                    <Input
                      {...field}
                      type='number'
                      min={min}
                      max={max}
                      disabled={save.isPending}
                      onChange={(event) =>
                        field.onChange(Number(event.target.value))
                      }
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          ))}
          <p className='text-muted-foreground text-xs'>
            {t(
              'Objects are retained until removed by your OSS bucket lifecycle policy'
            )}
          </p>
          <div className='flex flex-wrap gap-2'>
            <Button type='submit' disabled={save.isPending}>
              {t(save.isPending ? 'Saving...' : 'Save Changes')}
            </Button>
            <Button
              type='button'
              variant='outline'
              disabled={
                !enabled || isDirty || save.isPending || probe.isPending
              }
              onClick={() => {
                setTestPassed(false)
                probe.mutate()
              }}
            >
              {t('Test connection')}
            </Button>
          </div>
          {testPassed ? (
            <p role='status' className='text-sm'>
              {t('OSS connection test passed')}
            </p>
          ) : null}
          {probe.isError ? (
            <Alert variant='destructive'>
              <AlertTitle>{t('Connection failed')}</AlertTitle>
              <AlertDescription>
                {getServerErrorMessage(probe.error)}
              </AlertDescription>
            </Alert>
          ) : null}
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
