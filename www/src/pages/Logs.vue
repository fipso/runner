<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { useRoute } from "vue-router";

const route = useRoute();
const deploymentId = route.params.id;
const logType = route.params.logType;

const logContent = ref("");

const loadLogs = async () => {
  const logsReq = await fetch(
    `/runner/api/deployments/${deploymentId}/logs/${logType}`,
  );
  logContent.value = (await logsReq.json()).logs;
};

onMounted(() => {
  loadLogs();
});
</script>

<template>
  <main class="container p-5">
    <h1>Logs</h1>
    <pre>{{ logContent }}</pre>
  </main>
</template>
