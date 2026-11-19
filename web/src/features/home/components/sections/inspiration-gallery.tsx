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

const artworkPairs = [
  [
    { name: 'garden-portrait', alt: 'Woman on a bench in a sunlit garden' },
    { name: 'meadow-rabbit', alt: 'White rabbit among daisies' },
  ],
  [
    { name: 'window-portrait', alt: 'Woman sitting by a blue window' },
    { name: 'window-cat', alt: 'White cat on a blue windowsill' },
  ],
  [
    { name: 'wisteria-reader', alt: 'Woman reading beneath wisteria' },
    { name: 'autumn-fox', alt: 'Red fox beside an autumn stream' },
  ],
] as const

export function InspirationGallery() {
  const { t } = useTranslation()

  return (
    <section
      className='stolas-inspiration'
      aria-labelledby='stolas-inspiration-title'
    >
      <h2
        id='stolas-inspiration-title'
        className='stolas-editorial text-3xl sm:text-4xl'
      >
        {t('Inspiration gallery')}
      </h2>
      <div className='stolas-inspiration-grid'>
        {artworkPairs.map((pair) => (
          <div key={pair[0].name} className='stolas-inspiration-pair'>
            {pair.map((artwork) => (
              <img
                key={artwork.name}
                src={`/stolas/art/${artwork.name}.webp`}
                width={640}
                height={360}
                alt={t(artwork.alt)}
                loading='lazy'
                decoding='async'
              />
            ))}
          </div>
        ))}
      </div>
    </section>
  )
}
