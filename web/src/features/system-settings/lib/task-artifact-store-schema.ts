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
import { z } from 'zod'

const storageValidationMessage = 'Invalid task artifact storage configuration'

export const storeSchema = z
  .object({
    mode: z.enum(['upstream', 'oss']),
    endpoint: z.string(),
    bucket: z.string(),
    region: z.string(),
    prefix: z.string(),
    access_key: z.string(),
    secret_key: z.string(),
    credentials_configured: z.boolean(),
    presign_ttl: z
      .number({ error: storageValidationMessage })
      .int({ error: storageValidationMessage })
      .min(1, { error: storageValidationMessage })
      .max(604800, { error: storageValidationMessage }),
    max_mb: z
      .number({ error: storageValidationMessage })
      .int({ error: storageValidationMessage })
      .min(1, { error: storageValidationMessage })
      .max(4096, { error: storageValidationMessage }),
  })
  .superRefine((value, ctx) => {
    if (value.mode !== 'oss') return
    for (const field of ['endpoint', 'bucket', 'region'] as const) {
      if (!value[field].trim()) {
        ctx.addIssue({
          code: 'custom',
          path: [field],
          message: storageValidationMessage,
        })
      }
    }
    if (!value.credentials_configured) {
      for (const field of ['access_key', 'secret_key'] as const) {
        if (!value[field]) {
          ctx.addIssue({
            code: 'custom',
            path: [field],
            message: storageValidationMessage,
          })
        }
      }
    }
  })

export type StoreForm = z.infer<typeof storeSchema>
