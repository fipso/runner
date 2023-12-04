<script setup lang="ts">
import { onMounted, ref } from "vue";
import { RouterLink } from "vue-router";
import NewAppModal from "../components/NewAppModal.vue";
import DeployCommitModal from "../components/DeployCommitModal.vue";

const info = ref<any>({});
const apps = ref<any>([]);

const newAppModalRef = ref<InstanceType<typeof NewAppModal> | null>(null);
const deployCommitModalRef = ref<InstanceType<typeof DeployCommitModal> | null>(
  null,
);

const loadApps = async () => {
  const infoReq = await fetch("/runner/api/info");
  info.value = await infoReq.json();

  const appsReq = await fetch("/runner/api/app");
  apps.value = await appsReq.json();
};

const deleteApp = async (id: string) => {
  if (!confirm("Are you sure you want to delete this app?")) {
    return;
  }
  await fetch(`/runner/api/app/${id}`, { method: "DELETE" });
  loadApps();
};

const deleteDeployment = async (id: string) => {
  if (!confirm("Are you sure you want to delete this deployment?")) {
    return;
  }
  await fetch(`/runner/api/deployment/${id}`, { method: "DELETE" });
  loadApps();
};

onMounted(async () => {
  loadApps();
});
</script>

<template>
  <NewAppModal v-if="info.templates" ref="newAppModalRef" :templates="info.templates" @success="loadApps" />
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
                <RouterLink :to="`/deployment/${deployment.id}/logs/build`">Build</RouterLink>{{ " " }}
                <RouterLink :to="`/deployment/${deployment.id}/logs/running`">Running</RouterLink>
              </p>
              <div>
                <!--
                <button class="btn btn-warning btn-sm me-1" type="button">
                  SSH
                </button>
                -->
                <button class="btn btn-danger btn-sm" type="button" @click="deleteDeployment(deployment.id)">
                  Delete
                </button>
              </div>
            </div>
          </li>
        </ul>

        <button class="btn btn-primary me-3" type="button" @click="deployCommitModalRef?.show(app.id)">
          Deploy Commit
        </button>
        <button class="btn btn-warning me-3" type="button">Update Env</button>
        <button class="btn btn-danger" type="button" @click="deleteApp(app.id)">
          Delete
        </button>
      </div>
    </div>
  </main>
</template>

<style scoped></style>
