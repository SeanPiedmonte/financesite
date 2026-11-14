<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Category, type Status } from './api'
import SpendingView from './components/SpendingView.vue'
import TransactionsView from './components/TransactionsView.vue'
import AccountsView from './components/AccountsView.vue'
import ConnectionsView from './components/ConnectionsView.vue'

type Tab = 'spending' | 'transactions' | 'accounts' | 'connections'
const tabs: { id: Tab; label: string }[] = [
  { id: 'spending', label: 'Spending' },
  { id: 'transactions', label: 'Transactions' },
  { id: 'accounts', label: 'Net worth' },
  { id: 'connections', label: 'Connections' },
]

const fromHash = (): Tab => {
  const id = window.location.hash.slice(1) as Tab
  return tabs.some((t) => t.id === id) ? id : 'spending'
}

const tab = ref<Tab>(fromHash())
const status = ref<Status | null>(null)
const categories = ref<Category[]>([])
const loadError = ref('')
// Set when a category bar is clicked, so Transactions opens pre-filtered.
const filter = ref({ month: '', category: '' })

function go(id: Tab, keepFilter = false) {
  // The drill-down filter applies once; ordinary navigation starts unfiltered.
  if (!keepFilter) filter.value = { month: '', category: '' }
  tab.value = id
  window.location.hash = id
}

function openCategory(month: string, category: string) {
  filter.value = { month, category }
  go('transactions', true)
}

onMounted(async () => {
  window.addEventListener('hashchange', () => (tab.value = fromHash()))
  try {
    ;[status.value, categories.value] = await Promise.all([api.status(), api.categories()])
  } catch (e) {
    loadError.value = `Could not reach the server: ${(e as Error).message}. Is the Go server running on :8080?`
  }
})
</script>

<template>
  <header class="border-b border-stone-200 bg-white dark:border-stone-800 dark:bg-stone-900">
    <div class="mx-auto flex max-w-6xl flex-wrap items-center gap-x-8 gap-y-2 px-4 py-3">
      <span class="text-lg font-semibold">financesite</span>
      <nav class="flex gap-1" aria-label="Sections">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          class="rounded-md px-3 py-1.5 text-sm font-medium"
          :class="
            tab === t.id
              ? 'bg-stone-900 text-white dark:bg-stone-100 dark:text-stone-900'
              : 'text-stone-600 hover:bg-stone-100 dark:text-stone-300 dark:hover:bg-stone-800'
          "
          :aria-current="tab === t.id ? 'page' : undefined"
          @click="go(t.id)"
        >
          {{ t.label }}
        </button>
      </nav>
    </div>
  </header>

  <div
    v-if="status?.demo"
    class="border-b border-amber-300 bg-amber-50 text-amber-900 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-200"
  >
    <p class="mx-auto max-w-6xl px-4 py-2 text-sm">
      <strong>Demo data.</strong> These are made-up accounts and transactions, and changes are lost
      when the server restarts. Add Plaid keys to connect real accounts — see
      <button type="button" class="underline" @click="go('connections')">Connections</button>.
    </p>
  </div>

  <main class="mx-auto max-w-6xl px-4 py-6">
    <p v-if="loadError" class="card text-red-700 dark:text-red-400" role="alert">{{ loadError }}</p>
    <template v-else-if="status">
      <SpendingView v-if="tab === 'spending'" @open-category="openCategory" />
      <TransactionsView
        v-else-if="tab === 'transactions'"
        :categories="categories"
        :initial-month="filter.month"
        :initial-category="filter.category"
      />
      <AccountsView v-else-if="tab === 'accounts'" />
      <ConnectionsView v-else :status="status" />
    </template>
  </main>
</template>
