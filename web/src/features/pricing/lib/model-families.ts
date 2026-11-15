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
import type { PricingModel } from '../types'

// These keys keep custom series names separate from the built-in filter values.
export const ALL_MODEL_FAMILIES = 'all'
export const UNASSIGNED_MODEL_FAMILY = 'unassigned'

const versionedFamilyPattern =
  /(?:^|[^a-z0-9])(seedance|seedream)[-_. ]?(\d+)(?:[-_. ](\d+))?(?=$|[^a-z0-9])/
const familyPatterns: readonly [RegExp, string][] = [
  [/(?:^|[^a-z0-9])(?:minimax|h3(?:video)?)(?=$|[^a-z0-9])/, 'MiniMax'],
  [/(?:^|[^a-z0-9])wan(?=$|[^a-z0-9])/, 'Wan'],
  [/(?:^|[^a-z0-9])muse[-_ ]video(?=$|[^a-z0-9])/, 'Muse Video'],
  [/(?:^|[^a-z0-9])muse(?=$|[^a-z0-9])/, 'Muse'],
  [/(?:^|[^a-z0-9])(?:nano[-_ ]?)?banana(?=$|[^a-z0-9])/, 'Nano Banana'],
  [/(?:^|[^a-z0-9])image[-_ ]?2(?=$|[^a-z0-9])/, 'Image 2'],
  [/(?:^|[^a-z0-9])midjourney(?=$|[^a-z0-9])/, 'Midjourney'],
  [/(?:^|[^a-z0-9])gemini[-_ ]omni(?=$|[^a-z0-9])/, 'Gemini Omni'],
  [/(?:^|[^a-z0-9])gemini(?=$|[^a-z0-9])/, 'Gemini'],
  [/(?:^|[^a-z0-9])video[-_ ]upscale(?=$|[^a-z0-9])/, 'Video Upscale'],
  [/(?:^|[^a-z0-9])face[-_ ]processing(?=$|[^a-z0-9])/, 'Face Processing'],
]

export function inferModelFamily(name: string): string | undefined {
  const normalized = name.toLowerCase()
  const seed = normalized.match(versionedFamilyPattern)
  if (seed) {
    const brand = seed[1] === 'seedance' ? 'Seedance' : 'Seedream'
    return `${brand} ${seed[2]}.${seed[3] ?? '0'}`
  }
  return familyPatterns.find(([pattern]) => pattern.test(normalized))?.[1]
}

export function getModelFamily(model: PricingModel): string | undefined {
  return (
    model.square_display?.family?.trim() || inferModelFamily(model.model_name)
  )
}

export function getModelFamilyKey(model: PricingModel): string {
  const family = getModelFamily(model)
  return family ? `series:${family}` : UNASSIGNED_MODEL_FAMILY
}

export function getModelFamilies(models: PricingModel[]): {
  value: string
  label: string | undefined
  count: number
}[] {
  const families = new Map<
    string,
    { value: string; label: string | undefined; count: number }
  >()
  for (const model of models) {
    const value = getModelFamilyKey(model)
    const current = families.get(value)
    if (current) current.count++
    else families.set(value, { value, label: getModelFamily(model), count: 1 })
  }
  return [...families.values()].sort((a, b) => {
    if (a.label === undefined) return b.label === undefined ? 0 : 1
    if (b.label === undefined) return -1
    return a.label.localeCompare(b.label, undefined, { numeric: true })
  })
}
