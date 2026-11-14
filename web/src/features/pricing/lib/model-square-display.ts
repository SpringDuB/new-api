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
import { parseTags } from './filters'

export type ModelSquareColor = 'red' | 'amber' | 'blue' | 'emerald' | 'violet'

export type ModelSquareDecoration = {
  badge?: string
  badge_style?: 'ribbon' | 'burst'
  color?: ModelSquareColor
  note?: string
  highlight?: boolean
  tags?: string[]
}

export type ModelSquareDisplay = {
  order: string[]
  models: Record<string, ModelSquareDecoration>
}

export const EMPTY_MODEL_SQUARE_DISPLAY: ModelSquareDisplay = {
  order: [],
  models: {},
}

export const MODEL_SQUARE_COLORS = {
  red: {
    badge:
      'border-red-300 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-300',
    ring: 'ring-2 ring-red-400/70 hover:ring-red-500',
    note: 'text-red-700 dark:text-red-300',
  },
  amber: {
    badge:
      'border-amber-300 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-300',
    ring: 'ring-2 ring-amber-400/70 hover:ring-amber-500',
    note: 'text-amber-800 dark:text-amber-300',
  },
  blue: {
    badge:
      'border-blue-300 bg-blue-50 text-blue-700 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-300',
    ring: 'ring-2 ring-blue-400/70 hover:ring-blue-500',
    note: 'text-blue-700 dark:text-blue-300',
  },
  emerald: {
    badge:
      'border-emerald-300 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-300',
    ring: 'ring-2 ring-emerald-400/70 hover:ring-emerald-500',
    note: 'text-emerald-700 dark:text-emerald-300',
  },
  violet: {
    badge:
      'border-violet-300 bg-violet-50 text-violet-700 dark:border-violet-800 dark:bg-violet-950 dark:text-violet-300',
    ring: 'ring-2 ring-violet-400/70 hover:ring-violet-500',
    note: 'text-violet-700 dark:text-violet-300',
  },
} as const

export function applyModelSquareDisplay(
  models: PricingModel[],
  display: ModelSquareDisplay = EMPTY_MODEL_SQUARE_DISPLAY
): PricingModel[] {
  const positions = new Map(display.order.map((name, index) => [name, index]))
  return models
    .map((model) => {
      const decoration = display.models[model.model_name]
      const tags = new Set([
        ...parseTags(model.tags),
        ...(decoration?.tags ?? []),
      ])
      return { ...model, tags: [...tags], square_display: decoration }
    })
    .sort((a, b) => {
      const difference =
        (positions.get(a.model_name) ?? Number.MAX_SAFE_INTEGER) -
        (positions.get(b.model_name) ?? Number.MAX_SAFE_INTEGER)
      return difference || a.model_name.localeCompare(b.model_name)
    })
}

// Replace visible slots only, preserving settings for temporarily unavailable
// models and models in groups outside the current administrator's catalog.
export function reorderModelSquareDisplay(
  display: ModelSquareDisplay,
  visibleOrder: string[]
): ModelSquareDisplay {
  const visible = new Set(visibleOrder)
  const order = [...display.order]
  const existing = new Set(order)
  order.push(...visibleOrder.filter((name) => !existing.has(name)))
  let index = 0
  return {
    ...display,
    order: order.map((name) =>
      visible.has(name) ? visibleOrder[index++] : name
    ),
  }
}
