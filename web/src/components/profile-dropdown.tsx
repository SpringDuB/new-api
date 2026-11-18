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
import { useNavigate } from '@tanstack/react-router'
import {
  User,
  Wallet,
  LogOut,
  Settings,
  ShieldCheck,
  ChevronDown,
} from 'lucide-react'
import { useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { LanguageSwitcher } from '@/components/language-switcher'
import { SignOutDialog } from '@/components/sign-out-dialog'
import { ThemeSwitch } from '@/components/theme-switch'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import { Separator } from '@/components/ui/separator'
import useDialogState from '@/hooks/use-dialog'
import { useIsSidebarModuleVisible } from '@/hooks/use-sidebar-config'
import { useUserDisplay } from '@/hooks/use-user-display'
import { getUserAvatarFallback } from '@/lib/avatar'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

export function ProfileDropdown(props: {
  tools?: ReactNode
  showLanguageSwitcher?: boolean
  showThemeSwitch?: boolean
}) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [open, setOpen] = useDialogState()
  const [accountOpen, setAccountOpen] = useState(false)
  const user = useAuthStore((state) => state.auth.user)
  const { displayName, roleLabel } = useUserDisplay(user)
  const isSuperAdmin = user?.role === ROLE.SUPER_ADMIN
  const isWalletVisible = useIsSidebarModuleVisible('/wallet')
  const isSecurityVisible = useIsSidebarModuleVisible('/security')
  const avatarName = user?.username || displayName
  const avatarFallback = getUserAvatarFallback(avatarName)

  return (
    <>
      <Popover open={accountOpen} onOpenChange={setAccountOpen}>
        <PopoverTrigger
          render={
            <Button
              variant='ghost'
              aria-label={t('Account menu')}
              className='stolas-account-trigger h-10 gap-2 px-2'
            />
          }
        >
          <span className='stolas-account-initial' aria-hidden='true'>
            {avatarFallback}
          </span>
          <span className='hidden max-w-28 truncate text-xs sm:block'>
            {displayName}
          </span>
          <ChevronDown aria-hidden='true' />
        </PopoverTrigger>
        <PopoverContent
          align='end'
          sideOffset={12}
          className='stolas-account-panel w-[min(22rem,calc(100vw-2rem))] gap-5 rounded-sm p-5'
        >
          <div className='flex flex-col gap-1'>
            <p className='stolas-eyebrow'>{t('Your account')}</p>
            <PopoverTitle className='stolas-editorial truncate text-2xl'>
              {displayName}
            </PopoverTitle>
            <p className='text-muted-foreground text-xs'>{roleLabel}</p>
          </div>
          <Separator />
          <div className='grid grid-cols-2 gap-2'>
            <Button
              variant='ghost'
              className='h-auto justify-start py-3'
              onClick={() => {
                setAccountOpen(false)
                void navigate({ to: '/profile' })
              }}
            >
              <User aria-hidden='true' />
              {t('Profile')}
            </Button>

            {isSecurityVisible && (
              <Button
                variant='ghost'
                className='h-auto justify-start py-3'
                onClick={() => {
                  setAccountOpen(false)
                  void navigate({ to: '/security' })
                }}
              >
                <ShieldCheck aria-hidden='true' />
                {t('Security & Access')}
              </Button>
            )}

            {isWalletVisible && (
              <Button
                variant='ghost'
                className='h-auto justify-start py-3'
                onClick={() => {
                  setAccountOpen(false)
                  void navigate({ to: '/wallet' })
                }}
              >
                <Wallet aria-hidden='true' />
                {t('Wallet')}
              </Button>
            )}

            {isSuperAdmin && (
              <Button
                variant='ghost'
                className='h-auto justify-start py-3'
                onClick={() => {
                  setAccountOpen(false)
                  void navigate({
                    to: '/system-settings/site/$section',
                    params: { section: 'system-info' },
                  })
                }}
              >
                <Settings aria-hidden='true' />
                {t('System Settings')}
              </Button>
            )}
          </div>
          <Separator />
          <div className='flex items-center justify-between gap-2'>
            <span className='text-muted-foreground text-xs'>
              {t('Display & language')}
            </span>
            <div className='flex items-center gap-1'>
              {props.showLanguageSwitcher !== false && <LanguageSwitcher />}
              {props.showThemeSwitch !== false && <ThemeSwitch />}
              {props.tools}
            </div>
          </div>
          <Button
            variant='ghost'
            className='justify-start'
            onClick={() => {
              setAccountOpen(false)
              setOpen(true)
            }}
          >
            <LogOut aria-hidden='true' />
            {t('Sign out')}
          </Button>
        </PopoverContent>
      </Popover>

      <SignOutDialog open={!!open} onOpenChange={setOpen} />
    </>
  )
}
