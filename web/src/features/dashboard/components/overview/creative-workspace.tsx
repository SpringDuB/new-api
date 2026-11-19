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
import { CodeIcon, Film01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { CanvasEntry } from '@/components/canvas-entry'
import { Button } from '@/components/ui/button'

export function CreativeWorkspace() {
  const { t } = useTranslation()
  return (
    <section aria-label={t('Quick actions')} className='stolas-launch-actions'>
      <div className='stolas-launch-action'>
        <h2 className='flex items-center gap-2 text-sm font-semibold'>
          <HugeiconsIcon
            icon={Film01Icon}
            className='text-primary size-5 shrink-0'
            aria-hidden='true'
          />
          {t('Infinite Canvas')}
        </h2>
        <CanvasEntry className='h-8 px-3 text-xs' />
      </div>
      <div className='stolas-launch-action'>
        <h2 className='flex items-center gap-2 text-sm font-semibold'>
          <HugeiconsIcon
            icon={CodeIcon}
            className='text-primary size-5 shrink-0'
            aria-hidden='true'
          />
          {t('API access')}
        </h2>
        <div className='flex flex-wrap items-center gap-2'>
          <Button
            role='link'
            size='sm'
            variant='outline'
            render={<Link to='/keys' />}
          >
            {t('API Keys')}
          </Button>
          <Button
            role='link'
            size='sm'
            variant='ghost'
            render={<Link to='/playground' />}
          >
            {t('Playground')}
          </Button>
        </div>
      </div>
    </section>
  )
}
