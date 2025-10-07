<template>
  <Navbar />
  <HomePage />
</template>

<script lang="ts">
import Navbar from "./components/Navbar.vue";
import HomePage from "./components/HomePage.vue";

export default {
  name: "App",
  components: {
    HomePage,
    Navbar,
  }
}
</script>

<style scoped></style>
