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
      <img
        src='/stolas/art/collection/home/mountains-1200.webp'
        srcSet='/stolas/art/collection/home/mountains-720.webp 720w, /stolas/art/collection/home/mountains-1200.webp 1200w'
        sizes='(min-width: 1280px) 520px, (min-width: 1024px) 44vw, calc(100vw - 3rem)'
        width={1200}
        height={675}
        alt={t('Watercolor mountains in mist')}
        fetchPriority='high'
        className='h-auto w-full'
      />
    </figure>
  )
}
