<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { Modal } from "bootstrap";

const appId = ref<string>("");
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
    await fetch(`/runner/api/app/${appId.value}/deploy`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        branch: branch.value,
        commit: commit.value,
      }),
    });
    loading.value = false;
  } catch (err) {
    alert(err);
    console.log(err);
  }

  hide();
};
</script>

<template>
  <div ref="modalRef" class="modal" tabindex="-1">
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
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="hide">
            Close
          </button>
          <button v-if="!loading" @click="onSubmit" type="button" class="btn btn-primary">
            Deploy
          </button>
          <button v-else type="button" class="btn btn-primary" disabled>Deploying...</button>
        </div>
      </div>
    </div>
  </div>
</template>
