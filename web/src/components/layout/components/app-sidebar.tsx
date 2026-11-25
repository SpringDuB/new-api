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
import { Library } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarTrigger,
  useSidebar,
} from '@/components/ui/sidebar'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'

import type { ResolvedSidebarView } from '../types'
import { ModelLibraryNavigation } from './model-library-navigation'
import { NavGroup } from './nav-group'
import { SidebarViewHeader } from './sidebar-view-header'
import { SystemBrand } from './system-brand'

export function AppSidebar({
  navigation,
}: {
  navigation: ResolvedSidebarView
}) {
  const { t } = useTranslation()
  const { isMobile, state, setOpenMobile } = useSidebar()
  const compact = !isMobile && state === 'collapsed'
  const library = useTopNavLinks().find(
    (link) => link.href === '/pricing' && !link.disabled
  )

  return (
    <Sidebar collapsible='icon' className='top-0 h-dvh'>
      <SidebarHeader className='stolas-rail-brand flex-row items-center justify-between'>
        <SystemBrand variant='inline' compact={compact} />
        {isMobile && <SidebarTrigger />}
      </SidebarHeader>
      {navigation.view && <SidebarViewHeader view={navigation.view} />}
      <SidebarContent className='stolas-rail-scroll'>
        <nav aria-label={t('Workspace pages')} className='stolas-rail-nav'>
          {navigation.navGroups.map((group) => (
            <NavGroup key={group.id ?? group.title} {...group} />
          ))}
        </nav>
      </SidebarContent>
      {library && (
        <SidebarFooter className='stolas-rail-footer'>
          {isMobile ? (
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton
                  render={
                    <Link to='/pricing' onClick={() => setOpenMobile(false)} />
                  }
                >
                  <Library aria-hidden='true' />
                  <span>{library.title}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          ) : (
            <ModelLibraryNavigation
              title={library.title}
              side='right'
              compact={compact}
              className='stolas-rail-library'
            />
          )}
        </SidebarFooter>
      )}
    </Sidebar>
  )
}
