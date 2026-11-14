<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Connection, type ConnectionKind, type Status } from '../api'

const props = defineProps<{ status: Status }>()

const connections = ref<Connection[]>([])
const message = ref('')
const error = ref('')
const busy = ref('')

const kinds: { kind: ConnectionKind; title: string; body: string }[] = [
  {
    kind: 'bank',
    title: 'Bank accounts and credit cards',
    body: 'Checking, savings and cards. Brings in balances and transactions, already categorised.',
  },
  {
    kind: 'investment',
    title: 'Investments and 401(k)',
    body: 'Brokerage, IRA and employer retirement plans. Brings in balances and holdings.',
  },
  {
    kind: 'mortgage',
    title: 'Mortgage',
    body: 'Your loan servicer. Brings in the balance, interest rate and next payment.',
  },
]
const kindLabel = (k: ConnectionKind) => kinds.find((x) => x.kind === k)?.title ?? k

async function load() {
  try {
    connections.value = await api.connections()
  } catch (e) {
    error.value = (e as Error).message
  }
}

// Plaid Link is Plaid's own hosted login window. Bank credentials are typed
// into it and go straight to Plaid; this app only ever receives a token.
interface PlaidHandler {
  open: () => void
}
interface PlaidGlobal {
  create: (config: {
    token: string
    onSuccess: (publicToken: string, metadata: { institution?: { name?: string } | null }) => void
    onExit: (err: { display_message?: string; error_message?: string } | null) => void
  }) => PlaidHandler
}

let linkScript: Promise<PlaidGlobal> | null = null
function loadLink(): Promise<PlaidGlobal> {
  linkScript ??= new Promise((resolve, reject) => {
    const s = document.createElement('script')
    s.src = 'https://cdn.plaid.com/link/v2/stable/link-initialize.js'
    s.onload = () => resolve((window as unknown as { Plaid: PlaidGlobal }).Plaid)
    s.onerror = () => {
      linkScript = null
      reject(new Error('Could not load Plaid Link. Check your connection or ad blocker.'))
    }
    document.head.appendChild(s)
  })
  return linkScript
}

async function connect(kind: ConnectionKind) {
  error.value = message.value = ''
  busy.value = kind
  try {
    const [{ linkToken }, plaid] = await Promise.all([api.linkToken(kind), loadLink()])
    plaid
      .create({
        token: linkToken,
        onSuccess: async (publicToken, metadata) => {
          try {
            const res = await api.exchange(publicToken, metadata.institution?.name ?? '', kind)
            message.value = res.syncError
              ? `Connected ${res.connection.institution}, but the first sync failed (${res.syncError}). Try “Sync now” in a minute.`
              : `Connected ${res.connection.institution}.`
            await load()
          } catch (e) {
            error.value = (e as Error).message
          } finally {
            busy.value = ''
          }
        },
        onExit: (err) => {
          if (err) error.value = err.display_message || err.error_message || 'Plaid Link closed with an error.'
          busy.value = ''
        },
      })
      .open()
  } catch (e) {
    error.value = (e as Error).message
    busy.value = ''
  }
}

async function sync() {
  error.value = message.value = ''
  busy.value = 'sync'
  try {
    const res = await api.sync()
    const failed = Object.entries(res.errors)
    message.value = `Synced ${res.synced} of ${connections.value.length} connection(s).`
    if (failed.length) error.value = failed.map(([name, why]) => `${name}: ${why}`).join(' · ')
    await load()
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = ''
  }
}

async function disconnect(c: Connection) {
  if (!window.confirm(`Disconnect ${c.institution}? Its accounts and transactions will be removed from this app.`)) return
  try {
    await api.disconnect(c.id)
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function importCsv(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  error.value = message.value = ''
  try {
    const res = await api.importCsv(file)
    message.value = `Imported ${res.imported} transaction(s) from ${res.file}.`
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    input.value = ''
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <h1 class="text-xl font-semibold">Connections</h1>

    <p v-if="message" class="card text-sm" role="status">{{ message }}</p>
    <p v-if="error" class="card text-sm text-red-700 dark:text-red-400" role="alert">{{ error }}</p>

    <section v-if="props.status.demo" class="card text-sm">
      <h2 class="text-base font-semibold">Plaid is not set up yet</h2>
      <p class="mt-1">
        The server is running without Plaid keys, so the buttons below are disabled and the app is
        showing demo data. To connect real accounts:
      </p>
      <ol class="mt-2 list-decimal space-y-1 pl-5">
        <li>Create a free account at dashboard.plaid.com and copy your client ID and Sandbox secret.</li>
        <li>
          Restart the server with <code>PLAID_CLIENT_ID</code>, <code>PLAID_SECRET</code> and
          <code>PLAID_ENV=sandbox</code> set.
        </li>
        <li>In Sandbox, sign in to any bank with <code>user_good</code> / <code>pass_good</code>.</li>
      </ol>
    </section>
    <p v-else class="muted text-sm">
      Plaid environment: <strong>{{ props.status.plaidEnv }}</strong>. You sign in to your bank
      inside Plaid's window — this app never sees your bank username or password.
    </p>

    <div class="grid gap-4 md:grid-cols-3">
      <section v-for="k in kinds" :key="k.kind" class="card flex flex-col">
        <h2 class="font-semibold">{{ k.title }}</h2>
        <p class="muted mt-1 flex-1 text-sm">{{ k.body }}</p>
        <button
          type="button"
          class="btn-primary mt-4 self-start"
          :disabled="props.status.demo || busy !== ''"
          @click="connect(k.kind)"
        >
          {{ busy === k.kind ? 'Opening Plaid…' : 'Connect' }}
        </button>
      </section>
    </div>

    <section class="card">
      <div class="flex items-center justify-between gap-3">
        <h2 class="font-semibold">Connected institutions</h2>
        <button type="button" class="btn" :disabled="props.status.demo || !connections.length || busy !== ''" @click="sync">
          {{ busy === 'sync' ? 'Syncing…' : 'Sync now' }}
        </button>
      </div>
      <p v-if="!connections.length" class="muted mt-2 text-sm">Nothing connected yet.</p>
      <ul v-else class="mt-2 divide-y divide-stone-100 text-sm dark:divide-stone-800">
        <li v-for="c in connections" :key="c.id" class="flex flex-wrap items-center justify-between gap-3 py-2.5">
          <div>
            <span class="font-medium">{{ c.institution }}</span>
            <span class="muted"> · {{ kindLabel(c.kind) }}</span>
            <div class="muted text-xs">
              {{ c.lastSync ? `Last synced ${new Date(c.lastSync).toLocaleString()}` : 'Not synced yet' }}
            </div>
          </div>
          <button type="button" class="btn" @click="disconnect(c)">Disconnect</button>
        </li>
      </ul>
    </section>

    <div class="grid gap-4 md:grid-cols-2">
      <section class="card text-sm">
        <h2 class="text-base font-semibold">Thrift Savings Plan and home value</h2>
        <p class="muted mt-1">
          These two are entered by hand on the Net worth tab. TSP is not dependably available
          through Plaid, and no bank connection reports what a home is worth — a mortgage link
          supplies the loan balance only.
        </p>
      </section>
      <section class="card text-sm">
        <h2 class="text-base font-semibold">Import a CSV statement</h2>
        <p class="muted mt-1">
          For accounts you can't connect. The file needs date, description and amount (or
          debit/credit) columns; categories are guessed from the description.
        </p>
        <input type="file" accept=".csv,text/csv" class="mt-3 block text-sm" aria-label="CSV statement" @change="importCsv" />
      </section>
    </div>
  </div>
</template>
