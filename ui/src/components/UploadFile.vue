<template>
  <div class="p-4">
    <input type="file" @change="handleFileUpload" />
    <button
      class="mt-2 px-4 py-2 bg-blue-500 text-black rounded"
      @click="uploadFile"
    >
      UploadFile
    </button>
  </div>
</template>

<script>
export default {
  data() {
    return { selectedFile: null };
  },
  methods: {
    handleFileUpload(event) {
      this.selectedFile = event.target.files[0];
    },
    async uploadFile() {
      const formData = new FormData();
      formData.append("FILEKEY", this.selectedFile);

      const res = await fetch("http://localhost:8080/api/upload", {
        method: "POST",
        body: formData,
      });

      //const data = await res.json();
      const data = await res;
      console.log("File uploaded:", data);
    },
  },
};
</script>
