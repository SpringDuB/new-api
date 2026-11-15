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
import type { TFunction } from 'i18next'

import type { PricingModel } from '../types'
import { parseTags } from './filters'

export const MODEL_CATEGORIES = ['video', 'image', 'chat', 'other'] as const
export type ModelCategory = (typeof MODEL_CATEGORIES)[number]
export type ModelCategoryFilter = ModelCategory | 'all'
export type ModelCategoryChoice = ModelCategory | 'auto'
export type ModelCategoryCounts = Record<ModelCategoryFilter, number>

export function getModelCategoryLabels(
  t: TFunction
): Record<ModelCategoryFilter, string> {
  return {
    video: t('Videos'),
    image: t('Images'),
    chat: t('Conversations'),
    other: t('Other'),
    all: t('All'),
  }
}

export function inferModelCategory(model: PricingModel): ModelCategory {
  if (model.square_category) return model.square_category
  const output = model.output_modalities ?? []
  if (output.includes('video')) return 'video'
  if (output.includes('image')) return 'image'
  if (output.includes('audio')) return 'other'
  if (output.includes('text')) return 'chat'

  const endpoints = model.supported_endpoint_types ?? []
  if (endpoints.includes('openai-video')) return 'video'
  if (endpoints.includes('image-generation')) return 'image'
  if (
    endpoints.some((endpoint) =>
      ['embeddings', 'jina-rerank'].includes(endpoint)
    )
  ) {
    return 'other'
  }

  // Only generation tags describe the output reliably; plain image/video tags
  // often mean reference inputs, including image input for video generation.
  const tags = parseTags(model.tags).map((tag) => tag.toLowerCase())
  if (
    tags.some((tag) =>
      [
        'video-generation',
        'text-to-video',
        'image-to-video',
        '视频生成',
        '文生视频',
        '图生视频',
      ].includes(tag)
    )
  ) {
    return 'video'
  }
  if (
    tags.some((tag) =>
      [
        'image-generation',
        'image-editing',
        'text-to-image',
        '图像生成',
        '图片生成',
        '文生图',
        '图生图',
      ].includes(tag)
    )
  ) {
    return 'image'
  }
  if (
    endpoints.some((endpoint) =>
      ['openai-response', 'anthropic', 'gemini'].includes(endpoint)
    ) ||
    tags.some((tag) =>
      [
        'chat',
        'conversation',
        'text',
        'reasoning',
        '对话',
        '聊天',
        '推理',
        '文本',
      ].includes(tag)
    )
  ) {
    return 'chat'
  }
  return 'other'
}

export function getModelCategory(model: PricingModel): ModelCategory {
  return model.square_display?.category ?? inferModelCategory(model)
}

export function countModelCategories(
  models: PricingModel[]
): ModelCategoryCounts {
  const counts: ModelCategoryCounts = {
    all: models.length,
    video: 0,
    image: 0,
    chat: 0,
    other: 0,
  }
  for (const model of models) counts[getModelCategory(model)]++
  return counts
}
