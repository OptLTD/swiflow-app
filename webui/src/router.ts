import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'chat', component: () => import('./views/ChatView.vue') },
    { path: '/agents', name: 'agents', component: () => import('./views/AgentsView.vue') },
    { path: '/providers', name: 'providers', component: () => import('./views/ProvidersView.vue') },
    { path: '/skills', name: 'skills', component: () => import('./views/SkillsView.vue') },
    { path: '/settings', name: 'settings', component: () => import('./views/SettingsView.vue') },
  ],
})
