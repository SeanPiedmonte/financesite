<script setup lang="ts">
import { onMounted } from "vue";


async function getExpenses() {
  const url = "http://localhost:8080/api/expenses";
  try {
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`Response status: ${response.status}`);
    }

    const result = await response.json();
    console.log(result);
  } catch (error) {
    let msg = (error as Error).message;
    console.log(msg);
  }
}

onMounted(() => {
})
</script>

<template>
  <button @click="getExpenses" class="text-white bg-blue-700 hover:bg-blue-800 focus:outline-none focus:ring-4 focus:ring-blue-300 font-medium rounded-full text-sm px-5 py-2.5 text-center me-2 mb-2 dark:bg-blue-600 dark:hover:bg-blue-700 dark:focus:ring-blue-800">Get Expenses</button>
</template>

<style scoped></style>
