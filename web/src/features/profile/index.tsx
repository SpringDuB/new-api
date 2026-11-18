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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Main } from '@/components/layout'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useStatus } from '@/hooks/use-status'
import { useAuthStore } from '@/stores/auth-store'

import '@/styles/stolas-profile.css'

import { CheckinCalendarCard } from './components/checkin-calendar-card'
import { LanguagePreferencesCard } from './components/language-preferences-card'
import { ProfileHeader } from './components/profile-header'
import { ProfileSettingsCard } from './components/profile-settings-card'
import { SidebarModulesCard } from './components/sidebar-modules-card'
import { useProfile } from './hooks'

export function Profile() {
  const { t } = useTranslation()
  const { profile, loading, refreshProfile } = useProfile()
  const { status } = useStatus()
  const [selectedTab, setSelectedTab] = useState('notifications')
  const permissions = useAuthStore((s) => s.auth.user?.permissions)

  const checkinEnabled = status?.checkin_enabled === true
  const turnstileEnabled = !!(
    status?.turnstile_check && status?.turnstile_site_key
  )
  const turnstileSiteKey = status?.turnstile_site_key || ''
  const canConfigureSidebar = permissions?.sidebar_settings !== false
  // Older cached status payloads lack these fields; the site default is on.
  const quotaRemindEnabled = status?.quota_remind_enabled !== false
  const quotaRemindThreshold = status?.quota_remind_threshold ?? 0
  let activeTab = selectedTab
  if (
    (selectedTab === 'navigation' && !canConfigureSidebar) ||
    (selectedTab === 'checkin' && !checkinEnabled)
  ) {
    activeTab = 'notifications'
  }

  return (
    <Main>
      <div className='stolas-profile-scroll'>
        <div className='stolas-profile'>
          <header className='stolas-profile-heading'>
            <p className='stolas-eyebrow'>{t('Your Stolas account')}</p>
            <h1 className='stolas-editorial'>{t('Personal Settings')}</h1>
            <p>{t('Manage how Stolas works for you.')}</p>
          </header>

          <div className='stolas-profile-layout'>
            <ProfileHeader profile={profile} loading={loading} />

            <Tabs
              value={activeTab}
              onValueChange={setSelectedTab}
              className='stolas-profile-preferences'
            >
              <TabsList
                variant='line'
                activateOnFocus
                aria-label={t('Settings & Preferences')}
                className='stolas-profile-tabs'
              >
                <TabsTrigger value='notifications'>
                  {t('Notifications')}
                </TabsTrigger>
                <TabsTrigger value='language'>
                  {t('Language Preferences')}
                </TabsTrigger>
                {canConfigureSidebar && (
                  <TabsTrigger value='navigation'>
                    {t('Navigation preferences')}
                  </TabsTrigger>
                )}
                {checkinEnabled && (
                  <TabsTrigger value='checkin'>
                    {t('Daily Check-in')}
                  </TabsTrigger>
                )}
              </TabsList>

              <TabsContent value='notifications' keepMounted>
                <ProfileSettingsCard
                  profile={profile}
                  loading={loading}
                  onProfileUpdate={refreshProfile}
                  quotaRemindEnabled={quotaRemindEnabled}
                  quotaRemindThreshold={quotaRemindThreshold}
                />
              </TabsContent>
              <TabsContent value='language' keepMounted>
                <LanguagePreferencesCard
                  profile={profile}
                  onProfileUpdate={refreshProfile}
                />
              </TabsContent>
              {canConfigureSidebar && (
                <TabsContent value='navigation' keepMounted>
                  <SidebarModulesCard />
                </TabsContent>
              )}
              {checkinEnabled && (
                <TabsContent value='checkin' keepMounted>
                  <CheckinCalendarCard
                    checkinEnabled={checkinEnabled}
                    turnstileEnabled={turnstileEnabled}
                    turnstileSiteKey={turnstileSiteKey}
                  />
                </TabsContent>
              )}
            </Tabs>
          </div>
        </div>
      </div>
    </Main>
  )
}
