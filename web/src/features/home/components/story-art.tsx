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

export function StoryArt() {
  const { t } = useTranslation()
  return (
    <figure className='stolas-story-art'>
      <div className='stolas-art-caption'>
        <span>STOLAS / STUDIO</span>
        <span>01 — 03</span>
      </div>
      <svg
        viewBox='0 0 640 520'
        fill='none'
        aria-hidden='true'
        className='w-full'
      >
        <defs>
          <linearGradient
            id='stolas-sky'
            x1='320'
            y1='0'
            x2='320'
            y2='520'
            gradientUnits='userSpaceOnUse'
          >
            <stop stopColor='#e9ddc4' />
            <stop offset='1' stopColor='#bc9980' />
          </linearGradient>
          <linearGradient
            id='stolas-water'
            x1='320'
            y1='284'
            x2='320'
            y2='520'
            gradientUnits='userSpaceOnUse'
          >
            <stop stopColor='#d4b28b' />
            <stop offset='1' stopColor='#696b62' />
          </linearGradient>
          <pattern
            id='stolas-grain'
            width='7'
            height='7'
            patternUnits='userSpaceOnUse'
          >
            <circle cx='1' cy='1' r='.45' fill='#342c24' fillOpacity='.13' />
          </pattern>
        </defs>
        <path fill='url(#stolas-sky)' d='M0 0h640v520H0z' />
        <circle cx='428' cy='145' r='71' fill='#cb643e' />
        <path
          d='M0 270 106 166l55 34 86-91 143 164 79-62 171 75v234H0Z'
          fill='#a49a80'
        />
        <path
          d='m0 298 115-93 118 86 61-40 103 80 78-115 165 70v234H0Z'
          fill='#72796b'
        />
        <path
          d='M0 337c104-24 163-16 246 8 95 28 189-47 394-44v219H0Z'
          fill='url(#stolas-water)'
        />
        <path
          d='m0 391 88-25 82 12 71 142H0Zm640-31-71-35-56 51-23 144h150Z'
          fill='#3f514a'
        />
        <path d='M302 520V225a62 62 0 0 1 124 0v295' fill='#e9d6b8' />
        <path d='M318 520V225a46 46 0 0 1 92 0v295' fill='#504e43' />
        <path d='M321 520V226a43 43 0 0 1 86 0v294' fill='#bc805d' />
        <path d='M321 365h86v155h-86z' fill='#e9bd83' />
        <path d='m321 520 30-155h29l27 155' fill='#f4d6a6' />
        <path
          d='M0 448c134-22 204-4 320 16M405 454c101-20 157-14 235-7M11 483c125-8 204 2 290 13M425 492c96-11 164-9 215-3'
          stroke='#e9d9b3'
          strokeOpacity='.4'
        />
        <path
          d='m76 106 10-7 10 4m31-21 11-7 11 5'
          stroke='#5e6658'
          strokeWidth='2'
        />
        <path fill='url(#stolas-grain)' d='M0 0h640v520H0z' />
      </svg>
      <figcaption className='stolas-art-footer'>
        <span>{t('A little imagination. An open horizon.')}</span>
        <span aria-hidden='true'>↗</span>
      </figcaption>
    </figure>
  )
}
