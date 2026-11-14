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
import { ExternalLink, Video } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { INFINITE_CANVAS_URL } from '@/lib/constants'

export function InfiniteCanvasCard() {
  const { t } = useTranslation()

  return (
    <Card className='mt-6 w-full max-w-xl gap-3 bg-blue-500/5 py-4 ring-blue-500/15 dark:bg-blue-400/5 dark:ring-blue-400/20'>
      <CardHeader className='flex flex-row items-start gap-3'>
        <IconBadge tone='info'>
          <Video aria-hidden='true' />
        </IconBadge>
        <div className='min-w-0 space-y-1'>
          <CardTitle>
            <h2 className='text-sm font-semibold'>
              {t('Want to generate videos online?')}
            </h2>
          </CardTitle>
          <CardDescription className='text-xs leading-relaxed'>
            {t('Open Infinite Canvas and start creating.')}
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent>
        <Button
          variant='outline'
          role='link'
          className='h-auto min-h-8 max-w-full justify-start py-1.5 text-left whitespace-normal'
          render={
            <a
              href={INFINITE_CANVAS_URL}
              target='_blank'
              rel='noopener noreferrer'
            />
          }
        >
          {t('Open Infinite Canvas')}
          <ExternalLink aria-hidden='true' className='size-3.5' />
        </Button>
      </CardContent>
    </Card>
  )
}
