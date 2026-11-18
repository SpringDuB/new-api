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
import { Link, useLocation } from '@tanstack/react-router'
import { useState, type ReactElement } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useSidebarView } from '@/hooks/use-sidebar-view'

import { checkIsActive } from '../lib/url-utils'
import type { NavGroup as NavGroupData, NavItem, NavLink } from '../types'
import { AppHeader } from './app-header'
import { NavGroup } from './nav-group'

function getAreaDestination(items: NavItem[]): NavLink | undefined {
  const internal = items.find((item) => item.url && !item.external)
  if (internal?.url) return internal
  for (const item of items) {
    if (item.items?.[0]) return item.items[0]
    if (item.url) return item
  }
  return undefined
}

export function WorkspaceNavigation() {
  const { t } = useTranslation()
  const href = useLocation({ select: (location) => location.href })
  const { rootNavGroups, navGroups, view } = useSidebarView()
  const [browsedArea, setBrowsedArea] = useState<{
    href: string
    id: string
  } | null>(null)
  const overview = rootNavGroups
    .flatMap((group) => group.items)
    .find((item) => item.url === '/dashboard/overview')
  const areas: NavGroupData[] = []
  if (overview) {
    areas.push({ id: 'workspace', title: t('Console'), items: [overview] })
  }
  for (const group of rootNavGroups) {
    const items = group.items.filter(
      (item) => item.url !== '/dashboard/overview'
    )
    if (items.length === 0) continue
    let title = group.title
    if (group.id === 'personal') title = t('Account & billing')
    if (group.id === 'admin') title = t('Administration')
    areas.push({ ...group, title, items })
  }

  let currentArea = areas.find((area) =>
    area.items.some(
      (item) =>
        !('external' in item && item.external) && checkIsActive(href, item)
    )
  )
  if (view) currentArea = areas.find((area) => area.id === 'admin')
  if (href.split('?')[0].startsWith('/chat/')) {
    currentArea = areas.find((area) => area.id === 'chat')
  }
  const selectedArea =
    browsedArea?.href === href
      ? areas.find((area) => area.id === browsedArea.id)
      : currentArea
  const contextGroups = view
    ? navGroups
    : [selectedArea ?? areas[0]].filter(
        (group): group is NavGroupData => !!group
      )

  return (
    <>
      <AppHeader
        leftContent={
          <nav
            aria-label={t('Workspace areas')}
            className='stolas-area-navigation flex min-w-0 items-center gap-1 overflow-x-auto'
          >
            {areas.map((area) => {
              const destination = getAreaDestination(area.items)
              const active = selectedArea?.id === area.id
              let destinationLink: ReactElement | undefined
              if (destination?.external) {
                destinationLink = (
                  <a
                    href={destination.url}
                    target='_blank'
                    rel='noopener noreferrer'
                  />
                )
              } else if (destination) {
                destinationLink = (
                  <Link
                    to={destination.url}
                    activeProps={{ 'aria-current': 'location' }}
                  />
                )
              }
              return (
                <Button
                  key={area.id}
                  variant='ghost'
                  className='stolas-area-link shrink-0'
                  data-active={active || undefined}
                  role={destination ? 'link' : undefined}
                  aria-current={active ? 'location' : undefined}
                  aria-pressed={destination ? undefined : active}
                  onClick={
                    destination
                      ? undefined
                      : () => setBrowsedArea({ href, id: area.id ?? '' })
                  }
                  render={destinationLink}
                >
                  {area.title}
                </Button>
              )
            })}
          </nav>
        }
      />
      <div className='stolas-context-bar shrink-0'>
        <div className='stolas-context-inner'>
          <p className='stolas-context-label'>
            {view ? t('System Settings') : contextGroups[0]?.title}
          </p>
          <nav
            aria-label={t('Workspace pages')}
            className='stolas-page-navigation min-w-0 overflow-x-auto'
          >
            {contextGroups.map((group) => (
              <NavGroup
                key={group.id ?? group.title}
                {...group}
                orientation='horizontal'
              />
            ))}
          </nav>
        </div>
      </div>
    </>
  )
}
