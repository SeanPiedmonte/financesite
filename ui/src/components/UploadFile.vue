<template>
  <h2>Upload Expenses</h2>
  <input type="file" accept=".csv" @change="uploadFile" />
  <div v-if="loading">Uploading...</div>
  <div v-else-if="transactions.length">
    <h3>Transactions:</h3>
    <ul>
      <li v-for="(txn, idx) in transactions" :key="idx">
        {{ txn.transType }} | {{ txn.transOrigin }} | ${{ txn.amount }}a
      </li>
    </ul>
  </div>
</template>

<script setup>
import { ref } from "vue";

const transactions = ref([]);
const loading = ref(false);

async function uploadFile(event) {
  const file = event.target.files[0];
  if (!file) return;

  loading.value = true;

  try {
    const formData = new FormData();
    formData.append("FILEKEY", file);
    const res = await fetch("http://localhost:8080/api/uploadExpenses", {
      method: "POST",
      body: formData,
    });
    const data = await res.json();
    console.log(data);
    transactions.value = data;
  } catch (err) {
    console.error("Upload Failed:", err);
  } finally {
    loading.value = false;
  }
}
</script>
