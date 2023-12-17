import { createApp } from "vue";

import * as VueRouter from "vue-router";

import "bootstrap/dist/css/bootstrap.min.css";
import "bootstrap";

import "bootstrap-icons/font/bootstrap-icons.css";

import "./style.css";
import App from "./App.vue";
import Dashboard from "./pages/Dashboard.vue";
import AppPage from "./pages/App.vue";
import Logs from "./pages/Logs.vue";

const routes = [
  {
    path: "/",
    component: Dashboard,
  },
  {
    path: "/app/:id",
    component: AppPage,
  },
  {
    path: "/deployment/:id/logs",
    component: Logs,
  },
];

const router = VueRouter.createRouter({
  history: VueRouter.createWebHistory("/runner"),
  routes,
});

createApp(App).use(router).mount("#app");
