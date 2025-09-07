<template>
  <div class="p-4">
    <input type="file" @change="handleFileChange" />
    <button
      class="mt-2 px-4 py-2 bg-blue-500 text-white rounded"
      @click="uploadFile"
      :disabled="!selectedFile"
    >
      UploadFile
    </button>
  </div>
</template>

<script setup>
import { ref } from "vue";

const selectedFile = ref(null);

function handleFileChange(event) {
  selectedFile.value = event.target.files[0];
}

async function uploadFile() {
  if (!selectedFile.value) return;

  const formData = new FormData();
  formData.append("file", selectedFile.value);

  try {
    const res = await fetch("http://localhost:8080/upload", {
      method: "POST",
      body: formData,
    });

    if (!res.ok) throw new Error("Upload failed");
    const data = await res.json();
    console.log("Upload successful:", data);
  } catch (err) {
    console.error("Error uploading file:", err);
  }
}
</script>
