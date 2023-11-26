<script setup lang="ts">
import { onMounted, ref } from "vue";
import NewAppModal from "./components/NewAppModal.vue";
import DeployCommitModal from "./components/DeployCommitModal.vue";

const apps = ref<any>([]);

onMounted(async () => {
  const appsReq = await fetch("/runner/api/app");
  apps.value = await appsReq.json();

  console.log(apps.value);
});
</script>

<template>
  <NewAppModal />
  <DeployCommitModal appId="" />
  <main class="container p-5">
    <!-- Top Row -->
    <div class="row m-3">
      <h1 class="col-2">Apps</h1>
      <button class="btn btn-success col-2 offset-8" type="button" data-bs-toggle="modal" data-bs-target="#newAppModal">
        Add New App
      </button>
    </div>

    <!-- Apps List -->
    <div style="cursor: pointer" class="card m-3" v-for="app in apps">
      <h4 class="card-header">{{ app.name }}</h4>

      <div class="card-body">
        <p class="card-text">
          <strong>Template:</strong> {{ app.template_id }}
        </p>
        <p class="card-text"><strong>Git URL:</strong> {{ app.git_url }}</p>
        <button class="btn btn-primary me-3" type="button" data-bs-toggle="modal" data-bs-target="#deployCommitModal">
          Deploy Commit
        </button>
        <button class="btn btn-danger" type="button">Delete</button>
      </div>
    </div>
  </main>
</template>

<style scoped></style>
