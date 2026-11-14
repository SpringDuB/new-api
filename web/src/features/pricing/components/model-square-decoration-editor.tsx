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

import { TagInput } from '@/components/tag-input'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'

import type {
  ModelSquareColor,
  ModelSquareDecoration,
} from '../lib/model-square-display'

export function ModelSquareDecorationEditor(props: {
  value: ModelSquareDecoration
  onChange: (value: ModelSquareDecoration) => void
}) {
  const { t } = useTranslation()
  return (
    <div className='space-y-4'>
      <div className='space-y-2'>
        <Label htmlFor='square-badge'>{t('Recommendation badge')}</Label>
        <Input
          id='square-badge'
          value={props.value.badge ?? ''}
          maxLength={24}
          placeholder={t('e.g. Site recommended')}
          onChange={(event) =>
            props.onChange({ ...props.value, badge: event.target.value })
          }
        />
      </div>
      <div className='space-y-2'>
        <Label>{t('Badge style')}</Label>
        <ToggleGroup
          value={[props.value.badge_style ?? 'ribbon']}
          variant='outline'
          size='sm'
          aria-label={t('Badge style')}
          onValueChange={(values) => {
            if (values[0] === 'ribbon' || values[0] === 'burst') {
              props.onChange({ ...props.value, badge_style: values[0] })
            }
          }}
        >
          <ToggleGroupItem value='ribbon'>{t('Simple badge')}</ToggleGroupItem>
          <ToggleGroupItem value='burst'>
            {t('Recommendation sticker')}
          </ToggleGroupItem>
        </ToggleGroup>
      </div>
      <div className='space-y-2'>
        <Label>{t('Accent color')}</Label>
        <ToggleGroup
          value={[props.value.color ?? 'red']}
          variant='outline'
          size='sm'
          className='flex-wrap'
          aria-label={t('Accent color')}
          onValueChange={(values) => {
            const color = values[0] as ModelSquareColor | undefined
            if (color) props.onChange({ ...props.value, color })
          }}
        >
          <ToggleGroupItem value='red'>{t('Red')}</ToggleGroupItem>
          <ToggleGroupItem value='amber'>{t('Amber')}</ToggleGroupItem>
          <ToggleGroupItem value='blue'>{t('Blue')}</ToggleGroupItem>
          <ToggleGroupItem value='emerald'>{t('Green')}</ToggleGroupItem>
          <ToggleGroupItem value='violet'>{t('Purple')}</ToggleGroupItem>
        </ToggleGroup>
      </div>
      <div className='flex items-center justify-between gap-3 rounded-lg border p-3'>
        <Label htmlFor='square-highlight'>{t('Highlight card border')}</Label>
        <Switch
          id='square-highlight'
          checked={props.value.highlight ?? false}
          onCheckedChange={(highlight) =>
            props.onChange({ ...props.value, highlight })
          }
        />
      </div>
      <div className='space-y-2'>
        <Label htmlFor='square-note'>{t('Card notice')}</Label>
        <Input
          id='square-note'
          value={props.value.note ?? ''}
          maxLength={120}
          placeholder={t('e.g. Available all day')}
          onChange={(event) =>
            props.onChange({ ...props.value, note: event.target.value })
          }
        />
      </div>
      <div className='space-y-2'>
        <Label htmlFor='square-tags'>{t('Additional tags')}</Label>
        <TagInput
          id='square-tags'
          value={props.value.tags ?? []}
          onChange={(tags) => props.onChange({ ...props.value, tags })}
        />
        <p className='text-muted-foreground text-xs'>
          {t(
            'Additional tags appear alongside the existing model tags and support filtering.'
          )}
        </p>
      </div>
    </div>
  )
}
