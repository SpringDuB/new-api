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
import { useTheme } from '@/context/theme-provider'
import { cn } from '@/lib/utils'

export function StolasLogo(props: {
  variant?: 'lockup' | 'mark'
  inverse?: boolean
  className?: string
}) {
  const { resolvedTheme } = useTheme()
  const inverse = props.inverse || resolvedTheme === 'dark'
  let source = inverse
    ? '/stolas/lockup-white.png'
    : '/stolas/lockup-primary.png'
  if (props.variant === 'mark') {
    source = inverse ? '/stolas/mark-white.png' : '/stolas/mark-orange.png'
  }
  return (
    <img
      src={source}
      alt='Stolas'
      width={props.variant === 'mark' ? 1024 : 1200}
      height={props.variant === 'mark' ? 1024 : 326}
      className={cn('h-8 w-auto shrink-0 object-contain', props.className)}
    />
  )
}
