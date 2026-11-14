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
import { Reorder } from 'motion/react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { AutoGroupOrderItem } from '@/components/auto-group-order-item'
import { EmptyState } from '@/components/empty-state'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

export function ModelSquareOrderEditor(props: {
  names: string[]
  selectedName: string
  onSelect: (name: string) => void
  onReorder: (names: string[]) => void
}) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const [search, setSearch] = useState('')
  const matchingNames = props.names.filter((name) =>
    name.toLowerCase().includes(search.trim().toLowerCase())
  )
  const positions = new Map(props.names.map((name, index) => [name, index + 1]))
  const move = (index: number, direction: 'up' | 'down') => {
    const next = [...matchingNames]
    const target = index + (direction === 'up' ? -1 : 1)
    if (target < 0 || target >= next.length) return
    const name = next[index]
    next[index] = next[target]
    next[target] = name
    props.onReorder(next)
  }

  return (
    <div className='min-w-0 space-y-3'>
      <Label htmlFor='square-search'>{t('Model order')}</Label>
      <Input
        id='square-search'
        placeholder={t('Search models...')}
        value={search}
        onChange={(event) => setSearch(event.target.value)}
      />
      <p className='text-muted-foreground text-xs'>
        {t(
          'Drag models or use the arrow buttons to reorder. Search preserves the positions of other models.'
        )}
      </p>
      <Reorder.Group
        axis='y'
        values={matchingNames}
        onReorder={props.onReorder}
        aria-label={t('Model order')}
        layoutScroll
        className='max-h-[48dvh] space-y-2 overflow-y-auto p-1'
      >
        {matchingNames.map((name, index) => (
          <AutoGroupOrderItem
            key={name}
            group={name}
            index={index}
            count={matchingNames.length}
            onMove={move}
            leading={
              <span className='text-muted-foreground w-6 shrink-0 text-center text-xs tabular-nums'>
                {formatNumber(positions.get(name) ?? 0, locale)}
              </span>
            }
          >
            <Button
              variant={name === props.selectedName ? 'secondary' : 'ghost'}
              size='sm'
              aria-label={t('Edit {{name}}', { name })}
              aria-pressed={name === props.selectedName}
              onClick={() => props.onSelect(name)}
            >
              {t('Edit')}
            </Button>
          </AutoGroupOrderItem>
        ))}
      </Reorder.Group>
      {matchingNames.length === 0 && props.names.length > 0 && (
        <EmptyState title={t('No models found')} />
      )}
    </div>
  )
}
