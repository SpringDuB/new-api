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
import manifest from './catalog-cover-manifest.json'
import { getModelCategory } from './model-categories'

const assignedCovers = new Map(Object.entries(manifest))
const artwork = [...new Set(assignedCovers.values())]

export function assignCatalogCovers(models: PricingModel[]) {
  const covers = new Map<string, string>()
  const coverUses = new Map(artwork.map((url) => [url, 0]))
  const unassigned: string[] = []

  for (const model of models) {
    const category = getModelCategory(model)
    if (category !== 'video' && category !== 'image') continue

    const name = model.model_name
    const cover = assignedCovers.get(name)
    if (cover) {
      covers.set(name, cover)
      coverUses.set(cover, (coverUses.get(cover) ?? 0) + 1)
    } else {
      unassigned.push(name)
    }
  }

  // Assign new entries against the full directory so changes to filters or
  // pagination never replace their covers.
  for (const name of unassigned.sort()) {
    let cover = artwork[0]
    for (const candidate of artwork) {
      if ((coverUses.get(candidate) ?? 0) < (coverUses.get(cover) ?? 0)) {
        cover = candidate
      }
    }
    covers.set(name, cover)
    coverUses.set(cover, (coverUses.get(cover) ?? 0) + 1)
  }
  return covers
}
