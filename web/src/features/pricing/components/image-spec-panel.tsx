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
import { ImageIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import {
  formatTaskUsageUnitPrice,
  type DynamicPriceOptions,
} from '../lib/dynamic-price'
import { resolutionHeight } from '../lib/model-helpers'
import type { CatalogImageSpec } from '../types'
import { SpecOptions } from './spec-options'

export function ImageSpecPanel({
  specs,
  priceOptions,
}: {
  specs: CatalogImageSpec[]
  priceOptions: DynamicPriceOptions
}) {
  const { t, i18n } = useTranslation()
  const [provider, setProvider] = useState('')
  const [size, setSize] = useState('')
  const [action, setAction] = useState('')
  const providers = [...new Set(specs.map((s) => s.provider ?? ''))]
  const activeProvider = providers.includes(provider) ? provider : providers[0]
  const rows = specs.filter((s) => (s.provider ?? '') === activeProvider)
  const sizes = [
    ...new Set(rows.map((s) => s.resolution ?? s.imageSize ?? '')),
  ].sort((a, b) => resolutionHeight(a) - resolutionHeight(b))
  const activeSize = sizes.includes(size) ? size : sizes[0]
  const options = rows.filter(
    (s) => (s.resolution ?? s.imageSize ?? '') === activeSize
  )
  const spec = options.find((s) => s.action === action) ?? options[0]
  const actions = options.flatMap((s) => (s.action ? [s.action] : []))
  const labels: Record<string, string> = {
    imagine: t('Generate'),
    edits: t('Edit'),
    blend: t('Blend'),
    hd_upscale: t('Upscale'),
  }
  const count = (n: number) => formatNumber(n, toIntlLocale(i18n.language))
  const min = spec.minImages ?? 0
  const max = spec.maxImages
  let range = min ? `≥${count(min)}` : ''
  if (max !== undefined && max > 0) {
    range = `≤${count(max)}`
    if (min === max) range = count(max)
    else if (min) range = `${count(min)}–${count(max)}`
  }
  return (
    <div className='grid min-w-0 gap-2'>
      <div className='stolas-video-spec-box bg-muted/30 grid min-w-0 gap-1 rounded-xl border px-3 py-2'>
        {providers.length > 1 && (
          <SpecOptions
            label={t('Provider')}
            values={providers}
            active={activeProvider}
            onChange={setProvider}
          />
        )}
        {activeSize && (
          <SpecOptions
            label={spec.imageSize ? t('Image size') : t('Output resolution')}
            values={sizes}
            active={activeSize}
            onChange={setSize}
          />
        )}
        {spec.action && (
          <SpecOptions
            label={t('Action')}
            values={actions}
            active={spec.action}
            onChange={setAction}
            display={(v) => labels[v] ?? v}
          />
        )}
        <div
          role='group'
          aria-label={t('Pricing')}
          aria-live='polite'
          aria-atomic='true'
          className='mt-1 border-t pt-1.5 text-xs tabular-nums'
        >
          <strong className='text-sm'>
            {formatTaskUsageUnitPrice(spec.rate.amount, priceOptions)}
          </strong>{' '}
          <span className='text-muted-foreground'>
            / {spec.rate.unit === 'image' ? t('image') : t('call')}
          </span>
        </div>
      </div>
      <div
        role='group'
        aria-label={t('Reference material limits')}
        className='flex flex-wrap gap-1'
      >
        <Badge variant='outline' className='gap-1 font-normal'>
          <ImageIcon aria-hidden='true' />
          {max === 0
            ? t('No reference files')
            : [t('Reference images'), range].filter(Boolean).join(' ')}
        </Badge>
        {!!spec.maxOutputs && (
          <Badge variant='outline'>
            {t('Number of images to generate')} ≤{count(spec.maxOutputs)}
          </Badge>
        )}
        {spec.requiresSourceTask && (
          <Badge variant='outline'>{t('Completed source task required')}</Badge>
        )}
      </div>
    </div>
  )
}
