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
import { ArrowUpRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { StolasLogo } from '@/components/stolas-logo'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Skeleton } from '@/components/ui/skeleton'
import { useIsSidebarModuleVisible } from '@/hooks/use-sidebar-config'
import { toIntlLocale } from '@/i18n/languages'
import { formatCompactNumber, formatQuota } from '@/lib/format'
import { getRoleLabel } from '@/lib/roles'

import { getDisplayName, getUserInitials } from '../lib'
import type { UserProfile } from '../types'

// ============================================================================
// Profile Header Component
// ============================================================================

interface ProfileHeaderProps {
  profile: UserProfile | null
  loading: boolean
}

export function ProfileHeader(props: ProfileHeaderProps) {
  const { t, i18n } = useTranslation()
  const walletVisible = useIsSidebarModuleVisible('/wallet')
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)

  if (props.loading) {
    return (
      <aside
        className='stolas-profile-identity'
        aria-label={t('Account details')}
        aria-busy='true'
      >
        <Skeleton className='size-12 rounded-md' />
        <Skeleton className='mt-5 h-7 w-40' />
        <Skeleton className='mt-3 h-4 w-28' />
        <div className='mt-8 flex flex-col gap-5'>
          {['role', 'id', 'email', 'balance'].map((key) => (
            <Skeleton key={key} className='h-5 w-full' />
          ))}
        </div>
      </aside>
    )
  }

  const profile = props.profile
  if (!profile) return null

  return (
    <aside
      className='stolas-profile-identity'
      aria-label={t('Account details')}
    >
      <div className='stolas-profile-account-mark'>
        <StolasLogo variant='mark' className='size-8' />
        <span>{t('Your Stolas account')}</span>
      </div>
      <div className='stolas-profile-person'>
        <Avatar className='size-12 rounded-md'>
          <AvatarFallback className='stolas-profile-initials'>
            {getUserInitials(profile)}
          </AvatarFallback>
        </Avatar>
        <div className='min-w-0'>
          <h2 className='stolas-editorial'>{getDisplayName(profile)}</h2>
          <p>@{profile.username}</p>
        </div>
      </div>
      <dl className='stolas-profile-facts'>
        <div>
          <dt>{t('Role')}</dt>
          <dd>{getRoleLabel(profile.role)}</dd>
        </div>
        <div>
          <dt>{t('User ID')}</dt>
          <dd className='flex items-center gap-1'>
            <span>{profile.id}</span>
            <CopyButton
              value={String(profile.id)}
              className='size-6'
              aria-label={`${t('Copy to clipboard')}: ${t('User ID')}`}
            />
          </dd>
        </div>
        <div>
          <dt>{t('Email')}</dt>
          <dd>{profile.email || t('Not set')}</dd>
        </div>
        <div>
          <dt>{t('Group')}</dt>
          <dd>{profile.group || t('Not set')}</dd>
        </div>
      </dl>
      <dl className='stolas-profile-facts stolas-profile-usage'>
        <div>
          <dt>{t('Current Balance')}</dt>
          <dd>{formatQuota(profile.quota)}</dd>
        </div>
        <div>
          <dt>{t('Total Usage')}</dt>
          <dd>{formatQuota(profile.used_quota)}</dd>
        </div>
        <div>
          <dt>{t('API Requests')}</dt>
          <dd>{formatCompactNumber(profile.request_count, locale)}</dd>
        </div>
      </dl>
      {walletVisible && (
        <Link to='/wallet' className='stolas-profile-wallet'>
          {t('Open wallet')}
          <ArrowUpRight aria-hidden='true' />
        </Link>
      )}
    </aside>
  )
}
