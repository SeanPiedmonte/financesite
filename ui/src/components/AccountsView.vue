<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { api, money, moneyWhole, type Account, type AccountsResponse, type Group } from '../api'

const data = ref<AccountsResponse | null>(null)
const error = ref('')

const sections: { group: Group; title: string; owed?: boolean }[] = [
  { group: 'cash', title: 'Cash' },
  { group: 'investment', title: 'Investments' },
  { group: 'retirement', title: 'Retirement — 401(k) and IRA' },
  { group: 'tsp', title: 'Thrift Savings Plan' },
  { group: 'credit', title: 'Credit cards', owed: true },
  { group: 'mortgage', title: 'Mortgages', owed: true },
]

const byGroup = computed(() => {
  const out: Partial<Record<Group, Account[]>> = {}
  for (const a of data.value?.accounts ?? []) (out[a.group] ??= []).push(a)
  return out
})
const mortgages = computed(() => byGroup.value.mortgage ?? [])

async function load() {
  try {
    data.value = await api.accounts()
    error.value = ''
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function run(action: () => Promise<unknown>) {
  try {
    await action()
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

// ---- TSP: entered by fund, because TSP is not reliably available through Plaid ----
const tspFunds = ['G Fund', 'F Fund', 'C Fund', 'S Fund', 'I Fund', 'L Funds'] as const
const tsp = reactive({ open: false, id: '', values: {} as Record<string, number | null> })

function editTsp(account?: Account) {
  tsp.id = account?.id ?? ''
  tsp.values = Object.fromEntries(
    tspFunds.map((f) => [f, account?.holdings?.find((h) => h.name === f)?.value ?? null]),
  )
  tsp.open = true
}

const tspTotal = computed(() => tspFunds.reduce((sum, f) => sum + (Number(tsp.values[f]) || 0), 0))

function saveTsp() {
  run(async () => {
    await api.saveManualAccount(
      {
        name: 'TSP',
        institution: 'Thrift Savings Plan',
        group: 'tsp',
        subtype: 'thrift savings plan',
        balance: tspTotal.value,
        holdings: tspFunds.map((f) => ({ name: f, value: Number(tsp.values[f]) || 0 })),
      },
      tsp.id || undefined,
    )
    tsp.open = false
  })
}

// ---- Property: value is entered by hand, balance comes from the linked mortgage ----
const home = reactive({
  open: false,
  id: '',
  name: '',
  address: '',
  value: null as number | null,
  mortgageAccountId: '',
})

function editHome(p?: AccountsResponse['summary']['properties'][number]) {
  Object.assign(home, {
    open: true,
    id: p?.id ?? '',
    name: p?.name ?? 'Primary residence',
    address: p?.address ?? '',
    value: p?.value ?? null,
    mortgageAccountId: p?.mortgageAccountId ?? mortgages.value[0]?.id ?? '',
  })
}

function saveHome() {
  run(async () => {
    await api.saveProperty(
      {
        name: home.name,
        address: home.address,
        value: Number(home.value) || 0,
        mortgageAccountId: home.mortgageAccountId || undefined,
      },
      home.id || undefined,
    )
    home.open = false
  })
}

const mortgageFor = (id?: string) => mortgages.value.find((m) => m.id === id)

onMounted(load)
</script>

<template>
  <p v-if="error" class="card mb-5 text-red-700 dark:text-red-400" role="alert">{{ error }}</p>

  <div v-if="data" class="space-y-5">
    <h1 class="text-xl font-semibold">Net worth</h1>

    <div class="grid gap-4 sm:grid-cols-3">
      <div class="card">
        <div class="label">Net worth</div>
        <div class="mt-1 text-3xl font-semibold tabular-nums">{{ moneyWhole(data.summary.netWorth) }}</div>
      </div>
      <div class="card">
        <div class="label">Assets</div>
        <div class="mt-1 text-3xl font-semibold tabular-nums">{{ moneyWhole(data.summary.assets) }}</div>
        <div class="muted mt-1 text-xs">Accounts plus home values</div>
      </div>
      <div class="card">
        <div class="label">Owed</div>
        <div class="mt-1 text-3xl font-semibold tabular-nums">{{ moneyWhole(data.summary.liabilities) }}</div>
        <div class="muted mt-1 text-xs">Credit cards and mortgages</div>
      </div>
    </div>

    <!-- Home -->
    <section class="card">
      <div class="flex items-center justify-between gap-3">
        <h2 class="font-semibold">Home</h2>
        <button type="button" class="btn" @click="editHome()">Add a property</button>
      </div>
      <p v-if="!data.summary.properties.length && !home.open" class="muted mt-2 text-sm">
        Add a property and its estimated value to track home equity. Link it to a mortgage account
        to subtract the loan balance.
      </p>

      <div
        v-for="p in data.summary.properties"
        :key="p.id"
        class="mt-4 border-t border-stone-100 pt-4 first:mt-3 dark:border-stone-800"
      >
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div>
            <div class="font-medium">{{ p.name }}</div>
            <div v-if="p.address" class="muted text-sm">{{ p.address }}</div>
          </div>
          <div class="flex gap-2">
            <button type="button" class="btn" @click="editHome(p)">Update value</button>
            <button type="button" class="btn" @click="run(() => api.deleteProperty(p.id))">Remove</button>
          </div>
        </div>
        <dl class="mt-3 grid gap-3 text-sm sm:grid-cols-3">
          <div>
            <dt class="label">Estimated value</dt>
            <dd class="text-lg font-semibold tabular-nums">{{ moneyWhole(p.value) }}</dd>
            <dd class="muted text-xs">Entered by you{{ p.valueAsOf ? ` on ${p.valueAsOf}` : '' }}</dd>
          </div>
          <div>
            <dt class="label">Mortgage balance</dt>
            <dd class="text-lg font-semibold tabular-nums">{{ moneyWhole(p.mortgageBalance) }}</dd>
            <dd v-if="mortgageFor(p.mortgageAccountId)?.mortgage" class="muted text-xs">
              {{ mortgageFor(p.mortgageAccountId)!.mortgage!.interestRate }}% ·
              {{ money(mortgageFor(p.mortgageAccountId)!.mortgage!.nextPayment ?? 0) }} due
              {{ mortgageFor(p.mortgageAccountId)!.mortgage!.nextPaymentDue }}
            </dd>
            <dd v-else-if="!p.mortgageAccountId" class="muted text-xs">No mortgage linked</dd>
          </div>
          <div>
            <dt class="label">Equity</dt>
            <dd class="text-lg font-semibold tabular-nums">{{ moneyWhole(p.equity) }}</dd>
            <dd class="muted text-xs">Value minus balance</dd>
          </div>
        </dl>
      </div>

      <form v-if="home.open" class="mt-4 grid gap-3 border-t border-stone-100 pt-4 sm:grid-cols-2 dark:border-stone-800" @submit.prevent="saveHome">
        <label class="text-sm">
          <span class="label">Name</span>
          <input v-model="home.name" class="field mt-1 w-full" required />
        </label>
        <label class="text-sm">
          <span class="label">Address (optional)</span>
          <input v-model="home.address" class="field mt-1 w-full" />
        </label>
        <label class="text-sm">
          <span class="label">Estimated value ($)</span>
          <input v-model.number="home.value" type="number" min="0" step="1000" class="field mt-1 w-full" required />
        </label>
        <label class="text-sm">
          <span class="label">Mortgage</span>
          <select v-model="home.mortgageAccountId" class="field mt-1 w-full">
            <option value="">None / paid off</option>
            <option v-for="m in mortgages" :key="m.id" :value="m.id">
              {{ m.institution }} — {{ m.name }} ({{ moneyWhole(m.balance) }})
            </option>
          </select>
        </label>
        <div class="flex gap-2 sm:col-span-2">
          <button type="submit" class="btn-primary">Save property</button>
          <button type="button" class="btn" @click="home.open = false">Cancel</button>
        </div>
      </form>
    </section>

    <!-- Account groups -->
    <template v-for="s in sections" :key="s.group">
      <section v-if="byGroup[s.group]?.length || s.group === 'tsp'" class="card">
        <div class="flex items-center justify-between gap-3">
          <h2 class="font-semibold">{{ s.title }}</h2>
          <div class="flex items-center gap-3">
            <button
              v-if="s.group === 'tsp' && !byGroup.tsp?.length"
              type="button"
              class="btn"
              @click="editTsp()"
            >
              Add TSP balance
            </button>
            <span class="font-semibold tabular-nums">
              {{ moneyWhole(data.summary.groups[s.group] ?? 0) }}<span v-if="s.owed" class="muted text-xs font-normal"> owed</span>
            </span>
          </div>
        </div>

        <p v-if="s.group === 'tsp' && !byGroup.tsp?.length && !tsp.open" class="muted mt-2 text-sm">
          Enter your balance by fund from your TSP statement.
        </p>

        <ul class="mt-2 divide-y divide-stone-100 text-sm dark:divide-stone-800">
          <li v-for="a in byGroup[s.group]" :key="a.id" class="py-2.5">
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <span class="font-medium">{{ a.name }}</span>
                <span class="muted">
                  · {{ a.institution }}<template v-if="a.mask"> ••{{ a.mask }}</template>
                </span>
                <span v-if="a.manual" class="muted ml-1 text-xs">(entered by you)</span>
              </div>
              <div class="flex items-center gap-3">
                <template v-if="a.manual">
                  <button v-if="a.group === 'tsp'" type="button" class="btn" @click="editTsp(a)">Update</button>
                  <button type="button" class="btn" @click="run(() => api.deleteManualAccount(a.id))">Remove</button>
                </template>
                <span class="tabular-nums">{{ money(a.balance) }}</span>
              </div>
            </div>
            <details v-if="a.holdings?.length" class="mt-1">
              <summary class="muted cursor-pointer text-xs">{{ a.holdings.length }} holding{{ a.holdings.length === 1 ? '' : 's' }}</summary>
              <table class="mt-1 w-full text-xs">
                <tbody>
                  <tr v-for="h in a.holdings" :key="h.name">
                    <td class="py-0.5">{{ h.name }}<span v-if="h.ticker" class="muted"> ({{ h.ticker }})</span></td>
                    <td class="muted py-0.5 text-right">{{ a.balance ? Math.round((h.value / a.balance) * 100) : 0 }}%</td>
                    <td class="w-28 py-0.5 text-right tabular-nums">{{ money(h.value) }}</td>
                  </tr>
                </tbody>
              </table>
            </details>
          </li>
        </ul>

        <form v-if="s.group === 'tsp' && tsp.open" class="mt-3 border-t border-stone-100 pt-4 dark:border-stone-800" @submit.prevent="saveTsp">
          <div class="grid gap-3 sm:grid-cols-3">
            <label v-for="f in tspFunds" :key="f" class="text-sm">
              <span class="label">{{ f }} ($)</span>
              <input v-model.number="tsp.values[f]" type="number" min="0" step="0.01" class="field mt-1 w-full" placeholder="0" />
            </label>
          </div>
          <div class="mt-3 flex flex-wrap items-center gap-3">
            <button type="submit" class="btn-primary" :disabled="tspTotal <= 0">Save TSP</button>
            <button type="button" class="btn" @click="tsp.open = false">Cancel</button>
            <span class="muted text-sm">Total {{ money(tspTotal) }}</span>
          </div>
        </form>
      </section>
    </template>
  </div>
</template>
