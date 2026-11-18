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
import { lazy, Suspense, useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { LoadingState } from '@/components/loading-state'
import { PageTransition } from '@/components/page-transition'
import { Button } from '@/components/ui/button'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import '@/styles/stolas-catalog.css'

import {
  LoadingSkeleton,
  EmptyState,
  SearchBar,
  PricingTable,
  PricingToolbar,
  ModelCardGrid,
  ModelDetailsDrawer,
} from './components'
import { EXCLUDED_GROUPS, VIEW_MODES } from './constants'
import { useFilters } from './hooks/use-filters'
import { usePricingData } from './hooks/use-pricing-data'

const ModelSquareManager = lazy(
  () => import('./components/model-square-manager')
)

export function Pricing() {
  const { t } = useTranslation()
  const canManage = useAuthStore(
    (state) => (state.auth.user?.role ?? 0) >= ROLE.ADMIN
  )
  const [managerOpen, setManagerOpen] = useState(false)
  const [selectedModelName, setSelectedModelName] = useState<string | null>(
    null
  )

  const {
    models,
    catalogModels,
    groupRatio,
    usableGroup,
    endpointMap,
    autoGroups,
    isLoading,
    priceRate,
    usdExchangeRate,
    refetch,
  } = usePricingData(true, { refetchOnMount: 'always' })

  const {
    categoryFilter,
    familyFilter,
    setFamilyFilter,
    familyModels,
    sidebarModels,
    categoryCounts,
    setCategoryFilter,
    searchInput,
    sortBy,
    groupFilter,
    tokenUnit,
    viewMode,
    showRechargePrice,
    setSearchInput,
    setSortBy,
    setGroupFilter,
    setTokenUnit,
    setViewMode,
    setShowRechargePrice,
    filteredModels,
    hasActiveFilters,
    activeFilterCount,
    clearFilters,
    clearSearch,
  } = useFilters(models || [])

  const handleModelClick = useCallback(
    (modelName: string) => {
      setSelectedModelName(modelName)
      void refetch()
    },
    [refetch]
  )

  const selectedModel = useMemo(
    () =>
      selectedModelName
        ? (models || []).find(
            (model) => model.model_name === selectedModelName
          ) || null
        : null,
    [models, selectedModelName]
  )

  const availableGroups = useMemo(
    () =>
      Object.keys(usableGroup || {}).filter(
        (g) => !EXCLUDED_GROUPS.includes(g)
      ),
    [usableGroup]
  )

  const handleClearAll = useCallback(() => {
    clearFilters()
    clearSearch()
  }, [clearFilters, clearSearch])

  const renderPricingContent = () => {
    if (filteredModels.length === 0) {
      return (
        <EmptyState
          searchQuery={searchInput}
          hasActiveFilters={hasActiveFilters}
          onClearFilters={handleClearAll}
        />
      )
    }

    if (viewMode === VIEW_MODES.CARD) {
      return (
        <ModelCardGrid
          key={`${categoryFilter}:${familyFilter}`}
          presentation='directory'
          models={filteredModels}
          onModelClick={handleModelClick}
          priceRate={priceRate}
          usdExchangeRate={usdExchangeRate}
          tokenUnit={tokenUnit}
          showRechargePrice={showRechargePrice}
          selectedGroup={groupFilter}
        />
      )
    }

    return (
      <PricingTable
        key={`${categoryFilter}:${familyFilter}`}
        models={filteredModels}
        priceRate={priceRate}
        usdExchangeRate={usdExchangeRate}
        tokenUnit={tokenUnit}
        showRechargePrice={showRechargePrice}
        selectedGroup={groupFilter}
        onModelClick={handleModelClick}
      />
    )
  }

  if (isLoading) {
    return (
      <PublicLayout showMainContainer={false}>
        <div className='stolas-catalog'>
          <LoadingSkeleton viewMode={viewMode} />
        </div>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <PageTransition className='stolas-catalog'>
        <header className='stolas-catalog-header'>
          <div className='stolas-catalog-introduction'>
            <h1 className='stolas-editorial stolas-catalog-title'>
              {t('Model library')}
            </h1>
            <p className='text-muted-foreground stolas-catalog-purpose'>
              {t(
                'Compare capabilities and pricing, then choose a model for your next project.'
              )}
            </p>
          </div>
          <div className='stolas-catalog-search'>
            {canManage && (
              <Button
                variant='outline'
                size='sm'
                onClick={() => setManagerOpen(true)}
              >
                {t('Manage display')}
              </Button>
            )}
          </div>
        </header>

        <main className='stolas-catalog-results'>
          <PricingToolbar
            categoryFilter={categoryFilter}
            categoryCounts={categoryCounts}
            onCategoryChange={setCategoryFilter}
            search={
              <SearchBar
                value={searchInput}
                onChange={setSearchInput}
                onClear={clearSearch}
                placeholder={t('Search models...')}
              />
            }
            familyFilter={familyFilter}
            onFamilyChange={setFamilyFilter}
            familyModels={familyModels}
            filteredCount={filteredModels.length}
            totalCount={models?.length}
            sortBy={sortBy}
            onSortChange={setSortBy}
            tokenUnit={tokenUnit}
            onTokenUnitChange={setTokenUnit}
            showRechargePrice={showRechargePrice}
            onRechargePriceChange={setShowRechargePrice}
            viewMode={viewMode}
            onViewModeChange={setViewMode}
            groupFilter={groupFilter}
            onGroupChange={setGroupFilter}
            groups={availableGroups}
            models={sidebarModels}
            hasActiveFilters={hasActiveFilters}
            activeFilterCount={activeFilterCount}
            onClearFilters={clearFilters}
          />

          {renderPricingContent()}
        </main>

        {selectedModel && (
          <ModelDetailsDrawer
            open={Boolean(selectedModel)}
            onOpenChange={(open) => {
              if (!open) setSelectedModelName(null)
            }}
            model={selectedModel}
            groupRatio={groupRatio || {}}
            usableGroup={usableGroup || {}}
            endpointMap={
              (endpointMap as Record<
                string,
                { path?: string; method?: string }
              >) || {}
            }
            autoGroups={autoGroups || []}
            priceRate={priceRate ?? 1}
            usdExchangeRate={usdExchangeRate ?? 1}
            tokenUnit={tokenUnit}
            showRechargePrice={showRechargePrice}
          />
        )}
        {canManage && managerOpen && (
          <Suspense fallback={<LoadingState />}>
            <ModelSquareManager
              models={catalogModels}
              initialCategory={categoryFilter}
              initialFamily={familyFilter}
              priceRate={priceRate}
              usdExchangeRate={usdExchangeRate}
              onClose={() => setManagerOpen(false)}
            />
          </Suspense>
        )}
      </PageTransition>
    </PublicLayout>
  )
}
