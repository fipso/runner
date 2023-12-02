<script lang="ts" setup>
import { onMounted, ref } from "vue";
import { Modal } from "bootstrap";

const appName = ref<string>("");
const appTemplate = ref<string>("");
const appGitUrl = ref<string>("");
const appGitUsername = ref<string>("");
const appGitPassword = ref<string>("");
const appEnv = ref<string>("");

const loading = ref(false);
const modalRef = ref<HTMLElement | null>();
const modal = ref<Modal | null>();

onMounted(() => {
  modal.value = new Modal(modalRef.value as Element, {});
});

const show = () => {
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
  if (!appName.value || !appTemplate.value || !appGitUrl.value) {
    alert("Please fill out all required fields");
    return;
  }

  try {
    loading.value = true;
    await fetch("/runner/api/app", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        name: appName.value,
        template_id: appTemplate.value,
        git_url: appGitUrl.value,
        git_username: appGitUsername.value,
        git_password: appGitPassword.value,
        env: appEnv.value,
      }),
    });

    emit("success");

    hide();
  } catch (err) {
    console.log(err);
    alert(err);
  }

  loading.value = false;
};
</script>

<template>
  <div ref="modalRef" class="modal" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">Add new App</h5>
          <button type="button" class="btn-close" aria-label="Close" @click="hide"></button>
        </div>
        <div class="modal-body">
          <form>
            <div class="mb-3">
              <label for="appName" class="form-label">App Name*</label>
              <input type="text" class="form-control" id="appName" v-model="appName" />
            </div>
            <div class="mb-3">
              <label for="appTemplate" class="form-label">App Template*</label>
              <select v-model="appTemplate" class="form-select" id="appTemplate">
                <option value="nextjs" selected>NextJS</option>
                <option value="vite" disabled>Vite</option>
                <option value="react" disabled>React</option>
                <option value="static" disabled>Static</option>
              </select>
            </div>
            <div class="mb-3">
              <label for="appGitUrl" class="form-label">Git URL*</label>
              <input type="text" class="form-control" id="appGitUrl" v-model="appGitUrl" />
            </div>
            <div class="mb-3">
              <label for="appGitUsername" class="form-label">Git Username</label>
              <input type="text" class="form-control" id="appGitUsername" v-model="appGitUsername" />
            </div>
            <div class="mb-3">
              <label for="appGitPassword" class="form-label">Git Password</label>
              <input type="password" class="form-control" id="appGitPassword" v-model="appGitPassword" />
            </div>
            <div class="mb-3">
              <label for="appEnv" class="form-label">Environment Variables</label>
              <textarea v-model="appEnv" class="form-control" id="appEnv"></textarea>
            </div>
          </form>
        </div>
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="hide">
            Close
          </button>
          <button v-if="!loading" @click="onSubmit" type="button" class="btn btn-primary">
            Add App
          </button>
          <button v-else type="button" class="btn btn-primary" disabled>
            Creating...
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
