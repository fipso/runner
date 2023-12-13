<script lang="ts" setup>
import { useRoute, useRouter } from "vue-router";
import { ref } from "vue";
import DeploymentLog from "../components/DeploymentLog.vue";

const route = useRoute();
const deploymentId = route.params.id as string;
const initalLogType = route.query.logType as string;

const logType = ref(initalLogType || "build");
</script>

<template>
  <main class="container p-5">
    <RouterLink to="/">Back</RouterLink>

    <h1>
      <select v-model="logType">
        <option value="build">Build</option>
        <option value="running">Runtime</option>
        <option value="requests">Request</option>
      </select>
      Logs
    </h1>
    <DeploymentLog v-if="deploymentId" :deploymentId="deploymentId" :logType="logType" />
  </main>
</template>
