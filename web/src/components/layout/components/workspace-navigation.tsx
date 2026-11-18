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
import { ArrowUpRight, ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useSidebar } from '@/components/ui/sidebar'
import { useSidebarView } from '@/hooks/use-sidebar-view'

import { checkIsActive } from '../lib/url-utils'
import type { NavGroup as NavigationGroup, NavItem, NavLink } from '../types'
import { AppHeader } from './app-header'
import { NavGroup } from './nav-group'
import { StolasNavigationIcon } from './stolas-navigation-icon'

export function WorkspaceNavigation() {
  const { t } = useTranslation()
  const { isMobile } = useSidebar()
  const href = useLocation({ select: (location) => location.href })
  const { rootNavGroups, navGroups, view } = useSidebarView()
  const general = rootNavGroups.find((group) => group.id === 'general')
  const overview = general?.items.find(
    (item) => item.url === '/dashboard/overview'
  ) as NavLink | undefined
  const canvas = rootNavGroups
    .find((group) => group.id === 'chat')
    ?.items.find((item) => 'external' in item && item.external) as
    | NavLink
    | undefined
  const chatPresets =
    rootNavGroups
      .find((group) => group.id === 'chat')
      ?.items.filter((item) => item.type === 'chat-presets') ?? []
  const menus = rootNavGroups
    .filter((group) => group.id !== 'chat')
    .map((group) => {
      const items = group.items.filter((item) => item !== overview)
      const activeItem =
        view && group.id === 'admin'
          ? undefined
          : items.find((item) => checkIsActive(href, item))
      return { ...group, title: activeItem?.title ?? group.title, items }
    })
    .filter((group) => group.items.length > 0)

  if (isMobile) {
    const current = navGroups
      .flatMap((group) => group.items)
      .flatMap((item) => item.items ?? [item])
      .find((item) => checkIsActive(href, item))
    const mobileItems: NavItem[] = [
      ...(overview ? [overview] : []),
      { title: t('Model library'), url: '/pricing' },
      ...(canvas ? [canvas] : []),
      ...menus.map((group) => ({
        title: group.title,
        items: group.items.filter((item): item is NavLink => Boolean(item.url)),
      })),
      ...(view ? navGroups.flatMap((group) => group.items) : []),
    ]
    return (
      <AppHeader
        showSearch={false}
        leftContent={
          <nav
            aria-label={t('Workspace pages')}
            className='stolas-workspace-nav'
          >
            <WorkspaceMenu
              group={{
                id: current?.url ?? '/dashboard',
                title: current?.title ?? t('Console'),
                items: mobileItems,
              }}
            />
            <NavGroup
              title={t('Chat')}
              orientation='horizontal'
              items={chatPresets}
            />
          </nav>
        }
      />
    )
  }

  return (
    <AppHeader
      showSearch={false}
      leftContent={
        <nav
          aria-label={t('Workspace pages')}
          className='stolas-workspace-nav overflow-x-auto'
        >
          {overview && (
            <Button
              role='link'
              variant='ghost'
              className='stolas-workspace-link'
              aria-current={checkIsActive(href, overview) ? 'page' : undefined}
              render={<Link to={overview.url} />}
            >
              <StolasNavigationIcon destination='/dashboard/overview' />
              {t('Console')}
            </Button>
          )}
          <Button
            role='link'
            variant='ghost'
            className='stolas-workspace-link'
            render={<Link to='/pricing' />}
          >
            <StolasNavigationIcon destination='/pricing' />
            {t('Model library')}
          </Button>
          {canvas && (
            <Button
              role='link'
              variant='ghost'
              className='stolas-workspace-link stolas-canvas-link'
              render={
                <a
                  href={canvas.url}
                  target='_blank'
                  rel='noopener noreferrer'
                />
              }
            >
              <StolasNavigationIcon destination='canvas' />
              {canvas.title}
              <ArrowUpRight aria-hidden='true' className='size-3.5' />
            </Button>
          )}
          <span className='stolas-workspace-divider' aria-hidden='true' />
          {menus.map((group) => (
            <WorkspaceMenu key={group.id} group={group} />
          ))}
          {view && (
            <WorkspaceMenu
              group={{
                id: 'settings',
                title: t('System Settings'),
                items: navGroups.flatMap((group) => group.items),
              }}
            />
          )}
          <NavGroup
            title={t('Chat')}
            orientation='horizontal'
            items={chatPresets}
          />
        </nav>
      }
    />
  )
}

function WorkspaceMenu(props: { group: NavigationGroup }) {
  const href = useLocation({ select: (location) => location.href })
  const active = props.group.items.some((item) => checkIsActive(href, item))
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant='ghost'
            className='stolas-workspace-link max-w-full'
            data-active={active || undefined}
          />
        }
      >
        <StolasNavigationIcon destination={props.group.id ?? 'settings'} />
        <span className='min-w-0 truncate'>{props.group.title}</span>
        <ChevronDown aria-hidden='true' className='size-3' />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align='start'
        className='stolas-workspace-menu max-h-[min(36rem,75dvh)] min-w-56 overflow-y-auto'
      >
        {props.group.items.map((item: NavItem) => (
          <DropdownMenuGroup key={item.title}>
            {item.items && <DropdownMenuLabel>{item.title}</DropdownMenuLabel>}
            {(item.items ?? (item.url ? [item] : [])).map((link) => (
              <DropdownMenuItem
                key={link.url}
                aria-current={checkIsActive(href, link) ? 'page' : undefined}
                render={
                  'external' in link && link.external ? (
                    <a
                      href={link.url}
                      target='_blank'
                      rel='noopener noreferrer'
                    />
                  ) : (
                    <Link to={link.url} />
                  )
                }
              >
                <StolasNavigationIcon destination={link.url ?? '/'} tile />
                {link.title}
                {link.badge && (
                  <span className='text-muted-foreground ms-auto text-xs'>
                    {link.badge}
                  </span>
                )}
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
