<script setup lang="ts">
import { onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import NewAppModal from "../components/NewAppModal.vue";
import DeployCommitModal from "../components/DeployCommitModal.vue";

const info = ref({});
const apps = ref<any>([]);

const newAppModalRef = ref<InstanceType<typeof NewAppModal> | null>(null);
const deployCommitModalRef = ref<InstanceType<typeof DeployCommitModal> | null>(
  null,
);

const loadApps = async () => {
  const appsReq = await fetch("/runner/api/app");
  apps.value = await appsReq.json();
};

onMounted(async () => {
  const infoReq = await fetch("/runner/api/info");
  info.value = await infoReq.json();

  loadApps();
});
</script>

<template>
  <NewAppModal ref="newAppModalRef" @success="loadApps" />
  <DeployCommitModal ref="deployCommitModalRef" @closed="loadApps" />
  <main class="container p-5">
    <!-- Top Row -->
    <div class="row m-3">
      <h1 class="col-2">Apps</h1>
      <button class="btn fs-4 btn-primary col-2 offset-8" type="button" @click="newAppModalRef?.show()">
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
        <ul class="list-group mb-3">
          <li class="list-group-item" v-for="deployment in app.deployments">
            <div class="d-flex justify-content-between align-items-center">
              <strong>{{ deployment.git_branch }}/{{ deployment.git_commit }}</strong>
              <span class="badge fs-6 bg-success text-light">{{
                deployment.status
              }}</span>
            </div>
            <p class="m-0">
              Public URL:
              <a :href="deployment.url" target="_blank">{{ deployment.url }}</a>
            </p>

            <div class="d-flex justify-content-between align-items-center">
              <p class="m-0">
                Logs:
                <RouterLink :to="`/deployments/${deployment.id}/logs/build`">Build</RouterLink>{{ " " }}
                <RouterLink :to="`/deployments/${deployment.id}/logs/running`">Running</RouterLink>
              </p>
              <div>
                <button class="btn btn-warning btn-sm me-1" type="button">
                  SSH
                </button>
                <button class="btn btn-danger btn-sm" type="button">
                  Delete
                </button>
              </div>
            </div>
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
