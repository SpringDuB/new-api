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
import { FireIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'

import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

import {
  MODEL_SQUARE_COLORS,
  type ModelSquareDecoration,
} from '../lib/model-square-display'

export function ModelPromotionBadge(props: {
  decoration?: ModelSquareDecoration
  compact?: boolean
}) {
  if (!props.decoration?.badge) return null
  const burst = !props.compact && props.decoration.badge_style === 'burst'
  return (
    <Badge
      variant='outline'
      className={cn(
        'h-auto max-w-full gap-1.5 rounded-md px-2.5 py-1 text-xs font-semibold whitespace-normal [overflow-wrap:anywhere]',
        MODEL_SQUARE_COLORS[props.decoration.color ?? 'red'].badge,
        burst &&
          '-rotate-2 rounded-sm border-2 border-red-600 bg-amber-300 px-3 py-1.5 text-sm font-black text-red-800 shadow-[3px_3px_0_0_#dc2626] dark:border-red-400 dark:bg-amber-300 dark:text-red-900'
      )}
    >
      <HugeiconsIcon
        icon={FireIcon}
        className='size-3.5 shrink-0'
        aria-hidden='true'
      />
      {props.decoration.badge}
    </Badge>
  )
}
