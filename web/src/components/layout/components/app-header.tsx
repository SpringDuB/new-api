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
import { MoreHorizontal, Wallet } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { ConfigDrawer } from '@/components/config-drawer'
import { NotificationPopover } from '@/components/notification-popover'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { Search } from '@/components/search'
import { Button, buttonVariants } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { SystemUpdateAction } from '@/features/system-update/system-update-action'
import { useWalletUser } from '@/features/wallet/hooks/use-wallet-user'
import { useIsMobile } from '@/hooks/use-mobile'
import { useNotifications } from '@/hooks/use-notifications'
import { useIsSidebarModuleVisible } from '@/hooks/use-sidebar-config'
import { useTopNavLinks } from '@/hooks/use-top-nav-links'
import { INFINITE_CANVAS_URL } from '@/lib/constants'
import { formatQuota } from '@/lib/format'
import { useSystemConfigStore } from '@/stores/system-config-store'

import { defaultTopNavLinks } from '../config/top-nav.config'
import type { TopNavLink } from '../types'
import { ModelLibraryNavigation } from './model-library-navigation'
import { StolasNavigationIcon } from './stolas-navigation-icon'
import { SystemBrand } from './system-brand'

/**
 * General application Header component
 * Integrates navigation bar, search, configuration and profile functions
 *
 * @example
 * // Basic usage
 * <AppHeader />
 *
 * @example
 * // Custom navigation links
 * <AppHeader navLinks={customLinks} />
 *
 * @example
 * // Hide navigation bar and search box
 * <AppHeader showTopNav={false} showSearch={false} />
 *
 * @example
 * // Fully customize left and right content
 * <AppHeader
 *   leftContent={<CustomLeft />}
 *   rightContent={<CustomRight />}
 * />
 */
type AppHeaderProps = {
  /**
   * Custom navigation links, uses default global navigation or dynamically generated from backend if not provided
   */
  navLinks?: TopNavLink[]
  /**
   * Whether to show top navigation bar
   * @default true
   */
  showTopNav?: boolean
  /**
   * Left content, overrides TopNav if provided
   */
  leftContent?: React.ReactNode
  /**
   * Whether to show search box
   * @default true
   */
  showSearch?: boolean
  /**
   * Custom right content, overrides default right content if provided
   */
  rightContent?: React.ReactNode
  /**
   * Whether to show notification button
   * @default true
   */
  showNotifications?: boolean
  /**
   * Whether to show config drawer
   * @default true
   */
  showConfigDrawer?: boolean
  /**
   * Whether to show profile dropdown
   * @default true
   */
  showProfileDropdown?: boolean
}

export function AppHeader({
  navLinks = defaultTopNavLinks,
  showTopNav = true,
  leftContent,
  showSearch = true,
  rightContent,
  showNotifications = true,
  showConfigDrawer = true,
  showProfileDropdown = true,
}: AppHeaderProps) {
  const { t } = useTranslation()
  // Prioritize dynamically generated links from backend
  const dynamicLinks = useTopNavLinks()
  const showCanvas = useIsSidebarModuleVisible(INFINITE_CANVAS_URL)
  const links = (dynamicLinks.length > 0 ? dynamicLinks : navLinks).filter(
    (link) => link.module !== 'canvas' || showCanvas
  )
  const canvas = links.find(
    (link) => link.module === 'canvas' && !link.disabled
  )
  const library = links.find(
    (link) => link.href === '/pricing' && !link.disabled
  )
  const isMobile = useIsMobile()

  // Notifications hook
  const notifications = useNotifications()

  return (
    <header className='stolas-masthead shrink-0'>
      <div className='stolas-masthead-inner'>
        <div className='min-w-0'>
          {leftContent ?? <SystemBrand variant='inline' />}
        </div>

        {rightContent ?? (
          <div className='stolas-masthead-tools flex shrink-0 items-center gap-1'>
            {showTopNav && canvas && (
              <a
                href={canvas.href}
                target='_blank'
                rel='noopener noreferrer'
                title={isMobile ? canvas.title : undefined}
                className={buttonVariants({
                  variant: 'ghost',
                  size: isMobile ? 'icon' : 'default',
                })}
              >
                <StolasNavigationIcon destination='canvas' />
                <span className={isMobile ? 'sr-only' : undefined}>
                  {canvas.title}
                </span>
              </a>
            )}
            {showTopNav && library && (
              <ModelLibraryNavigation
                title={library.title}
                className='stolas-library-trigger'
                compact={isMobile}
                directLink
              />
            )}
            <WorkspaceBalance />
            {showTopNav && (
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <Button
                      variant='ghost'
                      size='icon'
                      aria-label={t('Explore Stolas')}
                    />
                  }
                >
                  <MoreHorizontal aria-hidden='true' />
                </DropdownMenuTrigger>
                <DropdownMenuContent align='end'>
                  <DropdownMenuGroup>
                    <DropdownMenuLabel>{t('Explore Stolas')}</DropdownMenuLabel>
                    {links.map((link) => (
                      <DropdownMenuItem
                        key={`${link.title}-${link.href}`}
                        disabled={link.disabled}
                        render={
                          link.external ? (
                            <a
                              href={link.href}
                              target='_blank'
                              rel='noopener noreferrer'
                            />
                          ) : (
                            <Link to={link.href} />
                          )
                        }
                      >
                        {link.title}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuGroup>
                  <DropdownMenuSeparator />
                  <div className='text-muted-foreground flex flex-col gap-2 px-2 py-2 text-xs'>
                    <span>Stolas · Powered by New API</span>
                    <SystemUpdateAction presentation='version' />
                  </div>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
            {showSearch && <Search compact />}
            {showProfileDropdown && (
              <ProfileDropdown
                tools={
                  <>
                    {showNotifications && (
                      <NotificationPopover
                        open={notifications.popoverOpen}
                        onOpenChange={notifications.setPopoverOpen}
                        unreadCount={notifications.unreadCount}
                        activeTab={notifications.activeTab}
                        onTabChange={notifications.setActiveTab}
                        notice={notifications.notice}
                        announcements={notifications.announcements}
                        loading={notifications.loading}
                      />
                    )}
                    {showConfigDrawer && (
                      <ConfigDrawer showSidebarSettings={false} />
                    )}
                  </>
                }
              />
            )}
          </div>
        )}
      </div>
    </header>
  )
}

function WorkspaceBalance() {
  const { t } = useTranslation()
  const { data: user } = useWalletUser(true)
  // The shared quota formatter reads the current currency configuration.
  useSystemConfigStore((state) => state.config.currency)
  if (!user) return null

  const balance = formatQuota(user.quota)
  const label = t('Current Balance')
  return (
    <div
      role='group'
      aria-label={label}
      title={`${label}: ${balance}`}
      className='border-border me-1 flex max-w-28 min-w-0 items-center gap-1.5 border-e pe-2 text-xs tabular-nums max-[400px]:max-w-20 sm:max-w-40'
    >
      <Wallet
        aria-hidden='true'
        className='text-muted-foreground size-4 shrink-0 max-[400px]:hidden'
      />
      <span className='truncate font-medium'>{balance}</span>
    </div>
  )
}
