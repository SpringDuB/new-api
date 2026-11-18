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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { ModelDetailsApi } from '../components/model-details-api'
import type { PricingModel } from '../types'

vi.mock('@/components/ai-elements/code-block', () => ({
  CodeBlock: (props: { code: string }) => (
    <pre data-testid='sample'>{props.code}</pre>
  ),
  CodeBlockCopyButton: () => null,
}))
const client = new QueryClient({
  defaultOptions: { queries: { retry: false } },
})
client.setQueryData(['status'], {
  server_address: 'https://stolas.localhost:8443',
})
afterEach(cleanup)
const baseModel: PricingModel = {
  id: 1,
  model_name: 'documented-model',
  quota_type: 1,
  model_ratio: 1,
  completion_ratio: 1,
  enable_groups: ['default'],
}
function renderApi(model: Partial<PricingModel>, type: string, path: string) {
  render(
    <QueryClientProvider client={client}>
      <ModelDetailsApi
        model={{ ...baseModel, ...model, supported_endpoint_types: [type] }}
        endpointMap={{ [type]: { path, method: 'POST' } }}
      />
    </QueryClientProvider>
  )
}

describe('model API documentation', () => {
  it('uses the video endpoint and documented required inputs instead of chat messages', () => {
    renderApi(
      {
        square_category: 'video',
        description:
          'prompt：提示词，必填；duration：生成时长，必填，支持 5–30 秒，默认 5 秒；images：参考图片，数组，必填，1–10 张；videos：当前模型不支持此素材',
      },
      'openai-video',
      '/v1/videos'
    )
    const code = screen.getByTestId('sample').textContent
    expect(code).toContain('/v1/videos')
    expect(code).toContain('"prompt"')
    expect(code).toContain('"duration": 5')
    expect(code).toContain('"images": [')
    expect(code).not.toContain('"messages"')
    expect(code).not.toContain('"videos"')
    expect(screen.getByText(/^GET \/v1\/videos\/\{task_id\}$/)).toBeVisible()
    expect(screen.queryByText('Rate limits')).not.toBeInTheDocument()
  })
  it.each([
    ['openai-response', '/v1/responses', '"input"', '"messages"'],
    ['jina-rerank', '/v1/rerank', '"documents"', '"input"'],
    ['embeddings', '/v1/embeddings', '"input"', '"messages"'],
    ['audio-speech', '/v1/audio/speech', '"voice"', '"messages"'],
    ['image-generation', '/v1/images/generations', '"prompt"', '"messages"'],
  ])(
    'builds the %s request using its own payload',
    (type, path, field, wrongField) => {
      renderApi({}, type, path)
      const code = screen.getByTestId('sample').textContent
      expect(code).toContain(path)
      expect(code).toContain(field)
      expect(code).not.toContain(wrongField)
    }
  )
  it('uses a documented resolution and aspect ratio rather than invalid generic size values', () => {
    renderApi(
      {
        square_category: 'video',
        description:
          'prompt：提示词，必填；resolution：输出分辨率，必填，可选 720p / 1080p，默认 720p；aspect_ratio：画面比例，必填，可选 16:9 / 9:16，默认 16:9；videos：参考视频，数组类型，必填，当前模型不支持此素材',
      },
      'openai-video',
      '/v1/videos'
    )
    const code = screen.getByTestId('sample').textContent
    expect(code).toContain('"resolution": "720p"')
    expect(code).toContain('"aspect_ratio": "16:9"')
    expect(code).not.toContain('"videos"')
    expect(code).not.toContain('1024x1024')
  })
  it('does not fabricate a chat example for an unknown endpoint or a video with stale chat metadata', () => {
    renderApi({ square_category: 'video' }, 'openai', '/v1/chat/completions')
    expect(screen.queryByTestId('sample')).not.toBeInTheDocument()
    expect(
      screen.getByText('API examples are not available for this model yet.')
    ).toBeVisible()
  })
  it('keeps the gateway URL and environment credentials in each language example', async () => {
    renderApi({}, 'gemini', '/v1beta/models/{model}:generateContent')
    const user = userEvent.setup()
    for (const language of ['Python', 'TypeScript', 'JavaScript']) {
      await user.click(screen.getByRole('tab', { name: language }))
      const code = screen.getByTestId('sample').textContent
      expect(code).toContain(
        'https://stolas.localhost:8443/v1beta/models/documented-model:generateContent'
      )
      expect(code).toContain('NEW_API_KEY')
    }
  })
})
