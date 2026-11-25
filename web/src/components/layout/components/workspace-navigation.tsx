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
import { useLocation } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { SidebarTrigger } from '@/components/ui/sidebar'
import { useSidebarView } from '@/hooks/use-sidebar-view'

import { checkIsActive } from '../lib/url-utils'
import { AppHeader } from './app-header'
import { AppSidebar } from './app-sidebar'

export function WorkspaceNavigation({ children }: { children: ReactNode }) {
  const { t } = useTranslation()
  const href = useLocation({ select: (location) => location.href })
  const navigation = useSidebarView()
  const group = navigation.navGroups.find((group) =>
    group.items.some((item) => checkIsActive(href, item))
  )
  const current = group?.items
    .flatMap((item) => item.items ?? [item])
    .find((item) => checkIsActive(href, item))

  return (
    <>
      <AppSidebar navigation={navigation} />
      <div className='flex min-h-0 min-w-0 flex-1 flex-col'>
        <AppHeader
          showSearch={false}
          leftContent={
            <div className='flex min-w-0 items-center gap-2.5 text-sm'>
              <SidebarTrigger />
              <span
                aria-hidden='true'
                className='bg-border h-4 w-px shrink-0'
              />
              {group && current && (
                <span className='text-muted-foreground hidden shrink-0 items-center gap-2.5 lg:flex'>
                  {group.title}
                  <span aria-hidden='true'>/</span>
                </span>
              )}
              <span className='truncate font-medium'>
                {current?.title ?? t('Workspace')}
              </span>
            </div>
          }
        />
        {children}
      </div>
    </>
  )
}
