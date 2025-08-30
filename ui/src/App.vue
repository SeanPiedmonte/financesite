<template>
  <NavBar />
  <Main msg="Personal Finance Planner" />
  <Graph />
</template>

<script lang="ts">
import { Options, Vue } from "vue-class-component";
import Main from "./components/Main.vue";
import NavBar from "./components/NavBar.vue";
import Graph from "./components/Graph.vue";
import "@/output.css";

@Options({
  components: {
    NavBar,
    Main,
    Graph,
  },
})
export default class App extends Vue {}
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
}
</style>
