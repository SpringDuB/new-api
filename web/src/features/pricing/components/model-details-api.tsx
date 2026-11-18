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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  CodeBlock,
  CodeBlockCopyButton,
} from '@/components/ai-elements/code-block'
import { CopyButton } from '@/components/copy-button'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useStatus } from '@/hooks/use-status'

import { getModelCategory } from '../lib/model-categories'
import {
  getDocumentedRequestBody,
  resolveCatalogApiBaseUrl,
} from '../lib/model-documentation'
import { replaceModelInPath } from '../lib/model-helpers'
import type { PricingModel } from '../types'

type Language = 'curl' | 'python' | 'typescript' | 'javascript'
type SampleKind =
  | 'chat'
  | 'responses'
  | 'anthropic'
  | 'gemini'
  | 'image'
  | 'video'
  | 'embeddings'
  | 'rerank'
  | 'speech'
  | 'transcription'
type Endpoint = { type: string; path: string; method: string; kind: SampleKind }
const LANGUAGES: Record<Language, string> = {
  curl: 'cURL',
  python: 'Python',
  typescript: 'TypeScript',
  javascript: 'JavaScript',
}

function sampleKind(path: string, model: PricingModel): SampleKind | null {
  const category = getModelCategory(model)
  if (path.endsWith('/chat/completions')) {
    return ['video', 'image'].includes(category) ? null : 'chat'
  }
  if (path.endsWith('/responses')) return 'responses'
  if (path.endsWith('/messages')) return 'anthropic'
  if (path.includes(':generateContent')) return 'gemini'
  if (path.endsWith('/images/generations')) return 'image'
  if (path.endsWith('/videos')) return 'video'
  if (path.endsWith('/embeddings')) return 'embeddings'
  if (path.endsWith('/rerank')) return 'rerank'
  if (path.endsWith('/audio/speech')) return 'speech'
  if (
    path.endsWith('/audio/transcriptions') ||
    path.endsWith('/audio/translations')
  ) {
    return 'transcription'
  }
  return null
}

function requestBody(
  kind: SampleKind,
  model: PricingModel
): Record<string, unknown> {
  if (kind === 'image' || kind === 'video') {
    return getDocumentedRequestBody(model, kind)
  }
  if (kind === 'gemini') {
    return { contents: [{ parts: [{ text: 'Explain how a rainbow forms.' }] }] }
  }
  const body: Record<string, unknown> = { model: model.model_name }
  if (kind === 'chat' || kind === 'anthropic') {
    body.messages = [{ role: 'user', content: 'Explain how a rainbow forms.' }]
  }
  if (kind === 'anthropic') body.max_tokens = 1024
  if (kind === 'responses') body.input = 'Explain how a rainbow forms.'
  if (kind === 'embeddings') body.input = 'A quiet garden in the morning.'
  if (kind === 'rerank') {
    body.query = 'What makes a garden peaceful?'
    body.documents = [
      'A quiet garden in the morning.',
      'A busy street at noon.',
    ]
  }
  if (kind === 'speech') {
    body.input = 'Welcome to Stolas.'
    body.voice = 'alloy'
  }
  return body
}

function buildSample(
  language: Language,
  endpoint: Endpoint,
  baseUrl: string,
  model: PricingModel
): string {
  const url = baseUrl + endpoint.path
  const body = JSON.stringify(requestBody(endpoint.kind, model), null, 2)
  const isAudio = endpoint.kind === 'speech'
  const multipart = endpoint.kind === 'transcription'
  const anthropic = endpoint.kind === 'anthropic'
  const headerName = anthropic ? 'x-api-key' : 'Authorization'
  const prefix = anthropic ? '' : 'Bearer '
  const extraHeaders = anthropic ? { 'anthropic-version': '2023-06-01' } : {}
  if (language === 'curl') {
    const lines = [
      `curl --fail-with-body "${url}" \\`,
      `  -H "${headerName}: ${prefix}$NEW_API_KEY" \\`,
      ...Object.entries(extraHeaders).map(
        ([key, value]) => `  -H "${key}: ${value}" \\`
      ),
    ]
    if (multipart) {
      lines.push(
        `  -F 'model=${model.model_name.replaceAll("'", "'\\''")}' \\`,
        `  -F 'file=@audio.mp3'`
      )
    } else {
      lines.push(
        '  -H "Content-Type: application/json" \\',
        `  --data-raw '${body.replaceAll("'", "'\\''")}'${isAudio ? ' \\' : ''}`
      )
    }
    if (isAudio) lines.push('  --output speech.mp3')
    return lines.join('\n')
  }
  if (language === 'python') {
    const lines = [
      'import os',
      'import requests',
      '',
      `url = ${JSON.stringify(url)}`,
      `headers = {${JSON.stringify(headerName)}: ${JSON.stringify(prefix)} + os.environ["NEW_API_KEY"]${anthropic ? ', "anthropic-version": "2023-06-01"' : ''}}`,
    ]
    if (multipart) {
      lines.push(
        '',
        'with open("audio.mp3", "rb") as audio:',
        `    response = requests.post(url, headers=headers, data={"model": ${JSON.stringify(model.model_name)}}, files={"file": audio})`
      )
    } else {
      lines.push(
        `payload = ${body}`,
        '',
        'response = requests.post(url, headers=headers, json=payload)'
      )
    }
    lines.push('response.raise_for_status()')
    if (isAudio) {
      lines.push(
        'with open("speech.mp3", "wb") as audio:',
        '    audio.write(response.content)'
      )
    } else lines.push('print(response.json())')
    return lines.join('\n')
  }
  const lines = [
    '// Node.js 22+',
    ...(isAudio || multipart
      ? [
          `import { ${
            isAudio ? 'writeFile' : 'readFile'
          } } from 'node:fs/promises'`,
        ]
      : []),
    '',
  ]
  if (multipart) {
    lines.push(
      'const form = new FormData()',
      `form.set('model', ${JSON.stringify(model.model_name)})`,
      "form.set('file', new Blob([await readFile('audio.mp3')]), 'audio.mp3')",
      ''
    )
  }
  lines.push(
    `const response = await fetch(${JSON.stringify(url)}, {`,
    "  method: 'POST',",
    '  headers: {',
    `    '${headerName}': ${JSON.stringify(prefix)} + process.env.NEW_API_KEY,`
  )
  if (anthropic) lines.push("    'anthropic-version': '2023-06-01',")
  if (!multipart) lines.push("    'Content-Type': 'application/json',")
  lines.push(
    '  },',
    multipart
      ? '  body: form,'
      : `  body: JSON.stringify(${body.replaceAll('\n', '\n  ')}),`,
    '})',
    '',
    'if (!response.ok) throw new Error(await response.text())'
  )
  if (isAudio) {
    lines.push(
      "await writeFile('speech.mp3', Buffer.from(await response.arrayBuffer()))"
    )
  } else lines.push('console.log(await response.json())')
  return lines.join('\n')
}

