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
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { parseSidebarModulesAdmin } from '@/features/system-settings/maintenance/config'
import { useStatus } from '@/hooks/use-status'
import { api } from '@/lib/api'
import { handleServerError } from '@/lib/handle-server-error'
import { useAuthStore } from '@/stores/auth-store'

type SidebarModuleConfig = {
  enabled: boolean
  [key: string]: boolean
}

type SidebarModulesConfig = Record<string, SidebarModuleConfig>

type SectionDef = {
  key: string
  title: string
  description: string
  modules: { key: string; title: string; description: string }[]
}

export function SidebarModulesCard() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const [loading, setLoading] = useState(false)
  const [config, setConfig] = useState<SidebarModulesConfig>({})
  const currentUser = useAuthStore((s) => s.auth.user)
  const setUser = useAuthStore((s) => s.auth.setUser)

  const sectionDefs: SectionDef[] = [
    {
      key: 'chat',
      title: t('Online Use'),
      description: t('Playground, chat, and Infinite Canvas.'),
      modules: [
        {
          key: 'playground',
          title: t('Playground'),
          description: t('AI model testing environment'),
        },
        {
          key: 'chat',
          title: t('Chat'),
          description: t('Chat session management'),
        },
        {
          key: 'canvas',
          title: t('Infinite Canvas'),
          description: t('Open Infinite Canvas in a new tab.'),
        },
      ],
    },
    {
      key: 'console',
      title: t('Console Area'),
      description: t('Data management and log viewing'),
      modules: [
        {
          key: 'detail',
          title: t('Dashboard'),
          description: t('System data statistics'),
        },
        {
          key: 'token',
          title: t('Token Management'),
          description: t('API token management'),
        },
        {
          key: 'log',
          title: t('Usage Logs'),
          description: t('API usage records'),
        },
        {
          key: 'audit',
          title: t('Audit Logs'),
          description: t('Login, security and access records'),
        },
        {
          key: 'midjourney',
          title: t('Drawing Logs'),
          description: t('Drawing task records'),
        },
        {
          key: 'task',
          title: t('Task Logs'),
          description: t('System task records'),
        },
      ],
    },
    {
      key: 'personal',
      title: t('Personal Center Area'),
      description: t('User personal functions'),
      modules: [
        {
          key: 'topup',
          title: t('Wallet Management'),
          description: t('Balance and top-up management'),
        },
        {
          key: 'personal',
          title: t('Personal Settings'),
          description: t('Personal info settings'),
        },
        {
          key: 'security',
          title: t('Security & Access'),
          description: t('Manage your security settings and account access'),
        },
      ],
    },
  ]

  const adminConfig = parseSidebarModulesAdmin(
    status?.SidebarModulesAdmin as string | null | undefined
  )
  const visibleSectionDefs = status
    ? sectionDefs.flatMap((section) => {
        const adminSection = adminConfig[section.key]
        if (!adminSection?.enabled) return []

        const modules = section.modules.filter(
          (module) => adminSection[module.key] === true
        )
        return modules.length ? [{ ...section, modules }] : []
      })
    : []

  const loadConfig = useCallback(async () => {
    try {
      const res = await api.get('/api/user/self')
      if (res.data.success && res.data.data?.sidebar_modules) {
        const raw = res.data.data.sidebar_modules
        const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw
        setConfig(parsed)
      } else {
        setConfig({})
      }
    } catch {
      /* ignore */
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    loadConfig()
  }, [loadConfig])

  const toggleSection = (sectionKey: string, val: boolean) => {
    setConfig((prev) => ({
      ...prev,
      [sectionKey]: { ...prev[sectionKey], enabled: val },
    }))
  }

  const toggleModule = (
    sectionKey: string,
    moduleKey: string,
    val: boolean
  ) => {
    setConfig((prev) => ({
      ...prev,
      [sectionKey]: { ...prev[sectionKey], [moduleKey]: val },
    }))
  }

  const handleSave = async () => {
    setLoading(true)
    try {
      const serialized = JSON.stringify(config)
      const res = await api.put('/api/user/self', {
        sidebar_modules: serialized,
      })
      if (res.data.success) {
        // Sync to auth-store so useSidebarConfig re-runs and the sidebar
        // updates immediately without needing a page refresh.
        if (currentUser) {
          setUser({ ...currentUser, sidebar_modules: serialized })
        }
        toast.success(t('Saved successfully'))
      } else {
        handleServerError(res.data, t('Save failed'))
      }
    } catch (error) {
      handleServerError(error, t('Save failed, please retry'))
    } finally {
      setLoading(false)
    }
  }

  const handleReset = () => {
    const defaults: SidebarModulesConfig = {}
    for (const sec of visibleSectionDefs) {
      defaults[sec.key] = { enabled: true }
      for (const mod of sec.modules) defaults[sec.key][mod.key] = true
    }
    setConfig(defaults)
    toast.success(t('Reset to default configuration'))
  }

  if (!status) return null

  return (
    <section
      className='stolas-profile-section'
      aria-labelledby='profile-navigation-title'
    >
      <header className='stolas-profile-section-heading'>
        <h2 id='profile-navigation-title' className='stolas-editorial'>
          {t('Navigation preferences')}
        </h2>
        <p>{t('Customize sidebar display content')}</p>
      </header>
      <div className='flex flex-col gap-6'>
        {visibleSectionDefs.map((section) => {
          const sectionEnabled = config[section.key]?.enabled !== false
          return (
            <section
              key={section.key}
              className='stolas-profile-module-group'
              aria-labelledby={`profile-section-label-${section.key}`}
            >
              <div className='flex items-start justify-between gap-3'>
                <div className='min-w-0'>
                  <h3 id={`profile-section-label-${section.key}`}>
                    <Label htmlFor={`profile-section-${section.key}`}>
                      {section.title}
                    </Label>
                  </h3>
                  <p className='text-muted-foreground text-xs'>
                    {section.description}
                  </p>
                </div>
                <Switch
                  id={`profile-section-${section.key}`}
                  checked={sectionEnabled}
                  onCheckedChange={(v) => toggleSection(section.key, v)}
                />
              </div>
              <div className='stolas-profile-modules'>
                {section.modules.map((mod) => (
                  <div
                    key={mod.key}
                    className='stolas-profile-module'
                    data-disabled={!sectionEnabled || undefined}
                  >
                    <div className='mr-2 min-w-0'>
                      <Label
                        htmlFor={`profile-module-${section.key}-${mod.key}`}
                      >
                        {mod.title}
                      </Label>
                      <p className='text-muted-foreground truncate text-xs'>
                        {mod.description}
                      </p>
                    </div>
                    <Switch
                      id={`profile-module-${section.key}-${mod.key}`}
                      checked={config[section.key]?.[mod.key] !== false}
                      onCheckedChange={(v) =>
                        toggleModule(section.key, mod.key, v)
                      }
                      disabled={!sectionEnabled}
                    />
                  </div>
                ))}
              </div>
            </section>
          )
        })}

        <div className='stolas-profile-actions'>
          <Button variant='outline' onClick={handleReset}>
            {t('Reset to Default')}
          </Button>
          <Button onClick={handleSave} disabled={loading}>
            {loading ? t('Saving...') : t('Save Changes')}
          </Button>
        </div>
      </div>
    </section>
  )
}
