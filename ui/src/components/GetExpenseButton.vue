<template>
  <button @click="getExpenses">Get Expenses</button>
  <ul>
    <li v-for="expense in expenses" :key="expense.id" data-test="expense">
      {{ expense.title }}
    </li>
  </ul>
</template>

<script>
export default {
  data() {
    return {
      expenses: null,
    };
  },
  methods: {
    async getExpenses() {
      const response = fetch("http://localhost:8080/api/expenses");
      this.expenses = await response.json();
      console.log("Requested");
    },
  },
};
</script>
