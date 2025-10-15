<script setup lang="ts">
import { computed } from 'vue'
import { Tippy } from 'vue-tippy'
import { useAppStore } from '@/stores/app'
import { request, toast } from '@/support';

const app = useAppStore()
const isLogin = computed(() => {
  return !!app.getLogin?.email
})

// 获取用户信息
const userProfile = computed(() => {
  const login = app.getLogin
  return {
    email: login.email || '',
    avatar: login.avatar || '',
    username: login.username,
    userPlan: login.userPlan,
    expireAt: login.expireAt,
  }
})

// 格式化过期时间
const formatExpireTime = computed(() => {
  if (!userProfile.value.expireAt) return '永久有效'
  const date = new Date(userProfile.value.expireAt)
  return date.toLocaleDateString('zh-CN')
})

// 获取头像显示内容
const avatarDisplay = computed(() => {
  if (userProfile.value.avatar) {
    return userProfile.value.avatar
  }
  const name = userProfile.value.username
  if (!name) {
    return 'M'
  }
  return name.charAt(0).toUpperCase()
})

// 处理退出登录
const handleLogout = async () => {
  try {
    const url = '/sign-out?act=success'
    const resp = await request.post<any>(url)
    if (resp && resp.errmsg) {
      throw resp.errmsg
    }
    app.setRefresh(true)
    toast.success('Logout Success!')
  } catch (err) {
    toast.error(err)
    console.log(err)
  }
}

// 处理登录
const handleLogin = () => {
  if (!app.authGateway) {
    console.warn('认证中心地址未配置')
    toast.error('认证中心地址未配置')
    return
  }
  
  const uri = 'authorization?from=swiflow-app'
  const loginLink = document.getElementById('loginUrl')
  loginLink?.setAttribute('href', `${app.authGateway}/${uri}`)
  return loginLink && loginLink.click && loginLink.click()
}

// 处理头像点击
const handleAvatarClick = () => {
  if (isLogin.value) {
    // 已登录，跳转到profile页面
    if (app.authGateway) {
      const profileLink = document.getElementById('profileUrl')
      profileLink?.setAttribute('href', `${app.authGateway}/profile`)
      return profileLink && profileLink.click && profileLink.click()
    }
  } else {
    // 未登录，执行登录
    handleLogin()
  }
}
</script>

<template>
  <tippy interactive :theme="app.getTheme" arrow
    placement="left-start" trigger="mouseenter click">
    <icon size="large" icon="icon-person"/>
    <template #content>
      <div class="profile-menu">
        <!-- 隐藏的链接元素 -->
        <a id="loginUrl" href="#" target="_blank" style="display: none;"></a>
        <a id="profileUrl" href="#" target="_blank" style="display: none;"></a>
        
        <!-- 头部用户信息 -->
        <div class="profile-header">
          <div class="profile-avatar" @click="handleAvatarClick">
            <img v-if="userProfile.avatar" 
              :src="userProfile.avatar" 
              :alt="userProfile.username" 
            />
            <span v-else>{{ avatarDisplay }}</span>
          </div>
          <div class="profile-info" @click="handleAvatarClick">
            <div class="username">{{ isLogin ? userProfile.username : $t('common.pleaseLogin') }}</div>
            <div class="email">{{ isLogin ? userProfile.email : $t('common.loginRecommend') }}</div>
          </div>
        </div>
        
        <!-- 已登录用户的详细信息 -->
        <template v-if="isLogin">
          <!-- 分割线 -->
          <div class="profile-divider"></div>
          
          <!-- 用户计划和过期时间 -->
          <div class="profile-details">
            <div class="detail-item">
              <span class="detail-label">用户计划:</span>
              <span class="detail-value plan-badge">
                {{ userProfile.userPlan }}
              </span>
            </div>
            <div class="detail-item">
              <span class="detail-label">过期时间:</span>
              <span class="detail-value">
                {{ formatExpireTime }}
              </span>
            </div>
          </div>
        </template>
        
        <!-- 分割线 -->
        <div class="profile-divider"></div>
        
        <!-- 菜单选项 -->
        <div class="profile-actions">
          <template v-if="isLogin">
            <div class="action-item" @click="handleLogout">
              <span class="action-icon">🚪</span>
              <span class="action-text">{{ $t('common.logout') }}</span>
            </div>
          </template>
          <template v-else>
            <div class="action-item" @click="handleLogin">
              <span class="action-icon">🔑</span>
              <span class="action-text">{{ $t('common.login') }}</span>
            </div>
          </template>
        </div>
      </div>
    </template>
  </tippy>
</template>

<style scoped>
.profile-menu {
  padding: 12px 8px;
  min-width: 160px;
  overflow: hidden;
}

.profile-header {
  display: flex;
  align-items: center;
}

.profile-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-secondary) 100%); 
  color: var(--bg-main);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: bold;
  margin-right: 10px;
  flex-shrink: 0;
  border: 2px solid var(--color-divider);
  position: relative;
  overflow: hidden;
}

.profile-avatar img {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
  z-index: 1;
}

.profile-info {
  flex: 1;
  min-width: 0;
}

.profile-header:hover {
  cursor: pointer;
  border-radius: 3px;
  background-color: var(--bg-menu);
  box-shadow: 0px 0px 3px 6px var(--bg-menu);
}

.username {
  font-weight: 600;
  font-size: 16px;
  margin-bottom: 4px;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.email {
  font-size: 13px;
  color: var(--color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.profile-divider {
  height: 1px;
  margin: 12px 0;
  background: var(--color-divider);
}

.detail-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.detail-item:last-child {
  margin-bottom: 0;
}

.detail-label {
  font-size: 13px;
  color: var(--color-secondary);
  font-weight: 500;
}

.detail-value {
  font-size: 13px;
  color: var(--text-main);
  font-weight: 600;
}

.plan-badge {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}


.action-item {
  display: flex;
  padding: 6px 0;
  cursor: pointer;
  align-items: center;
  color: var(--text-main);
}

.action-item:hover {
  border-radius: 3px;
  background-color: var(--bg-menu);
  box-shadow: 0px 0px 3px 6px var(--bg-menu);
}

.action-icon {
  margin-right: 10px;
  font-size: 16px;
  width: 20px;
  text-align: center;
  color: var(--icon-color);
}

.action-text {
  font-size: 14px;
  font-weight: 500;
}

/* 深色主题适配已通过CSS变量自动处理 */
</style>