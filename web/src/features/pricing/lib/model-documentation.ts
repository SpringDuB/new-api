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

export type DocumentedParameter = {
  name: string
  description: string
  required: boolean
  unsupported: boolean
}

/** Read catalog-authored parameter notes without treating billing facts as inputs. */
export function getDocumentedParameters(
  model: PricingModel
): DocumentedParameter[] {
  const description = model.description ?? ''
  const matches = [
    ...description.matchAll(/(?:^|[\n;；])\s*([a-z][a-z0-9_]*)\s*[：:]\s*/g),
  ]
  return matches
    .map((match, index) => {
      const start = (match.index ?? 0) + match[0].length
      const end = matches[index + 1]?.index ?? description.length
      const text = description
        .slice(start, end)
        .replace(/[;；\s]+$/, '')
        .trim()
      return {
        name: match[1],
        description: text,
        required:
          !/非必填|不必填|optional|not required/i.test(text) &&
          /必填|\brequired\b|\bmandatory\b/i.test(text),
        unsupported:
          /当前模型不支持|本模型不支持|^不支持此|^not supported|^unsupported|this model does not support|not supported (?:by|for) this model/i.test(
            text
          ),
      }
    })
    .filter((parameter) => parameter.description.length > 0)
}

/** Values here are documentation examples only; they never determine billing. */
export function getDocumentedRequestBody(
  model: PricingModel,
  kind: 'image' | 'video'
): Record<string, unknown> {
  const parameters = getDocumentedParameters(model)
  const body: Record<string, unknown> = { model: model.model_name }
  const prompt = parameters.find((parameter) => parameter.name === 'prompt')
  if (!parameters.length || (prompt && !prompt.unsupported)) {
    body.prompt =
      kind === 'video'
        ? 'A slow camera move through a sunlit forest.'
        : 'A quiet garden in soft morning light.'
  }
  for (const parameter of parameters) {
    if (
      !parameter.required ||
      parameter.unsupported ||
      parameter.name === 'prompt' ||
      parameter.name === 'model'
    ) {
      continue
    }
    const defaultNumber = parameter.description.match(
      /(?:默认(?:值)?(?:为)?\s*|default(?:s? to| value)?[:：= ]*)(-?\d+(?:\.\d+)?)/i
    )
    if (['duration', 'seconds', 'n'].includes(parameter.name)) {
      const firstAllowed = parameter.description.match(
        /(?:支持|between|range|one of)\s*(\d+)/i
      )
      const value = defaultNumber?.[1] ?? firstAllowed?.[1]
      body[parameter.name] = value ? Number(value) : `<${parameter.name}>`
    } else if (['images', 'videos', 'audios'].includes(parameter.name)) {
      const extensions: Record<string, string> = {
        images: 'png',
        videos: 'mp4',
        audios: 'mp3',
      }
      body[parameter.name] = [
        `https://example.com/reference.${extensions[parameter.name]}`,
      ]
    } else {
      const defaultValue = parameter.description.match(
        /(?:默认(?:值)?(?:为)?\s*|default(?:s? to| value)?[:：= ]*)([a-zA-Z0-9_:.+-]+)/i
      )?.[1]
      body[parameter.name] = defaultValue ?? `<${parameter.name}>`
    }
  }
  return body
}

/** Prefer the browser's reachable HTTPS origin over a default loopback address. */
export function resolveCatalogApiBaseUrl(
  candidate: unknown,
  currentOrigin: string
): string {
  let baseUrl = currentOrigin
  if (typeof candidate === 'string' && candidate.trim()) {
    try {
      const configured = new URL(candidate)
      const current = new URL(currentOrigin)
      const loopback =
        ['localhost', '127.0.0.1', '[::1]'].includes(configured.hostname) ||
        configured.hostname.endsWith('.localhost')
      const reachable =
        !loopback ||
        configured.origin === current.origin ||
        configured.protocol === 'https:'
      if (['http:', 'https:'].includes(configured.protocol) && reachable) {
        baseUrl = configured.href
      }
    } catch {
      // An invalid configured URL must not break documentation rendering.
    }
  }
  return baseUrl.replace(/\/(?:v1|v1beta)\/?$/, '').replace(/\/$/, '')
}
