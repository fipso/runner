<script setup lang="ts">
import { onMounted, ref } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";

import { statusColor } from "../util";

import DeployCommitModal from "../components/DeployCommitModal.vue";
import UpdateEnvModal from "../components/UpdateEnvModal.vue";

const route = useRoute();
const router = useRouter();

const app = ref<any>(null);

const deployCommitModalRef = ref<InstanceType<typeof DeployCommitModal> | null>(
  null,
);
const updateEnvModalRef = ref<InstanceType<typeof UpdateEnvModal> | null>(null);

const loadData = async () => {
  const appReq = await fetch(`/runner/api/app/${route.params.id}`);
  app.value = await appReq.json();
};

const deleteApp = async (id: string) => {
  if (!confirm("Are you sure you want to delete this app?")) {
    return;
  }
  await fetch(`/runner/api/app/${id}`, { method: "DELETE" });
  //loadData();
  router.push("/");
};

const deleteDeployment = async (id: string) => {
  if (!confirm("Are you sure you want to delete this deployment?")) {
    return;
  }
  await fetch(`/runner/api/deployment/${id}`, { method: "DELETE" });
  loadData();
};

onMounted(async () => {
  loadData();
});
</script>

<template>
  <DeployCommitModal ref="deployCommitModalRef" @closed="loadData" />
  <UpdateEnvModal ref="updateEnvModalRef" @success="loadData" />

  <main v-if="app" class="container p-5">
    <RouterLink to="/">Back</RouterLink>

    <!-- Apps List -->
    <div class="card m-3">
      <h4 class="card-header">{{ app.name }}</h4>

      <div class="card-body">
        <p class="card-text">
          <strong>Template:</strong> {{ app.template_id }}
          <br />
          <strong>Git URL:</strong> {{ app.git_url }}
          <br />
          <strong>Push Webhook URL:</strong> {{ app.webhook_url
          }}<select style="border: 1px solid lightgray; padding: 2px">
            <option>github</option>
            <option>gitlab</option>
          </select>
          <br />
          <strong>Webhook Secret:</strong> {{ app.webhook_secret }}
        </p>
        <p class="card-text"></p>
        <!-- Deployments List -->
        <ul class="list-group mb-3 overflow-y-scroll" style="max-height: 600px; !important">
          <li class="list-group-item" v-for="deployment in app?.deployments">
            <div class="d-flex justify-content-between align-items-center">
              <div>
                <strong class="me-1">{{ deployment.name }}</strong>
                <span class="text-secondary">{{ deployment.time }}</span>
              </div>
              <span class="badge fs-6 text-light" :class="`bg-${statusColor(deployment.status)}`">{{ deployment.status
              }}</span>
            </div>
            <p class="m-0 mb-2">
              Public URL:
              <a :href="deployment.url" target="_blank">{{ deployment.url }}</a>
              <br />
              Container ID: {{ deployment.container_id }}
            </p>

            <div class="d-flex justify-content-between align-items-center">
              <button class="btn btn-sm btn-outline-primary" @click="router.push(`/deployment/${deployment.id}/logs`)">
                <i class="bi bi-list-columns-reverse"></i> Logs
              </button>
              <div>
                <!--
                <button class="btn btn-warning btn-sm me-1" type="button">
                  SSH
                </button>
                -->
                <button class="btn btn-danger btn-sm" type="button" @click="deleteDeployment(deployment.id)">
                  <i class="bi bi-trash3"></i> Delete
                </button>
              </div>
            </div>
          </li>
        </ul>

        <button class="btn btn-primary me-3" type="button" @click="deployCommitModalRef?.show(app.id)">
          <i class="bi bi-box-seam"></i> Deploy Manually
        </button>
        <button class="btn btn-warning me-3" type="button" @click="updateEnvModalRef?.show(app.id, app.env)">
          <i class="bi bi-card-list"></i> Environment Vars
        </button>
        <button class="btn btn-danger" type="button" @click="deleteApp(app.id)">
          <i class="bi bi-trash3"></i> Delete
        </button>
      </div>
    </div>
  </main>
</template>

<style scoped></style>
