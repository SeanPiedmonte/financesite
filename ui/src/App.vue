<script setup lang="ts">
import { ref, onMounted } from "vue";

const count = ref(0);
const msg = "My example Message";
const rawHTML = '<span style="color:red">this should be red.</span>'
function increment() {
  count.value++;
}

onMounted(() => {
  console.log("the initial count is ${count.value}.")
})
</script>

<template>
  <span>Message: {{ msg }}</span>
  <p>Using text interpolation: {{ rawHTML }}</p>
  <p>Using v-html directive: <span v-html="rawHTML"></span></p>
  <button @click="increment">Count is: {{ count }}</button>
</template>

<style scoped></style>
