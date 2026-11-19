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
import {
  LibrariesIcon,
  Audit01Icon,
  BookOpen01Icon,
  ChartLineData01Icon,
  CodeIcon,
  CreditCardIcon,
  CubeIcon,
  DashboardSquare01Icon,
  File01Icon,
  Home01Icon,
  InformationCircleIcon,
  Key01Icon,
  Link01Icon,
  PenTool03Icon,
  PlugSocketIcon,
  SecurityCheckIcon,
  Settings02Icon,
  Task01Icon,
  TestTube01Icon,
  Ticket01Icon,
  UserCircleIcon,
  UserGroupIcon,
  Wallet01Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'

import { IconBadge } from '@/components/ui/icon-badge'

const DESTINATION_ICONS = [
  ['/dashboard/overview', DashboardSquare01Icon],
  ['/dashboard/models', ChartLineData01Icon],
  ['/dashboard', DashboardSquare01Icon],
  ['/pricing', LibrariesIcon],
  ['/keys', Key01Icon],
  ['/playground', TestTube01Icon],
  ['/usage-logs/task', Task01Icon],
  ['/usage-logs/drawing', Task01Icon],
  ['/usage-logs/audit', Audit01Icon],
  ['/usage-logs', File01Icon],
  ['/wallet', Wallet01Icon],
  ['/profile', UserCircleIcon],
  ['/security', SecurityCheckIcon],
  ['/channels', PlugSocketIcon],
  ['/models', CubeIcon],
  ['/users', UserGroupIcon],
  ['/redemption-codes', Ticket01Icon],
  ['/subscriptions', CreditCardIcon],
  ['/system-info', InformationCircleIcon],
  ['/task-plugins', CodeIcon],
  ['/system-settings', Settings02Icon],
  ['general', CodeIcon],
  ['personal', Wallet01Icon],
  ['admin', Settings02Icon],
  ['settings', Settings02Icon],
  ['canvas', PenTool03Icon],
  ['docs', BookOpen01Icon],
] as const

/** One pictogram vocabulary for public pages, workspace tabs and destination menus. */
export function StolasNavigationIcon(props: {
  destination: string
  tile?: boolean
}) {
  let destination = props.destination
  if (
    destination.includes('/api/auth/stolas/') ||
    destination.includes('canvas.')
  ) {
    destination = 'canvas'
  }
  if (destination.includes('docs.')) destination = 'docs'
  let icon =
    DESTINATION_ICONS.find(([prefix]) => destination.startsWith(prefix))?.[1] ??
    Link01Icon
  if (destination === '/') icon = Home01Icon
  const pictogram = (
    <HugeiconsIcon
      icon={icon}
      strokeWidth={1.65}
      aria-hidden='true'
      className='stolas-nav-symbol size-4'
    />
  )
  if (props.tile) {
    return (
      <IconBadge tone='primary' size='sm' className='stolas-nav-icon-tile'>
        {pictogram}
      </IconBadge>
    )
  }
  return pictogram
}
