<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { api, money, moneyWhole, monthLabel, type SpendingSummary } from '../api'
import MonthlyChart from './MonthlyChart.vue'

const emit = defineEmits<{ 'open-category': [month: string, category: string] }>()

const summary = ref<SpendingSummary | null>(null)
const error = ref('')

async function load(month = '') {
  try {
    summary.value = await api.spending(month)
    error.value = ''
  } catch (e) {
    error.value = (e as Error).message
  }
}

// Bars are scaled to the largest category so the biggest fills the track.
const largest = computed(() =>
  Math.max(1, ...(summary.value?.byCategory.map((c) => c.amount) ?? [])),
)
const net = computed(() => (summary.value ? summary.value.income - summary.value.spent : 0))

onMounted(() => load())
</script>

<template>
  <p v-if="error" class="card text-red-700 dark:text-red-400" role="alert">{{ error }}</p>

  <div v-else-if="summary && !summary.month" class="card">
    <h1 class="text-lg font-semibold">No transactions yet</h1>
    <p class="muted mt-1 text-sm">
      Connect a bank or import a CSV statement from the Connections tab to see where your money
      goes.
    </p>
  </div>

  <div v-else-if="summary" class="space-y-5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-xl font-semibold">Spending</h1>
      <label class="flex items-center gap-2 text-sm">
        <span class="muted">Month</span>
        <select
          class="field"
          :value="summary.month"
          @change="load(($event.target as HTMLSelectElement).value)"
        >
          <option v-for="m in summary.months" :key="m" :value="m">{{ monthLabel(m) }}</option>
        </select>
      </label>
    </div>

    <div class="grid gap-4 sm:grid-cols-3">
      <div class="card">
        <div class="label">Spent</div>
        <div class="mt-1 text-3xl font-semibold tabular-nums">{{ money(summary.spent) }}</div>
        <div class="muted mt-1 text-xs">Excludes transfers; refunds are netted out</div>
      </div>
      <div class="card">
        <div class="label">Income</div>
        <div class="mt-1 text-3xl font-semibold tabular-nums">{{ money(summary.income) }}</div>
      </div>
      <div class="card">
        <div class="label">{{ net >= 0 ? 'Left over' : 'Overspent' }}</div>
        <div class="mt-1 text-3xl font-semibold tabular-nums">{{ money(Math.abs(net)) }}</div>
        <div class="muted mt-1 text-xs">Income minus spending</div>
      </div>
    </div>

    <section class="card">
      <h2 class="font-semibold">Where it went in {{ monthLabel(summary.month) }}</h2>
      <p class="muted text-sm">Select a category to see its transactions.</p>
      <p v-if="!summary.byCategory.length" class="muted mt-4 text-sm">No spending this month.</p>
      <ul class="mt-4 space-y-1">
        <li v-for="c in summary.byCategory" :key="c.category">
          <button
            type="button"
            class="group grid w-full grid-cols-[minmax(0,11rem)_1fr_9.5rem] items-center gap-3 rounded-md px-2 py-1.5 text-left hover:bg-stone-100 dark:hover:bg-stone-800"
            :title="`${c.label}: ${money(c.amount)} across ${c.count} transaction${c.count === 1 ? '' : 's'}`"
            @click="emit('open-category', summary.month, c.category)"
          >
            <span class="truncate text-sm">{{ c.label }}</span>
            <span class="block h-3 rounded-sm" style="background: var(--track)">
              <span
                class="block h-3 rounded-r-sm"
                :style="{
                  width: `${Math.max(0, (c.amount / largest) * 100)}%`,
                  background: 'var(--mark)',
                  minWidth: c.amount > 0 ? '2px' : '0',
                }"
              />
            </span>
            <span class="text-right text-sm tabular-nums">
              {{ money(c.amount) }}
              <span class="muted ml-1 inline-block w-10 text-xs">{{ Math.round(c.share * 100) }}%</span>
            </span>
          </button>
        </li>
      </ul>
    </section>

    <div class="grid gap-5 lg:grid-cols-2">
      <section class="card">
        <h2 class="font-semibold">Spending by month</h2>
        <p class="muted text-sm">Select a month to switch to it.</p>
        <MonthlyChart class="mt-4" :trend="summary.trend" :selected="summary.month" @select="load" />
      </section>

      <section class="card">
        <h2 class="font-semibold">Top merchants</h2>
        <table class="mt-3 w-full text-sm">
          <tbody>
            <tr
              v-for="m in summary.topMerchants"
              :key="m.name"
              class="border-t border-stone-100 first:border-0 dark:border-stone-800"
            >
              <td class="py-1.5 pr-2">{{ m.name }}</td>
              <td class="muted py-1.5 pr-2 text-right text-xs">{{ m.count }}×</td>
              <td class="py-1.5 text-right tabular-nums">{{ moneyWhole(m.amount) }}</td>
            </tr>
          </tbody>
        </table>
      </section>
    </div>
  </div>
</template>
