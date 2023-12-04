<script lang="ts" setup>
import { onMounted, onUnmounted, ref, defineProps } from "vue";

const props = defineProps<{
  deploymentId: string;
  logType: string;
}>();

const emit = defineEmits(["buildDone"]);

const logs = ref("");
const loadingLogs = ref(false);
const buildDone = ref(false);
const deploymentUrl = ref("");
let logsInterval: any = null;

const loadLogs = async () => {
  if (loadingLogs.value) {
    return;
  }
  loadingLogs.value = true;
  buildDone.value = false;
  try {
    const logsReq = await fetch(
      `/runner/api/deployment/${props.deploymentId}/logs/${props.logType}`,
    );
    const data = await logsReq.json();

    logs.value = data.logs;
    deploymentUrl.value = data.url;

    if (props.logType == "build" && data.build_status !== "Building") {
      buildDone.value = true;
      clearInterval(logsInterval);
      emit("buildDone", data.build_status);
    }
  } catch (err) {
    console.log(err);
  }
  loadingLogs.value = false;
};

onMounted(() => {
  loadLogs();
  logsInterval = setInterval(loadLogs, 500);
});

onUnmounted(() => {
  clearInterval(logsInterval);
});
</script>

<template>
  <div v-if="buildDone || deploymentUrl" class="alert alert-success">
    <span v-if="props.logType == 'build' && buildDone">Deployment was built successfully</span>
    <span v-if="props.logType == 'running' && deploymentUrl">Deployment successfully deployed to:
      <a :href="deploymentUrl" target="_blank">{{ deploymentUrl }}</a></span>
  </div>
  <pre>{{ logs }}</pre>
</template>
