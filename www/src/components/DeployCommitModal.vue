<script lang="ts" setup>
import { ref } from "vue";

const props = defineProps(["appId"]);

const branch = ref<string>("");
const commit = ref<string>("");

const onSubmit = () => {
  if (!branch.value || !commit.value) {
    alert("Please fill out all required fields");
    return;
  }

  fetch(`/runner/api/app/${props.appId}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      branch: branch.value,
      commit: commit.value,
    }),
  });
};
</script>

<template>
  <div id="deployCommitModal" class="modal" tabindex="-1">
    <div class="modal-dialog">
      <div class="modal-content">
        <div class="modal-header">
          <h5 class="modal-title">Add new App</h5>
          <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
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
          <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">
            Close
          </button>
          <button @click="onSubmit" type="button" class="btn btn-primary" data-bs-dismiss="modal">
            Deploy
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
