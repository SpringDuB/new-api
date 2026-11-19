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
import { Link } from '@tanstack/react-router'
import { ArrowRight, ChevronDown, ChevronRight, Shapes } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import {
  getModelCategory,
  getModelCategoryLabels,
  MODEL_CATEGORIES,
  type ModelCategory,
} from '@/features/pricing/lib/model-categories'
import {
  getModelFamilies,
  getModelFamilyKey,
} from '@/features/pricing/lib/model-families'
import type { PricingModel } from '@/features/pricing/types'
import { getLobeIcon } from '@/lib/lobe-icon'

import { StolasNavigationIcon } from './stolas-navigation-icon'

const artRoot = '/stolas/art/model-library'
const categoryArt: Record<ModelCategory, string> = {
  video: 'campfire',
  image: 'ice-palace',
  chat: 'rain-market',
  other: 'desert-gate',
}
const familyCoverArt: Partial<Record<ModelCategory, readonly string[]>> = {
  video: [
    'storm-crew',
    'mountain-dusk',
    'winter-visit',
    'lava-valley',
    'sea-vortex',
    'frozen-citadel',
  ].map((name) => `${artRoot}/${name}.webp`),
  image: [
    'wisteria-courtyard',
    'paris-sketch',
    'temple-lotus',
    'blue-brushwork',
    'autumn-lake',
    'garden-window',
  ].map((name) => `/stolas/art/catalog-covers/${name}.webp`),
}
const previewSize = 6

function getFamilyIcon(models: PricingModel[], family: string) {
  for (const model of models) {
    if (getModelFamilyKey(model) !== family) continue
    for (const key of [model.icon?.trim(), model.vendor_icon?.trim()]) {
      const icon = key && getLobeIcon(key, 20)
      if (icon) return icon
    }
  }
  return null
}

export function ModelLibraryNavigation({
  title,
  active,
  className,
}: {
  title: string
  active?: boolean
  className: string
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [preview, setPreview] = useState<ModelCategory | null>(null)
  const { models, isLoading, error, refetch } = usePricingData(open)
  const categories = useMemo(() => {
    const groups: Record<ModelCategory, PricingModel[]> = {
      video: [],
      image: [],
      chat: [],
      other: [],
    }
    for (const model of models) groups[getModelCategory(model)].push(model)
    return groups
  }, [models])
  const labels = getModelCategoryLabels(t)
  const category =
    preview ??
    MODEL_CATEGORIES.find((item) => categories[item].length > 0) ??
    'video'
  const categoryModels = categories[category]
  const featured = getModelFamilies(categoryModels).slice(0, previewSize)
  const hasFeatured = featured.length > 0

  return (
    <Popover
      open={open}
      onOpenChange={(nextOpen) => {
        setOpen(nextOpen)
        if (!nextOpen) setPreview(null)
      }}
    >
      <PopoverTrigger
        openOnHover
        delay={90}
        closeDelay={150}
        aria-current={active ? 'page' : undefined}
        data-active={active || undefined}
        className={className}
        render={<Button variant='ghost' />}
      >
        <StolasNavigationIcon destination='/pricing' />
        {title}
        <ChevronDown
          aria-hidden='true'
          className='stolas-library-chevron size-3'
        />
      </PopoverTrigger>
      <PopoverContent
        align='start'
        sideOffset={5}
        collisionPadding={16}
        className='stolas-library-menu'
        aria-label={title}
      >
        <div className='stolas-library-layout'>
          <nav className='stolas-library-categories' aria-label={title}>
            {MODEL_CATEGORIES.filter(
              (item) => isLoading || error || categories[item].length > 0
            ).map((item) => (
              <Link
                key={item}
                to='/pricing'
                search={{ category: item }}
                className='stolas-library-category'
                data-preview={category === item}
                onPointerEnter={(event) => {
                  if (event.pointerType === 'mouse') setPreview(item)
                }}
                onFocus={() => setPreview(item)}
                onClick={() => setOpen(false)}
              >
                <img
                  src={`${artRoot}/${categoryArt[item]}.webp`}
                  alt=''
                  width={640}
                  height={360}
                  decoding='async'
                />
                <span className='stolas-library-category-copy'>
                  <strong>{labels[item]}</strong>
                  {categories[item].length > 0 && (
                    <span>{categories[item].length}</span>
                  )}
                </span>
                <ChevronRight aria-hidden='true' className='size-4 shrink-0' />
              </Link>
            ))}
            <Link
              to='/pricing'
              search={{}}
              className='stolas-library-all'
              onClick={() => setOpen(false)}
            >
              {t('All models')}
              <ArrowRight aria-hidden='true' className='size-4' />
            </Link>
          </nav>
          <section
            className='stolas-library-preview'
            aria-label={labels[category]}
          >
            <div className='stolas-library-preview-header'>
              <h2>{labels[category]}</h2>
              {hasFeatured && (
                <Link
                  to='/pricing'
                  search={{ category }}
                  onClick={() => setOpen(false)}
                >
                  {t('More')}
                  <ArrowRight aria-hidden='true' className='size-3.5' />
                </Link>
              )}
            </div>
            {isLoading && (
              <p className='stolas-library-state' role='status'>
                {t('Loading')}
              </p>
            )}
            {!isLoading && !hasFeatured && error && (
              <div className='stolas-library-state'>
                <Button
                  size='sm'
                  variant='outline'
                  onClick={() => void refetch()}
                >
                  {t('Retry')}
                </Button>
              </div>
            )}
            {!isLoading && !hasFeatured && !error && (
              <p className='stolas-library-state'>{t('No models found')}</p>
            )}
            {!isLoading && hasFeatured && (
              <ul className='stolas-library-cards'>
                {featured.map((family, index) => {
                  const label = family.label ?? t('Unassigned series')
                  const icon =
                    family.label && getFamilyIcon(categoryModels, family.value)
                  return (
                    <li key={family.value}>
                      <Link
                        to='/pricing'
                        search={{ category, family: family.value }}
                        className='stolas-library-card'
                        onClick={() => setOpen(false)}
                      >
                        <img
                          className='stolas-library-cover'
                          src={
                            familyCoverArt[category]?.[index] ??
                            `${artRoot}/${categoryArt[category]}.webp`
                          }
                          alt=''
                          width={640}
                          height={360}
                          decoding='async'
                        />
                        <div className='stolas-library-card-copy'>
                          <span className='stolas-library-card-title'>
                            <span
                              className='stolas-library-card-icon'
                              aria-hidden='true'
                            >
                              {icon || <Shapes className='size-4' />}
                            </span>
                            <strong title={label}>{label}</strong>
                          </span>
                          <p>
                            {t('{{count}} models', { count: family.count })}
                          </p>
                        </div>
                      </Link>
                    </li>
                  )
                })}
              </ul>
            )}
          </section>
        </div>
      </PopoverContent>
    </Popover>
  )
}
