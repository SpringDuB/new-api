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
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'

import { ModelCapabilities } from '../components/model-capabilities'
import {
  getDocumentedParameters,
  getDocumentedRequestBody,
  resolveCatalogApiBaseUrl,
} from '../lib/model-documentation'
import type { PricingModel } from '../types'

const model: PricingModel = {
  id: 1,
  model_name: 'documented-video',
  square_category: 'video',
  quota_type: 1,
  model_ratio: 1,
  completion_ratio: 1,
  enable_groups: ['default'],
}
afterEach(cleanup)

describe('published model capabilities', () => {
  it('shows video duration, resolution and reference requirements from the selected model documentation', () => {
    const documented: PricingModel = {
      ...model,
      description:
        'prompt：提示词，必填；duration：生成时长，支持 4–15 秒，默认 4 秒；resolution：输出分辨率，可选 720p；images：参考图片，最多 9 张；videos：参考视频，数组类型，当前模型不支持此素材',
      billing_usage_schema: { video_input: { enum: ['none', 'video'] } },
    }
    render(<ModelCapabilities model={documented} />)
    expect(screen.getByText('Video duration')).toBeVisible()
    expect(screen.getByText('生成时长，支持 4–15 秒，默认 4 秒')).toBeVisible()
    expect(screen.getByText('Output resolution')).toBeVisible()
    expect(screen.getByText('Reference images')).toBeVisible()
    expect(screen.getByText('参考图片，最多 9 张')).toBeVisible()
    expect(
      getDocumentedParameters(documented).find(
        (field) => field.name === 'videos'
      )?.unsupported
    ).toBe(true)
    expect(screen.queryByText('video_input')).not.toBeInTheDocument()
    expect(screen.queryByText('Context')).not.toBeInTheDocument()
  })
  it('does not invent capabilities when metadata is absent', () => {
    render(<ModelCapabilities model={model} />)
    expect(
      screen.getByText(
        'Detailed capabilities have not been published for this model yet.'
      )
    ).toBeVisible()
    expect(screen.queryByText('Function calling')).not.toBeInTheDocument()
    expect(screen.queryByText('Max output')).not.toBeInTheDocument()
  })
  it('keeps unstructured descriptions intact and does not treat optional fields as required', () => {
    expect(
      getDocumentedParameters({
        ...model,
        description: 'A model for creative work.',
      })
    ).toEqual([])
    const documented = {
      ...model,
      description:
        'prompt: Required prompt; duration: optional, defaults to 5; images: not required; audios: optional, URLs must be HTTPS',
    }
    expect(
      getDocumentedParameters(documented)
        .filter((parameter) => parameter.required)
        .map((parameter) => parameter.name)
    ).toEqual(['prompt'])
    expect(getDocumentedRequestBody(documented, 'video')).toEqual({
      model: model.model_name,
      prompt: 'A slow camera move through a sunlit forest.',
    })
  })
})

describe('documented API base URL', () => {
  it.each([
    ['http://localhost:3000', 'https://stolas.localhost:8443'],
    ['javascript:alert(1)', 'https://stolas.localhost:8443'],
    ['not a URL', 'https://stolas.localhost:8443'],
    ['https://gateway.example.com/v1/', 'https://gateway.example.com'],
  ])('resolves %s to the reachable gateway %s', (configured, expected) => {
    expect(
      resolveCatalogApiBaseUrl(configured, 'https://stolas.localhost:8443')
    ).toBe(expected)
  })
})
