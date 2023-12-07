<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { Modal } from "bootstrap";

const appId = ref<string>("");
const appEnv = ref<string>("");

const modalRef = ref<HTMLElement | null>();
const modal = ref<Modal | null>();

onMounted(() => {
  modal.value = new Modal(modalRef.value as Element, {});
});

const show = (id: string, currentEnv: string) => {
  appId.value = id;
  appEnv.value = currentEnv;
  modal.value?.show();
};
defineExpose({
  show,
});

const emit = defineEmits(["success"]);

const hide = () => {
  modal.value?.hide();
};

const onSubmit = async () => {
  if (!appEnv.value) {
    alert("Please fill out all required fields");
    return;
  }

  try {
    await fetch(`/runner/api/app/${appId.value}/env`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        env: appEnv.value,
      }),
    });

    emit("success");

    hide();
  } catch (err) {
    console.log(err);
    alert(err);
  }
};
</script>

<template>
  <div ref="modalRef" class="modal modal-lg" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">Add new App</h5>
          <button type="button" class="btn-close" aria-label="Close" @click="hide"></button>
        </div>
        <div class="modal-body">
          <form>
            <div class="mb-3">
              <label for="appEnv" class="form-label">Environment Variables</label>
              <textarea v-model="appEnv" class="form-control" id="appEnv" :rows="appEnv.split('\n').length + 1"
                placeholder="NODE_ENV=production"></textarea>
            </div>
          </form>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="hide">
            Close
          </button>
          <button @click="onSubmit" type="button" class="btn btn-primary">
            Update Env
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
