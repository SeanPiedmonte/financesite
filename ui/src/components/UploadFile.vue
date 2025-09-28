<template>
  <h2>Upload Expenses</h2>
  <input type="file" accept=".csv" @change="uploadFile" />
  <div v-if="loading">Uploading...</div>
  <ExpenseChart :transactions="transactions" />
</template>

<script setup>
import { ref, onMounted } from "vue";
import ExpenseChart from "./ExpenseChart.vue";


const loading = ref(false);

async function uploadFile(event) {
  const file = event.target.files[0];
  if (!file) return;

  loading.value = true;
  //const transactions = ref<any[]>([]);

  try {
    const formData = new FormData();
    formData.append("FILEKEY", file);
    const res1 = await fetch("http://localhost:8080/api/uploadExpenses", {
      method: "POST",
      body: formData,
    });
    const data1 = await res1.text();
    console.log(data1);
  } catch (err) {
    console.error("Upload Failed:", err);
  } finally {
    loading.value = false;
  }

  onMounted(async () => {
    const res = await fetch("http://localhost:8080/api/expenses", {
      method: "GET",
    });
    transactions.value = await res.json();
  });
}
</script>
