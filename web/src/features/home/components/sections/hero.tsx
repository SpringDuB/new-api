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
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'

import { StoryArt } from '../story-art'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  return (
    <main
      className={cn(
        'mx-auto max-w-7xl px-6 pt-28 pb-16 sm:px-10 lg:pt-36',
        props.className
      )}
    >
      <section className='grid items-center gap-14 pb-20 lg:grid-cols-[1.15fr_1fr] lg:gap-20'>
        <div className='flex flex-col items-start gap-7'>
          <p className='stolas-eyebrow'>
            {t('For the things only you can imagine')}
          </p>
          <h1 className='stolas-editorial max-w-3xl text-5xl leading-[1.1] sm:text-6xl xl:text-7xl'>
            {t('Stories take shape.')}
            <br />
            <span className='text-primary'>{t('Possibilities open up.')}</span>
          </h1>
          <p className='text-muted-foreground max-w-lg text-base leading-relaxed sm:text-lg'>
            {t(
              'Create films on an open canvas. Build products with a unified API. Stolas brings your ideas and AI models together.'
            )}
          </p>
          <div className='flex w-full flex-wrap gap-3'>
            <CanvasEntry className='h-12 px-6' />
            <Button
              variant='outline'
              size='lg'
              className='h-12 px-6'
              render={
                <Link
                  to={props.isAuthenticated ? '/keys' : '/sign-up'}
                  search={
                    props.isAuthenticated ? undefined : { redirect: '/keys' }
                  }
                />
              }
            >
              {t('Build with the API')}
              <HugeiconsIcon
                icon={ArrowRight01Icon}
                data-icon='inline-end'
                aria-hidden='true'
              />
            </Button>
          </div>
          <p className='text-muted-foreground text-xs'>
            {t('One account. One model wallet. Room for every idea.')}
          </p>
        </div>
        <StoryArt />
      </section>
      <Separator />
      <section aria-labelledby='stolas-workspaces' className='py-14 sm:py-20'>
        <div className='mb-10 flex flex-col justify-between gap-5 sm:flex-row sm:items-end'>
          <div className='flex flex-col gap-4'>
            <p className='stolas-eyebrow'>
              {t('Two ways in. Your direction.')}
            </p>
            <h2
              id='stolas-workspaces'
              className='stolas-editorial text-3xl sm:text-4xl'
            >
              {t('What would you like to make?')}
            </h2>
          </div>
          <Button variant='ghost' render={<Link to='/pricing' />}>
            {t('Explore available models')}
            <HugeiconsIcon
              icon={ArrowRight01Icon}
              data-icon='inline-end'
              aria-hidden='true'
            />
          </Button>
        </div>
        <div className='grid gap-8 md:grid-cols-2 md:gap-0 md:divide-x'>
          <article className='flex flex-col items-start gap-5 md:pr-12'>
            <HugeiconsIcon
              icon={Film01Icon}
              className='text-primary size-8'
              aria-hidden='true'
            />
            <p className='stolas-eyebrow'>{t('For creators')}</p>
            <h3 className='stolas-editorial text-3xl'>
              {t('From the first idea to the final frame.')}
            </h3>
            <p className='text-muted-foreground max-w-md text-sm leading-7'>
              {t(
                'Bring scripts, images and video together on an infinite canvas. Develop a scene, explore a visual direction and keep your story connected.'
              )}
            </p>
            <CanvasEntry variant='outline' />
          </article>
          <article className='flex flex-col items-start gap-5 border-t pt-8 md:border-t-0 md:pt-0 md:pl-12'>
            <HugeiconsIcon
              icon={CodeIcon}
              className='text-primary size-8'
              aria-hidden='true'
            />
            <p className='stolas-eyebrow'>{t('For developers')}</p>
            <h3 className='stolas-editorial text-3xl'>
              {t('A clear route from prototype to product.')}
            </h3>
            <p className='text-muted-foreground max-w-md text-sm leading-7'>
              {t(
                'Connect your application to the models you need. Manage API keys, inspect usage and track spending in the Console.'
              )}
            </p>
            <Button
              variant='outline'
              size='lg'
              render={<Link to='/dashboard' />}
            >
              {t('Console')}
              <HugeiconsIcon
                icon={ArrowRight01Icon}
                data-icon='inline-end'
                aria-hidden='true'
              />
            </Button>
          </article>
        </div>
      </section>
      <section className='bg-secondary grid gap-8 p-8 sm:p-12 lg:grid-cols-[1fr_1.2fr]'>
        <div className='flex flex-col gap-4'>
          <p className='stolas-eyebrow'>STOLAS / ACCOUNT</p>
          <h2 className='stolas-editorial text-3xl sm:text-4xl'>
            {t('Your work, connected.')}
          </h2>
          <p className='text-muted-foreground max-w-sm text-sm leading-7'>
            {t(
              'Move between the canvas and your developer tools with the same Stolas account.'
            )}
          </p>
        </div>
        <ol className='divide-border flex flex-col divide-y'>
          <li className='flex gap-5 pb-5'>
            <span className='text-primary font-mono text-xs'>01</span>
            <div>
              <h3 className='font-medium'>{t('Choose where to begin')}</h3>
              <p className='text-muted-foreground mt-2 text-sm'>
                {t('Start in the canvas or connect your own application.')}
              </p>
            </div>
          </li>
          <li className='flex gap-5 py-5'>
            <span className='text-primary font-mono text-xs'>02</span>
            <div>
              <h3 className='font-medium'>
                {t('Keep control of your access')}
              </h3>
              <p className='text-muted-foreground mt-2 text-sm'>
                {t('Manage model access and API credentials in your account.')}
              </p>
            </div>
          </li>
          <li className='flex gap-5 pt-5'>
            <span className='text-primary font-mono text-xs'>03</span>
            <div>
              <h3 className='font-medium'>{t('Follow every request')}</h3>
              <p className='text-muted-foreground mt-2 text-sm'>
                {t(
                  'Review usage and your shared model balance whenever you need.'
                )}
              </p>
            </div>
          </li>
        </ol>
      </section>
    </main>
  )
}
