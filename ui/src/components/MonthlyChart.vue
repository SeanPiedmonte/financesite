<script setup lang="ts">
import { computed, ref } from 'vue'
import { money, monthLabel, moneyWhole, type MonthTotal } from '../api'

const props = defineProps<{ trend: MonthTotal[]; selected: string }>()
const emit = defineEmits<{ select: [month: string] }>()

const W = 480
const H = 200
const pad = { top: 22, right: 8, bottom: 24, left: 8 }
const hovered = ref<string | null>(null)

// Round the axis top up to a clean number so the tallest bar has headroom.
const top = computed(() => {
  const max = Math.max(1, ...props.trend.map((t) => t.spent))
  const step = 10 ** Math.floor(Math.log10(max))
  return Math.ceil(max / step) * step
})

const bars = computed(() => {
  const n = props.trend.length
  const slot = (W - pad.left - pad.right) / Math.max(n, 1)
  const width = Math.min(44, slot * 0.6)
  const plot = H - pad.top - pad.bottom
  return props.trend.map((t, i) => {
    const h = Math.max(0, (t.spent / top.value) * plot)
    const cx = pad.left + slot * i + slot / 2
    return { ...t, cx, x: cx - width / 2, y: H - pad.bottom - h, w: width, h, slotX: pad.left + slot * i, slot }
  })
})

// A bar with a rounded top and a square foot on the baseline.
function path(b: { x: number; y: number; w: number; h: number }) {
  const r = Math.min(4, b.h, b.w / 2)
  const base = b.y + b.h
  return `M${b.x},${base} V${b.y + r} Q${b.x},${b.y} ${b.x + r},${b.y} H${b.x + b.w - r} Q${b.x + b.w},${b.y} ${b.x + b.w},${b.y + r} V${base} Z`
}
</script>

<template>
  <figure>
    <svg :viewBox="`0 0 ${W} ${H}`" class="w-full" role="img" aria-label="Spending per month">
      <line
        :x1="pad.left"
        :x2="W - pad.right"
        :y1="H - pad.bottom"
        :y2="H - pad.bottom"
        stroke="currentColor"
        stroke-opacity="0.2"
      />
      <g v-for="b in bars" :key="b.month">
        <path
          :d="path(b)"
          :fill="b.month === selected || b.month === hovered ? 'var(--mark)' : 'var(--mark-quiet)'"
        />
        <text
          v-if="b.month === selected || b.month === hovered"
          :x="b.cx"
          :y="b.y - 6"
          text-anchor="middle"
          class="fill-current text-[12px] font-medium"
        >
          {{ moneyWhole(b.spent) }}
        </text>
        <text
          :x="b.cx"
          :y="H - 7"
          text-anchor="middle"
          class="fill-current text-[11px]"
          :opacity="b.month === selected ? 1 : 0.6"
        >
          {{ monthLabel(b.month, 'short') }}
        </text>
        <!-- Full-height hit target, wider than the bar. -->
        <rect
          :x="b.slotX"
          :y="0"
          :width="b.slot"
          :height="H"
          fill="transparent"
          class="cursor-pointer"
          tabindex="0"
          role="button"
          :aria-label="`${monthLabel(b.month)}: ${money(b.spent)} spent`"
          @mouseenter="hovered = b.month"
          @mouseleave="hovered = null"
          @focus="hovered = b.month"
          @blur="hovered = null"
          @click="emit('select', b.month)"
          @keydown.enter="emit('select', b.month)"
        >
          <title>{{ monthLabel(b.month) }}: {{ money(b.spent) }} spent, {{ money(b.income) }} income</title>
        </rect>
      </g>
    </svg>
  </figure>
</template>
