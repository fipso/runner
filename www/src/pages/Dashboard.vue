<script lang="ts" setup>
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import NewAppModal from "../components/NewAppModal.vue";

const apps = ref<any>([]);
const info = ref<any>(null);

const newAppModalRef = ref<InstanceType<typeof NewAppModal> | null>(null);

const router = useRouter();

const loadData = async () => {
  const infoReq = await fetch("/runner/api/info");
  info.value = await infoReq.json();

  const appsReq = await fetch("/runner/api/app");
  apps.value = await appsReq.json();
};

onMounted(async () => {
  loadData();
});
</script>

<template>
  <NewAppModal v-if="info?.templates" ref="newAppModalRef" :templates="info.templates" @success="loadData" />

  <main class="container p-5">
    <!-- Top Row -->
    <div class="row m-3 mb-5">
      <h1 class="col-2">Apps</h1>
      <button class="btn fs-4 btn-primary col-2 offset-8" type="button" @click="newAppModalRef?.show()">
        Add New App
      </button>
    </div>

    <!-- App List -->
    <div class="row">
      <div v-for="app in apps" class="col-3 card" style="cursor: pointer" @click="router.push(`/app/${app.id}`)">
        <div class="card-body">
          <h2 class="card-title m-0">{{ app.name }}</h2>
          <span class="text-secondary mb-2 d-block">{{ info.templates[app.template_id].name }}</span>
          <span v-if="app.deployments.length" class="badge bg-success">{{ app.deployments[app.deployments.length - 1].status }}</span>
        </div>
      </div>
    </div>
  </main>
</template>
