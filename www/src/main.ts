import { createApp } from "vue";

import * as VueRouter from "vue-router";

import "bootstrap/dist/css/bootstrap.min.css";
import "bootstrap";

import "./style.css";
import App from "./App.vue";
import Home from "./pages/Home.vue";
import Logs from "./pages/Logs.vue";

const routes = [
  {
    path: "/",
    component: Home,
  },
  {
    path: "/deployment/:id/logs/:logType",
    component: Logs,
  },
];

const router = VueRouter.createRouter({
  history: VueRouter.createWebHashHistory(),
  routes,
});

createApp(App).use(router).mount("#app");
