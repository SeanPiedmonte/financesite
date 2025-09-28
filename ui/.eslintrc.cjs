module.exports = {
  root: true,
  env: {
    node: true,
    browser: true,
    es2021: true
},
parser: "vue-eslint-parser",
parserOptions: {
  parser: "@typescript-eslint/parser",
  ecmaVersion: 2021,
  sourceType: "module",
},
extends: [
  "eslint:recommended",
  "plugin:vue/vue3-recommended",
  "@vue/eslint-config-typescript/recommended"
],
globals: {
  defineProps: "readonly",
  defineEmits: "readonly",
  defineExpose: "readonly",
  withDefaults: "readonly",
},
rules: {
  "vue/multi-word-component-names":"off",
  "no-undef": "off",
  "@typescript-eslint/no-explicit-any": "off"
}
}
