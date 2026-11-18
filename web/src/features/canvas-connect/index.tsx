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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { Button } from '@/components/ui/button'
import { AuthLayout } from '@/features/auth/auth-layout'

export function CanvasConnectError() {
  const { t } = useTranslation()
  return (
    <AuthLayout>
      <ErrorState
        title={t('Unable to connect to your canvas')}
        description={t(
          'Start again from the canvas to renew your secure connection.'
        )}
        action={
          <Button variant='outline' render={<Link to='/' />}>
            {t('Back to home')}
          </Button>
        }
      />
    </AuthLayout>
  )
}
