<script lang="ts" setup>
import { useRoute } from "vue-router";
import DeploymentLog from "../components/DeploymentLog.vue";

const route = useRoute();
const deploymentId = route.params.id as string;
const logType = route.params.logType as string;
</script>

<template>
  <main class="container p-5">
    <h1>Logs</h1>
    <DeploymentLog v-if="deploymentId" :deploymentId="deploymentId" :logType="logType" />
  </main>
</template>
