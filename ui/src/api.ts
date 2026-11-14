export interface Category {
  id: string
  label: string
  spending: boolean
}

export interface Transaction {
  id: string
  accountId: string
  date: string
  name: string
  merchant: string
  amount: number // positive = money out
  pending: boolean
  category: string
  categorySource: 'manual' | 'rule' | 'source'
}

export interface CategoryTotal {
  category: string
  label: string
  amount: number
  count: number
  share: number
}

export interface MonthTotal {
  month: string
  spent: number
  income: number
}

export interface SpendingSummary {
  month: string
  months: string[]
  spent: number
  income: number
  byCategory: CategoryTotal[]
  trend: MonthTotal[]
  topMerchants: { name: string; amount: number; count: number }[]
}

export interface Holding {
  name: string
  ticker?: string
  quantity?: number
  value: number
}

export type Group = 'cash' | 'credit' | 'investment' | 'retirement' | 'tsp' | 'mortgage'

export interface Account {
  id: string
  institution: string
  name: string
  mask?: string
  group: Group
  subtype?: string
  balance: number
  manual: boolean
  updatedAt?: string
  holdings?: Holding[]
  mortgage?: {
    interestRate?: number
    nextPayment?: number
    nextPaymentDue?: string
    originalBalance?: number
    propertyAddress?: string
  }
}

export interface PropertyView {
  id: string
  name: string
  address?: string
  value: number
  valueAsOf?: string
  mortgageAccountId?: string
  mortgageBalance: number
  equity: number
}

export interface AccountsResponse {
  accounts: Account[]
  summary: {
    assets: number
    liabilities: number
    netWorth: number
    groups: Record<string, number>
    properties: PropertyView[]
  }
}

export interface Rule {
  id: string
  match: string
  category: string
}

export type ConnectionKind = 'bank' | 'investment' | 'mortgage'

export interface Connection {
  id: string
  institution: string
  kind: ConnectionKind
  lastSync?: string
}

export interface Status {
  demo: boolean
  plaidEnv: string
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init)
  if (res.status === 204) return undefined as T
  const body = await res.json().catch(() => null)
  if (!res.ok) {
    throw new Error(body?.error ?? `Request failed (${res.status})`)
  }
  return body as T
}

const json = (method: string, body: unknown): RequestInit => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
})

export const api = {
  status: () => request<Status>('/api/status'),
  categories: () => request<Category[]>('/api/categories'),
  spending: (month = '') => request<SpendingSummary>(`/api/spending?month=${month}`),
  transactions: (month = '', category = '') =>
    request<Transaction[]>(`/api/transactions?month=${month}&category=${category}`),
  categorize: (id: string, category: string) =>
    request<Transaction>(`/api/transactions/${encodeURIComponent(id)}`, json('PATCH', { category })),
  importCsv: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return request<{ imported: number; file: string }>('/api/transactions/import', {
      method: 'POST',
      body: form,
    })
  },
  rules: () => request<Rule[]>('/api/rules'),
  addRule: (match: string, category: string) =>
    request<Rule>('/api/rules', json('POST', { match, category })),
  deleteRule: (id: string) => request<void>(`/api/rules/${id}`, { method: 'DELETE' }),
  accounts: () => request<AccountsResponse>('/api/accounts'),
  saveManualAccount: (account: Partial<Account>, id?: string) =>
    request<Account>(
      id ? `/api/accounts/manual/${id}` : '/api/accounts/manual',
      json(id ? 'PUT' : 'POST', account),
    ),
  deleteManualAccount: (id: string) =>
    request<void>(`/api/accounts/manual/${id}`, { method: 'DELETE' }),
  saveProperty: (
    property: { name: string; address?: string; value: number; mortgageAccountId?: string },
    id?: string,
  ) =>
    request<PropertyView>(
      id ? `/api/properties/${id}` : '/api/properties',
      json(id ? 'PUT' : 'POST', property),
    ),
  deleteProperty: (id: string) => request<void>(`/api/properties/${id}`, { method: 'DELETE' }),
  connections: () => request<Connection[]>('/api/connections'),
  linkToken: (kind: ConnectionKind) =>
    request<{ linkToken: string }>('/api/plaid/link-token', json('POST', { kind })),
  exchange: (publicToken: string, institution: string, kind: ConnectionKind) =>
    request<{ connection: Connection; syncError?: string }>(
      '/api/plaid/exchange',
      json('POST', { publicToken, institution, kind }),
    ),
  sync: () =>
    request<{ synced: number; errors: Record<string, string> }>('/api/plaid/sync', {
      method: 'POST',
    }),
  disconnect: (id: string) => request<void>(`/api/connections/${id}`, { method: 'DELETE' }),
}

const usd = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' })
const usdWhole = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
  maximumFractionDigits: 0,
})

export const money = (n: number) => usd.format(n)
export const moneyWhole = (n: number) => usdWhole.format(n)

export function monthLabel(month: string, style: 'long' | 'short' = 'long') {
  const [y, m] = month.split('-').map(Number)
  if (!y || !m) return month
  return new Date(y, m - 1, 1).toLocaleDateString('en-US', {
    month: style,
    year: style === 'long' ? 'numeric' : undefined,
  })
}

export function dayLabel(date: string) {
  const [y, m, d] = date.split('-').map(Number)
  if (!y || !m || !d) return date
  return new Date(y, m - 1, d).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}
