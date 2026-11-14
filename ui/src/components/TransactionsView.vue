<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import {
  api,
  dayLabel,
  money,
  monthLabel,
  type Account,
  type Category,
  type Rule,
  type Transaction,
} from '../api'

const props = defineProps<{
  categories: Category[]
  initialMonth: string
  initialCategory: string
}>()

const month = ref(props.initialMonth)
const category = ref(props.initialCategory)
const search = ref('')
const months = ref<string[]>([])
const transactions = ref<Transaction[]>([])
const accounts = ref<Record<string, Account>>({})
const rules = ref<Rule[]>([])
const error = ref('')
// The last manual change, offered as a rule for the whole merchant.
const lastChange = ref<{ merchant: string; category: string } | null>(null)

const label = (id: string) => props.categories.find((c) => c.id === id)?.label ?? id

const visible = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return transactions.value
  return transactions.value.filter((t) => `${t.merchant} ${t.name}`.toLowerCase().includes(q))
})
const total = computed(() => visible.value.reduce((sum, t) => sum + t.amount, 0))

async function load() {
  try {
    ;[transactions.value, rules.value] = await Promise.all([
      api.transactions(month.value, category.value),
      api.rules(),
    ])
    error.value = ''
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function recategorize(t: Transaction, next: string) {
  try {
    await api.categorize(t.id, next)
    lastChange.value = { merchant: t.merchant || t.name, category: next }
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function remember() {
  if (!lastChange.value) return
  try {
    await api.addRule(lastChange.value.merchant, lastChange.value.category)
    lastChange.value = null
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function removeRule(id: string) {
  try {
    await api.deleteRule(id)
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

watch([month, category], load)

onMounted(async () => {
  try {
    const [summary, accts] = await Promise.all([api.spending(), api.accounts()])
    months.value = summary.months
    accounts.value = Object.fromEntries(accts.accounts.map((a) => [a.id, a]))
  } catch (e) {
    error.value = (e as Error).message
  }
  await load()
})
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="text-xl font-semibold">Transactions</h1>
      <div class="flex flex-wrap items-center gap-2 text-sm">
        <select v-model="month" class="field" aria-label="Month">
          <option value="">All months</option>
          <option v-for="m in months" :key="m" :value="m">{{ monthLabel(m) }}</option>
        </select>
        <select v-model="category" class="field" aria-label="Category">
          <option value="">All categories</option>
          <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.label }}</option>
        </select>
        <input v-model="search" type="search" class="field" placeholder="Search merchant" aria-label="Search merchant" />
      </div>
    </div>

    <p v-if="error" class="card text-red-700 dark:text-red-400" role="alert">{{ error }}</p>

    <div
      v-if="lastChange"
      class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-blue-200 bg-blue-50 px-4 py-3 text-sm dark:border-blue-900 dark:bg-blue-950"
      role="status"
    >
      <span>
        Filed that transaction under <strong>{{ label(lastChange.category) }}</strong
        >. File everything from <strong>{{ lastChange.merchant }}</strong> this way?
      </span>
      <span class="flex gap-2">
        <button type="button" class="btn-primary" @click="remember">Always</button>
        <button type="button" class="btn" @click="lastChange = null">Just this one</button>
      </span>
    </div>

    <section class="card overflow-x-auto p-0">
      <table class="w-full min-w-[640px] text-sm">
        <thead>
          <tr class="label border-b border-stone-200 text-left dark:border-stone-800">
            <th class="px-4 py-2.5 font-medium">Date</th>
            <th class="px-4 py-2.5 font-medium">Merchant</th>
            <th class="px-4 py-2.5 font-medium">Account</th>
            <th class="px-4 py-2.5 font-medium">Category</th>
            <th class="px-4 py-2.5 text-right font-medium">Amount</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!visible.length">
            <td colspan="5" class="muted px-4 py-6 text-center">No transactions match these filters.</td>
          </tr>
          <tr
            v-for="t in visible"
            :key="t.id"
            class="border-b border-stone-100 last:border-0 dark:border-stone-800"
          >
            <td class="muted px-4 py-2 whitespace-nowrap">{{ dayLabel(t.date) }}</td>
            <td class="px-4 py-2">
              {{ t.merchant || t.name }}
              <span v-if="t.pending" class="muted ml-1 text-xs">pending</span>
            </td>
            <td class="muted px-4 py-2">{{ accounts[t.accountId]?.name ?? 'Imported' }}</td>
            <td class="px-4 py-2">
              <select
                class="field py-1"
                :value="t.category"
                :aria-label="`Category for ${t.merchant || t.name}`"
                @change="recategorize(t, ($event.target as HTMLSelectElement).value)"
              >
                <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.label }}</option>
              </select>
              <span
                v-if="t.categorySource !== 'source'"
                class="muted ml-2 text-xs"
                :title="t.categorySource === 'rule' ? 'Set by one of your rules' : 'Set by you for this transaction'"
              >
                {{ t.categorySource === 'rule' ? 'rule' : 'edited' }}
              </span>
            </td>
            <td class="px-4 py-2 text-right whitespace-nowrap tabular-nums">
              {{ t.amount < 0 ? '+' : '' }}{{ money(Math.abs(t.amount)) }}
            </td>
          </tr>
        </tbody>
        <tfoot v-if="visible.length">
          <tr class="border-t border-stone-200 dark:border-stone-800">
            <td colspan="4" class="muted px-4 py-2.5">
              {{ visible.length }} transaction{{ visible.length === 1 ? '' : 's' }}, net
            </td>
            <td class="px-4 py-2.5 text-right font-medium tabular-nums">
              {{ total < 0 ? '+' : '' }}{{ money(Math.abs(total)) }}
            </td>
          </tr>
        </tfoot>
      </table>
    </section>

    <section class="card">
      <h2 class="font-semibold">Rules</h2>
      <p class="muted text-sm">
        A rule files every transaction whose merchant contains the text under one category. Rules
        win over the bank's category; a change you make to a single transaction wins over both.
      </p>
      <p v-if="!rules.length" class="muted mt-3 text-sm">
        No rules yet. Change a transaction's category and choose “Always” to create one.
      </p>
      <ul v-else class="mt-3 divide-y divide-stone-100 text-sm dark:divide-stone-800">
        <li v-for="r in rules" :key="r.id" class="flex items-center justify-between gap-3 py-2">
          <span>
            Merchant contains <strong>{{ r.match }}</strong> → {{ label(r.category) }}
          </span>
          <button type="button" class="btn" @click="removeRule(r.id)">Delete</button>
        </li>
      </ul>
    </section>
  </div>
</template>
