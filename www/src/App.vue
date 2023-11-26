<script setup lang="ts">
import { onMounted, ref } from "vue";
import NewAppModal from "./components/NewAppModal.vue";
import DeployCommitModal from "./components/DeployCommitModal.vue";

const apps = ref<any>([]);

const newAppModalRef = ref<InstanceType<typeof NewAppModal> | null>(null);
const deployCommitModalRef = ref<InstanceType<typeof DeployCommitModal> | null>(
  null,
);

onMounted(async () => {
  const appsReq = await fetch("/runner/api/app");
  apps.value = await appsReq.json();
});
</script>

<template>
  <NewAppModal ref="newAppModalRef" />
  <DeployCommitModal ref="deployCommitModalRef" />
  <main class="container p-5">
    <!-- Top Row -->
    <div class="row m-3">
      <h1 class="col-2">Apps</h1>
      <button class="btn btn-success col-2 offset-8" type="button" @click="newAppModalRef?.show()">
        Add New App
      </button>
    </div>

    <!-- Apps List -->
    <div class="card m-3" v-for="app in apps">
      <h4 class="card-header">{{ app.name }}</h4>

      <div class="card-body">
        <p class="card-text">
          <strong>Template:</strong> {{ app.template_id }}
          <br />
          <strong>Git URL:</strong> {{ app.git_url }}
        </p>
        <p class="card-text"></p>
        <!-- Deployments List -->
        <ul class="list-group">
          <li class="list-group-item mb-3" v-for="deployment in app.deployments">
            <div class="d-flex justify-content-between">
              <strong>{{ deployment.git_branch }}/{{ deployment.git_commit }}</strong>
              <span class="badge bg-success text-light">{{
                deployment.status
              }}</span>
            </div>
            <p>Public URL:</p>
          </li>
        </ul>

        <button class="btn btn-primary me-3" type="button" @click="deployCommitModalRef?.show(app.id)">
          Deploy Commit
        </button>
        <button class="btn btn-danger" type="button">Delete</button>
      </div>
    </div>
  </main>
</template>

<style scoped></style>
