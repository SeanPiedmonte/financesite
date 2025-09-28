<template>
  <h2>Upload Expenses</h2>
  <input type="file" accept=".csv" @change="uploadFile" />
  <div v-if="loading">Uploading...</div>
  <div v-else-if="transactions.length">
    <h3>Transactions:</h3>
    <ul>
      <li v-for="(txn, idx) in transactions" :key="idx">
        {{ txn.transType }} | {{ txn.transOrigin }} | ${{ txn.amount }}
      </li>
    </ul>
  </div>
  <div class="chart">
    <Pie v-if="!loading" :data="transactions" :options="options" />
  </div>
</template>

<script setup>
import { ref } from "vue";
import { Chart as ChartJS, ArcElement, ToolTip, Legend } from "chart.js";
import { Pie } from "vue-chartjs";

ChartJS.register(ArcElement, ToolTip, Legend);

const transactions = ref([]);
const loading = ref(false);
const options = {
  responsive: true,
  plugins: {
    legend: { position: "top" },
  },
};
async function uploadFile(event) {
  const file = event.target.files[0];
  if (!file) return;

  loading.value = true;

  try {
    const formData = new FormData();
    formData.append("FILEKEY", file);
    const res1 = await fetch("http://localhost:8080/api/uploadExpenses", {
      method: "POST",
      body: formData,
    });
    const res2 = await fetch("http://localhost:8080/api/expenses", {
      method: "GET",
    });
    const data1 = await res1.text();
    const data2 = await res2.json();
    console.log(data1);
    console.log(data2);
    transactions.value = data2;
  } catch (err) {
    console.error("Upload Failed:", err);
  } finally {
    loading.value = false;
  }
}
</script>
