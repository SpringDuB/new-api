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
import {
  ArrowRight01Icon,
  CodeIcon,
  Film01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { CanvasEntry } from '@/components/canvas-entry'
import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'
import { useAuthStore } from '@/stores/auth-store'

export function CreativeWorkspace() {
  const { t } = useTranslation()
  const user = useAuthStore((state) => state.auth.user)
  const { status } = useStatus()
  return (
    <section
      aria-labelledby='stolas-workspace-title'
      className='stolas-workspace flex flex-col gap-7'
    >
      <div className='flex flex-wrap items-start justify-between gap-4 pt-2 pb-1'>
        <div className='flex min-w-0 flex-col gap-3'>
          <p className='stolas-eyebrow'>STOLAS</p>
          <h2 id='stolas-workspace-title' className='text-3xl sm:text-4xl'>
            {t('Welcome back, {{name}}', {
              name: user?.display_name || user?.username || 'Stolas',
            })}
          </h2>
          <p className='text-muted-foreground text-sm'>
            {t('Make room for your next idea.')}
          </p>
        </div>
        <Button variant='outline' render={<Link to='/wallet' />}>
          {t('Wallet')}
        </Button>
      </div>
      <div className='grid gap-5 md:grid-cols-2'>
        <article className='stolas-launch-panel stolas-launch-studio flex flex-col items-start gap-5'>
          <span className='stolas-launch-number' aria-hidden='true'>
            01 / STUDIO
          </span>
          <HugeiconsIcon
            icon={Film01Icon}
            className='text-primary size-6'
            aria-hidden='true'
          />
          <h3 className='stolas-editorial text-3xl'>{t('Infinite Canvas')}</h3>
          <p className='text-muted-foreground text-sm leading-relaxed'>
            {t(
              'Continue your stories, scenes and visual experiments on the canvas.'
            )}
          </p>
          <div className='mt-auto pt-3'>
            <CanvasEntry />
          </div>
        </article>
        <article className='stolas-launch-panel flex flex-col items-start gap-5'>
          <span className='stolas-launch-number' aria-hidden='true'>
            02 / API
          </span>
          <HugeiconsIcon
            icon={CodeIcon}
            className='text-primary size-6'
            aria-hidden='true'
          />
          <h3 className='stolas-editorial text-3xl'>{t('API access')}</h3>
          <p className='text-muted-foreground text-sm leading-relaxed'>
            {t(
              'Manage your keys, test a model and understand your application usage.'
            )}
          </p>
          <div className='mt-auto flex flex-wrap gap-2 pt-3'>
            <Button role='link' variant='outline' render={<Link to='/keys' />}>
              {t('API Keys')}
              <HugeiconsIcon
                icon={ArrowRight01Icon}
                data-icon='inline-end'
                aria-hidden='true'
              />
            </Button>
            <Button
              role='link'
              variant='ghost'
              render={<Link to='/playground' />}
            >
              {t('Playground')}
            </Button>
            <Button
              role='link'
              variant='ghost'
              render={<Link to='/usage-logs' />}
            >
              {t('Usage Logs')}
            </Button>
          </div>
        </article>
      </div>
      {status?.canvas_enabled === true && (
        <p className='text-muted-foreground max-w-2xl text-xs leading-relaxed'>
          {t(
            'Your default API key connects the canvas securely. Manage or revoke access in API Keys.'
          )}
        </p>
      )}
    </section>
  )
}
