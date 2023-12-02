<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { Modal } from "bootstrap";

import DeploymentLog from "./DeploymentLog.vue";

const appId = ref<string>("");
const deploymentId = ref<string>("");
const branch = ref<string>("");
const commit = ref<string>("");

const loading = ref(false);
const modalRef = ref<HTMLElement | null>();
const modal = ref<Modal | null>();

onMounted(() => {
  modal.value = new Modal(modalRef.value as Element, {});
});

const show = (id: string) => {
  appId.value = id;
  modal.value?.show();
};
defineExpose({
  show,
});

const hide = () => {
  modal.value?.hide();
};

const onSubmit = async () => {
  if (!branch.value || !commit.value) {
    alert("Please fill out all required fields");
    return;
  }

  try {
    loading.value = true;
    const res = await fetch(`/runner/api/app/${appId.value}/deploy`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        branch: branch.value,
        commit: commit.value,
      }),
    });

    const data = await res.json();
    deploymentId.value = data.id;
  } catch (err) {
    alert(err);
    console.log(err);
  }
};
</script>

<template>
  <div ref="modalRef" class="modal modal-xl" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">Deploy commit</h5>
          <button type="button" class="btn-close" @click="hide"></button>
        </div>
        <div class="modal-body">
          <form>
            <div class="mb-3">
              <label for="branch" class="form-label">Branch*</label>
              <input type="text" class="form-control" id="branch" v-model="branch" />
            </div>
            <div class="mb-3">
              <label for="commit" class="form-label">Commit*</label>
              <input type="text" class="form-control" id="commit" v-model="commit" />
            </div>
          </form>

          <h6>Build Logs</h6>
          <DeploymentLog v-if="deploymentId" :deploymentId="deploymentId" logType="build" @buildDone="loading = false" />
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="hide">
            Close
          </button>
          <button v-if="!loading" @click="onSubmit" type="button" class="btn btn-primary">
            Deploy
          </button>
          <button v-else type="button" class="btn btn-primary" disabled>
            Deploying...
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
