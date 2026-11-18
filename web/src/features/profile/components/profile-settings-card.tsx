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
import { useTranslation } from 'react-i18next'

import { Skeleton } from '@/components/ui/skeleton'

import type { UserProfile } from '../types'
import { NotificationTab } from './tabs/notification-tab'

// ============================================================================
// Profile Settings Card Component
// ============================================================================

interface ProfileSettingsCardProps {
  profile: UserProfile | null
  loading: boolean
  onProfileUpdate: () => void
  quotaRemindEnabled: boolean
  quotaRemindThreshold: number
}

export function ProfileSettingsCard({
  profile,
  loading,
  onProfileUpdate,
  quotaRemindEnabled,
  quotaRemindThreshold,
}: ProfileSettingsCardProps) {
  const { t } = useTranslation()

  if (loading) {
    return (
      <section className='stolas-profile-section' aria-busy='true'>
        <header className='stolas-profile-section-heading'>
          <Skeleton className='h-6 w-32' />
          <Skeleton className='mt-2 h-4 w-48' />
        </header>
        <div className='flex flex-col gap-4'>
          {['notifications', 'threshold', 'preferences'].map((key) => (
            <Skeleton key={key} className='h-20 w-full' />
          ))}
        </div>
      </section>
    )
  }

  return (
    <section
      className='stolas-profile-section'
      aria-labelledby='profile-notifications-title'
    >
      <header className='stolas-profile-section-heading'>
        <h2 id='profile-notifications-title' className='stolas-editorial'>
          {t('Notifications')}
        </h2>
        <p>{t('Choose where and when Stolas sends updates.')}</p>
      </header>
      <NotificationTab
        profile={profile}
        onUpdate={onProfileUpdate}
        quotaRemindEnabled={quotaRemindEnabled}
        quotaRemindThreshold={quotaRemindThreshold}
      />
    </section>
  )
}
