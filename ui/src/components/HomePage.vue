<script setup lang="ts">
import { ref, onMounted } from "vue";

interface Transaction {
  transType: string
  transOrigin: string
  amount: number
}

const loading = ref(false);
const transactions = ref<Transaction[]>([])

async function getExpenses() {
  const url = "http://localhost:8080/api/expenses";
  try {
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`Response status: ${response.status}`);
    }

    const result = await response.json();
    console.log(result);
    transactions.value = result;
  } catch (error) {
    const msg = (error as Error).message;
    console.log(msg);
  }
}

async function uploadExpenses(event: Event) {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  loading.value = true;

  try {
    const formData = new FormData();
    formData.append("FILEKEY", file);
    const res = await fetch("http://localhost:8080/api/uploadExpenses", {
      method: "POST",
      body: formData,
    });
    const data = await res.text();
    console.log(data);
  } catch (err) {
    console.error("Upload Failed:", err);
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
})
</script>

<template>
  <h2>UploadExpenses</h2>
  <input type="file" accept=".csv" @change="uploadExpenses" />
  <div v-if="loading">Uploading...</div>
  <div v-else-if="transactions.length">
    <h3>Transactions:</h3>
    <ul>
      <li v-for="(txn, idx) in transactions" :key="idx">
        {{ txn.transType }} | {{ txn.transOrigin }} | ${{ txn.amount }}
      </li>
    </ul>
  </div>
  <button @click="getExpenses" class="text-white bg-blue-700 hover:bg-blue-800 focus:outline-none focus:ring-4 focus:ring-blue-300 font-medium rounded-full text-sm px-5 py-2.5 text-center me-2 mb-2 dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-blue-800">Get Expenses</button>
</template>

<style scoped></style>
