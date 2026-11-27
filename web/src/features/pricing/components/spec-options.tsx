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
import { Badge } from '@/components/ui/badge'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

export function SpecOptions({
  label,
  values,
  active,
  onChange,
  display,
}: {
  label: string
  values: string[]
  active: string
  onChange: (v: string) => void
  display?: (v: string) => string
}) {
  const text = (v: string) =>
    display?.(v) ?? (v.toLowerCase() === '4k' ? '4K' : v)
  return (
    <div className='flex flex-wrap items-center gap-x-2 gap-y-1'>
      <span className='text-muted-foreground text-xs'>{label}</span>
      {values.length === 1 ? (
        <Badge variant='secondary'>{text(active)}</Badge>
      ) : (
        <ToggleGroup
          aria-label={label}
          value={[active]}
          onValueChange={([value]) => {
            if (value) onChange(value)
          }}
          spacing={1}
          size='sm'
          className='stolas-video-resolutions'
        >
          {values.map((v) => (
            <ToggleGroupItem key={v} value={v}>
              {text(v)}
            </ToggleGroupItem>
          ))}
        </ToggleGroup>
      )}
    </div>
  )
}