export function ModelDetailsApi(props: {
  model: PricingModel
  endpointMap: Record<string, { path?: string; method?: string }>
}) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const candidate =
    status?.server_address ??
    status?.serverAddress ??
    status?.data?.server_address ??
    status?.data?.serverAddress
  const currentOrigin =
    typeof window === 'undefined'
      ? 'https://api.example.com'
      : window.location.origin
  const baseUrl = resolveCatalogApiBaseUrl(candidate, currentOrigin)
  const endpoints: Endpoint[] = (
    props.model.supported_endpoint_types ?? []
  ).flatMap((type) => {
    const info = props.endpointMap[type]
    if (!info?.path || (info.method && info.method.toUpperCase() !== 'POST')) {
      return []
    }
    const path = replaceModelInPath(info.path, props.model.model_name)
    const kind = sampleKind(path, props.model)
    return kind ? [{ type, path, method: 'POST', kind }] : []
  })
  const [selectedType, setSelectedType] = useState('')
  const [language, setLanguage] = useState<Language>('curl')
  const endpoint =
    endpoints.find((item) => item.type === selectedType) ?? endpoints[0]
  if (!endpoint) {
    return (
      <p className='text-muted-foreground text-sm'>
        {t('API examples are not available for this model yet.')}
      </p>
    )
  }
  const code = buildSample(language, endpoint, baseUrl, props.model)
  return (
    <section className='stolas-model-api space-y-5'>
      <div className='flex flex-wrap items-center gap-2 rounded-lg border px-3 py-2.5'>
        <span className='text-primary text-xs font-semibold'>POST</span>
        <code className='min-w-0 flex-1 overflow-x-auto text-xs'>
          {baseUrl}
          {endpoint.path}
        </code>
        <CopyButton
          value={baseUrl + endpoint.path}
          tooltip={t('Copy')}
          aria-label={t('Copy')}
        />
      </div>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        {endpoints.length > 1 ? (
          <Tabs value={endpoint.type} onValueChange={setSelectedType}>
            <TabsList>
              {endpoints.map((item) => (
                <TabsTrigger value={item.type} key={item.type}>
                  {item.path}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        ) : (
          <h3 className='text-sm font-medium'>{t('Code samples')}</h3>
        )}
        <Tabs
          value={language}
          onValueChange={(value) => setLanguage(value as Language)}
        >
          <TabsList>
            {(Object.keys(LANGUAGES) as Language[]).map((value) => (
              <TabsTrigger value={value} key={value}>
                {LANGUAGES[value]}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>
      <CodeBlock code={code} language={language === 'curl' ? 'bash' : language}>
        <CodeBlockCopyButton />
      </CodeBlock>
      <p className='text-muted-foreground text-xs leading-6'>
        {t(
          'Set NEW_API_KEY to your API key. Replace example media URLs and file paths with your own before sending the request.'
        )}
      </p>
      {endpoint.kind === 'video' && (
        <section className='stolas-model-video-workflow'>
          <h3 className='mb-3 text-sm font-medium'>
            {t('Retrieve your video')}
          </h3>
          <ol className='list-inside list-decimal space-y-3 text-sm'>
            <li>{t('Save the id returned by the creation request.')}</li>
            <li>
              <code>
                GET {endpoint.path}/{'{task_id}'}
              </code>
              <p className='text-muted-foreground mt-1 text-xs'>
                {t(
                  'Query the task until its status is completed. Stop if it fails.'
                )}
              </p>
            </li>
            <li>
              <code>
                GET {endpoint.path}/{'{task_id}'}/content
              </code>
              <p className='text-muted-foreground mt-1 text-xs'>
                {t('Download the completed video using the same API key.')}
              </p>
            </li>
          </ol>
        </section>
      )}
    </section>
  )
}
