<template>
  <NavBar />
  <Main msg="Personal Finance Planner" />
  <div class="container">
    <div class="info">
      <h2>Finance Insights</h2>
      <p>Here is for supporting text and other template</p>
      <UploadFile />
      <GetExpenseButton />
    </div>
    <div class="chart">
      <Pie :data="data" :options="options" />
    </div>
  </div>
</template>

<script lang="ts">
import { Options, Vue } from "vue-class-component";
import Main from "./components/Main.vue";
import NavBar from "./components/NavBar.vue";
import UploadFile from "./components/UploadFile.vue";
import GetExpenseButton from "./components/GetExpenseButton.vue";
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from "chart.js";
import { Pie } from "vue-chartjs";
import * as chartConfig from "@/data/chartConfig.json";
import "@/output.css";

ChartJS.register(ArcElement, Tooltip, Legend);
@Options({
  components: {
    NavBar,
    Main,
    Pie,
    UploadFile,
  },
})
export default class App extends Vue {
  data() {
    return chartConfig;
  }
}
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
}

.container {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px;
}

.info {
  flex: 1;
  padding: 20px;
  text-align: left;
}

.chart {
  flex: 1;
}
</style>
