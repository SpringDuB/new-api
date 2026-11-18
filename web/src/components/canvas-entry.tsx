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
import { ArrowUpRight01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { getCanvasStartURL } from '@/features/canvas-connect/connection'
import { useStatus } from '@/hooks/use-status'

type CanvasEntryProps = {
  className?: string
  variant?: 'default' | 'outline' | 'ghost'
  label?: string
}

export function CanvasEntry(props: CanvasEntryProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  if (status?.canvas_enabled === false) {
    return (
      <Button
        size='lg'
        variant={props.variant}
        className={props.className}
        disabled
      >
        {t('Service unavailable')}
      </Button>
    )
  }
  return (
    <Button
      size='lg'
      role='link'
      variant={props.variant}
      className={props.className}
      render={<a href={getCanvasStartURL(status)} />}
    >
      {t(props.label ?? 'Open Infinite Canvas')}
      <HugeiconsIcon
        icon={ArrowUpRight01Icon}
        data-icon='inline-end'
        aria-hidden='true'
      />
    </Button>
  )
}
