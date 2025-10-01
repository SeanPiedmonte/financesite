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
    console.error(error.message);
  }
}

onMounted(() => {
})
</script>

<template>
  <button @click="getExpenses">Get Expenses</button>
</template>

<style scoped></style>
