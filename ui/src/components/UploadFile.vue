<template>
  <div class="p-4">
    <input type="file" @change="handleFileChange" />
    <button
      class="mt-2 px-4 py-2 bg-blue-500 text-black rounded"
      @click="uploadFile"
      :disabled="!selectedFile"
    >
      UploadFile
    </button>
  </div>
</template>

<script>
export default {
  name: "upload-file",
  data() {
    return {
      postId: null,
    };
  },
  created() {
    const requestOptions = {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title: "Vue POST Request Example" }),
    };
    fetch("http://localhost:8080/api/expenses", requestOptions)
      .then((response) => response.json())
      .then((data) => (this.postId = data.id));
  },
};
</script>
