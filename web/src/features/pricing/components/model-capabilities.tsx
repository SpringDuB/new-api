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
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { toIntlLocale } from '@/i18n/languages'
import { formatCompactNumber } from '@/lib/format'

import {
  getModelCategory,
  getModelCategoryLabels,
} from '../lib/model-categories'
import { getDocumentedParameters } from '../lib/model-documentation'
import type { PricingModel } from '../types'

const PARAMETER_LABELS: Record<string, string> = {
  prompt: 'Prompt',
  duration: 'Video duration',
  seconds: 'Video duration',
  resolution: 'Output resolution',
  aspect_ratio: 'Aspect ratio',
  size: 'Image size',
  images: 'Reference images',
  videos: 'Reference videos',
  audios: 'Reference audio',
  first_frame: 'First frame',
  last_frame: 'Last frame',
  n: 'Image count',
}
const MODALITY_LABELS: Record<string, string> = {
  text: 'Text',
  image: 'Image',
  audio: 'Audio',
  video: 'Video',
  file: 'File',
}
const CAPABILITY_LABELS: Record<string, string> = {
  function_calling: 'Function calling',
  streaming: 'Streaming',
  vision: 'Vision',
  json_mode: 'JSON mode',
  structured_output: 'Structured output',
  reasoning: 'Reasoning',
  tools: 'Tools',
  system_prompt: 'System prompt',
  web_search: 'Web search',
  code_interpreter: 'Code interpreter',
  caching: 'Prompt caching',
  embeddings: 'Embeddings',
}

export function ModelCapabilities(props: { model: PricingModel }) {
  const { t, i18n } = useTranslation()
  const parameters = getDocumentedParameters(props.model)
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const input = props.model.input_modalities?.map((modality) =>
    t(MODALITY_LABELS[modality])
  )
  const output = props.model.output_modalities?.map((modality) =>
    t(MODALITY_LABELS[modality])
  )
  const highlights = parameters.filter((parameter) =>
    ['duration', 'seconds', 'resolution', 'aspect_ratio', 'size'].includes(
      parameter.name
    )
  )
  const materials = parameters.filter(
    (parameter) =>
      ![
        'duration',
        'seconds',
        'resolution',
        'aspect_ratio',
        'size',
        'prompt',
      ].includes(parameter.name)
  )

  return (
    <section
      aria-label={t('Capabilities')}
      className='stolas-model-capabilities'
    >
      <div className='stolas-model-output'>
        <div>
          <span className='text-muted-foreground text-xs'>
            {t('Model type')}
          </span>
          <p className='mt-1 text-lg font-medium'>
            {getModelCategoryLabels(t)[getModelCategory(props.model)]}
          </p>
        </div>
        {input?.length || output?.length ? (
          <div className='flex flex-wrap items-center gap-3 text-sm'>
            <span>{input?.join(' / ')}</span>
            <ArrowRight
              aria-hidden='true'
              className='text-muted-foreground size-4'
            />
            <span>{output?.join(' / ')}</span>
          </div>
        ) : null}
        <dl className='flex flex-wrap gap-6 text-sm'>
          {props.model.context_length ? (
            <div>
              <dt className='text-muted-foreground'>{t('Context')}</dt>
              <dd>{formatCompactNumber(props.model.context_length, locale)}</dd>
            </div>
          ) : null}
          {props.model.max_output_tokens ? (
            <div>
              <dt className='text-muted-foreground'>{t('Max output')}</dt>
              <dd>
                {formatCompactNumber(props.model.max_output_tokens, locale)}
              </dd>
            </div>
          ) : null}
        </dl>
      </div>
      {props.model.capabilities?.length ? (
        <div className='flex flex-wrap gap-2'>
          {props.model.capabilities.map((capability) => (
            <Badge variant='secondary' key={capability}>
              {t(CAPABILITY_LABELS[capability] ?? capability)}
            </Badge>
          ))}
        </div>
      ) : null}
      {highlights.length > 0 && (
        <dl className='stolas-model-capability-grid'>
          {highlights.map((parameter) => (
            <div key={parameter.name}>
              <dt>{t(PARAMETER_LABELS[parameter.name] ?? parameter.name)}</dt>
              <dd>{parameter.description}</dd>
            </div>
          ))}
        </dl>
      )}
      {materials.length > 0 && (
        <section>
          <h3 className='mb-3 text-sm font-medium'>
            {t('Inputs and requirements')}
          </h3>
          <dl className='stolas-model-requirements'>
            {materials.map((parameter) => (
              <div key={parameter.name}>
                <dt>
                  {t(PARAMETER_LABELS[parameter.name] ?? parameter.name)}
                  {parameter.required && (
                    <Badge variant='outline'>{t('required')}</Badge>
                  )}
                </dt>
                <dd
                  className={
                    parameter.unsupported ? 'text-muted-foreground' : undefined
                  }
                >
                  {parameter.description}
                </dd>
              </div>
            ))}
          </dl>
        </section>
      )}
      {parameters.length === 0 && props.model.description && (
        <p className='text-muted-foreground text-sm leading-7 whitespace-pre-line'>
          {props.model.description}
        </p>
      )}
      {parameters.length === 0 &&
        !props.model.description &&
        !props.model.capabilities?.length && (
          <p className='text-muted-foreground text-sm'>
            {t(
              'Detailed capabilities have not been published for this model yet.'
            )}
          </p>
        )}
    </section>
  )
}
