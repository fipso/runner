<script lang="ts" setup>
import { onMounted, ref, defineProps } from "vue";

const props = defineProps<{
  deploymentId: string;
  logType: string;
}>();

const emit = defineEmits(["buildDone"]);

const buildLogs = ref("");
const loadingBuildLogs = ref(false);
const buildDone = ref(false);
let logsInterval: any = null;

const loadBuildLogs = async () => {
  if (loadingBuildLogs.value) {
    return;
  }
  loadingBuildLogs.value = true;
  buildDone.value = false;
  try {
    const logsReq = await fetch(
      `/runner/api/deployments/${props.deploymentId}/logs/build`,
    );
    const data = await logsReq.json();

    buildLogs.value = data.logs;

    if (data.build_status !== "Building") {
      buildDone.value = true;
      clearInterval(logsInterval);
      emit("buildDone");
    }
  } catch (err) {
    console.log(err);
  }
  loadingBuildLogs.value = false;
};

onMounted(() => {
  loadBuildLogs();
  logsInterval = setInterval(loadBuildLogs, 500);
});
</script>

<template>
  <div v-if="buildDone" class="alert alert-success">
    <span>Deployment built successfuly</span>
  </div>
  <pre>{{ buildLogs }}</pre>
</template>
