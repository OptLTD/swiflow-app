import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'chat', component: () => import('./views/ChatView.vue') },
    { path: '/agents', name: 'agents', component: () => import('./views/AgentsView.vue') },
    { path: '/providers', name: 'providers', component: () => import('./views/ProvidersView.vue') },
    { path: '/skills', name: 'skills', component: () => import('./views/SkillsView.vue') },
    { path: '/tools', name: 'tools', component: () => import('./views/ToolsView.vue') },
    { path: '/mcp', name: 'mcp', component: () => import('./views/MCPServersView.vue') },
    { path: '/cron', name: 'cron', component: () => import('./views/CronView.vue') },
    { path: '/settings', name: 'settings', component: () => import('./views/SettingsView.vue') },
  ],
})
