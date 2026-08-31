import { useRuntimeConfig } from '#app'
import { useUserStore } from '~/stores/user'

export function useApiFetch<T = any>(
  url: string,
  options: any = {}
): Promise<T> {
  const config = useRuntimeConfig()
  const userStore = useUserStore()
  const baseURL = process.server
    ? String(config.public.apiServer)
    : String(config.public.apiBase)

  // 自动带上 token
  const headers = {
    ...(options.headers || {}),
    ...(userStore.authHeaders || {})
  }

  return $fetch<T>(url, {
    baseURL,
    ...options,
    headers,
    onResponse({ response }) {
      // console.log('API响应:', {
      //   status: response.status,
      //   data: response._data,
      //   url: url
      // })
      
      // 处理401认证错误
      if (response.status === 401 ||
        (response._data && (response._data.code === 401 || response._data.error === '无效的令牌'))
      ) {
        userStore.logout()
        if (process.client) {
          window.location.href = '/login'
        }
        // 触发 onResponseError 逻辑
        throw Object.assign(new Error('登录已过期，请重新登录'), {
          data: response._data,
          status: response.status,
        })
      }

      // 处理403权限错误
      if (response.status === 403 ||
        (response._data && (response._data.code === 403 || response._data.error === '需要管理员权限'))
      ) {
        throw Object.assign(new Error('需要管理员权限，请使用管理员账号登录'), {
          data: response._data,
          status: response.status,
        })
      }

      // 统一处理 code/message
      if (response._data && response._data.code && response._data.code !== 200) {
        console.error('API错误响应:', response._data)
        throw new Error(response._data.message || '请求失败')
      }
    },
    onResponseError({ response }: { response: any }) {
      const data = response?._data
      const status = response?.status

      // 检查是否为"无效的令牌"错误
      if (data?.error === '无效的令牌' || status === 401) {
        // 清除用户状态
        userStore.logout()
        // 跳转到登录页面
        if (process.client) {
          window.location.href = '/login'
        }
        throw new Error('登录已过期，请重新登录')
      }
      
      // 检查是否为权限错误
      if (data?.error === '需要管理员权限' || status === 403) {
        throw new Error('需要管理员权限，请使用管理员账号登录')
      }

      // ofetch 的 onResponseError 上下文没有 error 字段。始终抛出标准
      // Error，避免 Nuxt useAsyncData 收到 undefined 后再次包装时报错。
      const responseMessage = typeof data === 'object' && data
        ? data.message || data.error
        : typeof data === 'string'
          ? data.trim()
          : ''
      const message = responseMessage || response?.statusText || `请求失败（HTTP ${status || '未知'}）`
      throw Object.assign(new Error(message), { data, status })
    },
    onRequestError({ error }: { error: any }) {
      throw error instanceof Error ? error : new Error('网络连接失败，请检查后端服务')
    }
  })
}
