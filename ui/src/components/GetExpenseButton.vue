<template>
  <button @click="getExpenses">Get Expenses</button>
  <ul>
    <li v-for="expense in expenses" :key="expense.id" data-test="expense">
      {{ expense.title }}
    </li>
  </ul>
</template>

<script>
import axios from "axios";

export default {
  data() {
    return {
      expenses: null,
    },
  },
  methods: {
    async getExpenses() {
      this.expenses = await axios.get("/localhost:8080/expenses");
    },
  },
};
</script>
