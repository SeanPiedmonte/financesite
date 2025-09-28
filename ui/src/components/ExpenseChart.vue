<template>
  <div class="w-96 h-96">
    <Pie :data="chartData" :options="chartOptions" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { Pie } from "vue-chartjs";
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from "chart.js";

ChartJS.register(ArcElement, Tooltip, Legend);

const props = defineProps<{
  transactions: { transType: string; transOrigin: string; amount: number };
}>();

const chartData = ref({
  labels: [] as string[],
  datasets: [
    {
      data: [] as number[],
      backgroundColor: ["#f87171", "#60a5fa", "#facc15", "#34d399", "a78bfa"],
    },
  ],
});

const chartOptions = {
  responsive: true,
  plugins: {
    legend: { position: "bottom" as const },
  },
};

watch(
  () => props.transactions,
  (newTxs) => {
    const grouped: Record<string, number> = {};
    for (const tx of newTxs) {
      grouped[tx.transOrigin] = (grouped[tx.transOrigin] || 0) + tx.amount;
    }
    chartData.value.labels = Object.keys(grouped);
    chartData.value.datasets[0].data = Object.values(grouped);
  },
  { immediate: true }
);
</script>
