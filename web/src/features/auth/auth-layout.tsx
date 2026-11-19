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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { LanguageSwitcher } from '@/components/language-switcher'
import { StolasLogo } from '@/components/stolas-logo'
import { ThemeSwitch } from '@/components/theme-switch'
import { Skeleton } from '@/components/ui/skeleton'
import { useSystemConfig } from '@/hooks/use-system-config'
import { DEFAULT_LOGO } from '@/lib/constants'

type AuthLayoutProps = {
  children: React.ReactNode
}

export function AuthLayout({ children }: AuthLayoutProps) {
  const { t } = useTranslation()
  const { systemName, logo, loading } = useSystemConfig()

  return (
    <div className='stolas-surface relative grid min-h-svh max-w-none lg:grid-cols-[1fr_1.1fr]'>
      <Link
        to='/'
        className='absolute top-4 left-4 z-10 flex items-center gap-2 transition-opacity hover:opacity-80 sm:top-8 sm:left-8 lg:text-[var(--stolas-light-ink)]'
      >
        {logo === DEFAULT_LOGO && !loading ? (
          <>
            <StolasLogo className='lg:hidden' />
            <StolasLogo inverse className='hidden lg:block' />
          </>
        ) : (
          <div className='relative h-8 w-8'>
            {loading ? (
              <Skeleton className='absolute inset-0 rounded-none' />
            ) : (
              <img
                src={logo}
                alt={t('Logo')}
                className='h-8 w-8 object-cover'
              />
            )}
          </div>
        )}
        {loading ? (
          <Skeleton className='h-6 w-24' />
        ) : (
          <div className='flex flex-col'>
            {systemName.toLowerCase() !== 'stolas' && (
              <span className='text-sm font-medium'>{systemName}</span>
            )}
          </div>
        )}
      </Link>
      <div className='absolute top-5 right-4 z-10 flex gap-1 sm:top-8 sm:right-8'>
        <LanguageSwitcher />
        <ThemeSwitch />
      </div>
      <aside className='stolas-auth-art hidden flex-col justify-end gap-4 px-12 pt-28 pb-14 lg:flex xl:px-20'>
        <p className='stolas-editorial max-w-lg text-5xl leading-tight'>
          {t('A place for your next possibility.')}
        </p>
        <p className='max-w-md text-base leading-8 opacity-75'>
          {t(
            'One account for the stories you create and the products you build.'
          )}
        </p>
      </aside>
      <div className='flex items-center px-4 pt-28 pb-10 sm:px-10 lg:py-28'>
        <div className='mx-auto flex w-full max-w-md flex-col justify-center gap-2'>
          {children}
        </div>
      </div>
    </div>
  )
}
