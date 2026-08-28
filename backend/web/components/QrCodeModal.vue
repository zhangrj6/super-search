<template>
  <n-modal
    :show="visible"
    preset="card"
    title="资源链接"
    class="max-w-sm"
    @update:show="closeModal"
  >
    <div class="text-center">
      <div v-if="loading" class="flex flex-col items-center justify-center py-8" aria-live="polite">
        <n-spin size="large" />
        <p class="mt-4 text-sm text-gray-600 dark:text-gray-400">正在获取链接...</p>
      </div>

      <div v-else-if="forbidden" class="flex flex-col items-center justify-center py-4">
        <img src="/assets/svg/forbidden.svg" alt="禁止访问" class="mb-6 h-48 w-48" />
        <h3 class="mb-2 text-xl font-bold text-red-600 dark:text-red-400">禁止访问</h3>
        <p class="mb-4 text-gray-600 dark:text-gray-400">该资源包含违禁内容，无法访问</p>
        <n-button type="error" @click="closeModal">我知道了</n-button>
      </div>

      <div v-else-if="error" class="space-y-4" role="alert">
        <n-alert type="error" :show-icon="false">
          <template #icon>
            <i class="fas fa-exclamation-triangle mr-2 text-red-500" aria-hidden="true"></i>
          </template>
          {{ error }}
        </n-alert>
        <n-button class="min-h-11 w-full" secondary @click="closeModal">关闭</n-button>
      </div>

      <div v-else-if="deliveryUrl" class="space-y-4">
        <n-alert v-if="message" type="success" :show-icon="false">
          {{ message }}
        </n-alert>

        <div class="flex justify-center">
          <div class="qr-container flex items-center justify-center">
            <QRCodeDisplay
              v-if="qrCodePreset"
              :data="deliveryUrl"
              :preset="qrCodePreset"
              :width="size"
              :height="size"
            />
            <QRCodeDisplay v-else :data="deliveryUrl" :width="size" :height="size" />
          </div>
        </div>

        <div>
          <p class="mb-2 text-sm text-gray-600 dark:text-gray-400">
            {{ isQuarkLink ? '新的夸克分享链接' : '分享链接' }}
          </p>
          <n-card size="small">
            <p class="break-all text-xs text-gray-700 dark:text-gray-300">{{ deliveryUrl }}</p>
          </n-card>
        </div>

        <div class="grid grid-cols-2 gap-2">
          <n-button type="primary" class="min-h-11" @click="openLink">
            <template #icon><i class="fas fa-external-link-alt" aria-hidden="true"></i></template>
            打开
          </n-button>
          <n-button type="success" class="min-h-11" @click="copyUrl">
            <template #icon><i :class="copied ? 'fas fa-check' : 'fas fa-copy'" aria-hidden="true"></i></template>
            {{ copied ? '已复制' : '复制' }}
          </n-button>
        </div>

        <n-button class="min-h-11 w-full" secondary @click="downloadQrCode">
          <template #icon><i class="fas fa-download" aria-hidden="true"></i></template>
          下载二维码
        </n-button>
      </div>

      <div v-else class="space-y-4" role="alert">
        <n-alert type="error" :show-icon="false">没有可用的分享链接</n-alert>
        <n-button class="min-h-11 w-full" secondary @click="closeModal">关闭</n-button>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { QRCodeDisplay, preloadCommonLogos } from './QRCode'
import { useSystemConfigStore } from '~/stores/systemConfig'
import { findPresetByName } from './QRCode/presets'

interface Props {
  visible: boolean
  save_url?: string
  url?: string
  loading?: boolean
  linkType?: string
  platform?: string
  message?: string
  error?: string
  forbidden?: boolean
  forbidden_words?: string[]
}

interface Emits {
  (event: 'close'): void
}

const props = withDefaults(defineProps<Props>(), {
  url: '',
  save_url: ''
})
const emit = defineEmits<Emits>()
const systemConfigStore = useSystemConfigStore()
const size = ref(180)
const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | undefined

const deliveryUrl = computed(() => props.save_url || props.url)
const qrCodePreset = computed(() => {
  const styleName = systemConfigStore.config?.qr_code_style || 'Plain'
  return findPresetByName(styleName)
})
const isQuarkLink = computed(() => deliveryUrl.value.includes('pan.quark.cn') || deliveryUrl.value.includes('quark.cn'))

const closeModal = () => emit('close')

const copyUrl = async () => {
  if (!deliveryUrl.value) return
  try {
    await navigator.clipboard.writeText(deliveryUrl.value)
    copied.value = true
    if (copiedTimer) clearTimeout(copiedTimer)
    copiedTimer = setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (error) {
    console.error('复制失败:', error)
  }
}

const openLink = () => {
  if (process.client && deliveryUrl.value) {
    window.open(deliveryUrl.value, '_blank', 'noopener,noreferrer')
  }
}

const downloadQrCode = () => {
  const qrElement = document.querySelector('.n-qr-code canvas') as HTMLCanvasElement | null
  if (!qrElement) return
  try {
    const link = document.createElement('a')
    link.download = 'qrcode.png'
    link.href = qrElement.toDataURL()
    link.click()
  } catch (error) {
    console.error('下载失败:', error)
  }
}

onMounted(async () => {
  try {
    await preloadCommonLogos()
  } catch (error) {
    console.warn('Failed to preload common logos:', error)
  }
})

watch(() => props.visible, (visible) => {
  if (visible) copied.value = false
})
</script>

<style scoped>
.qr-container {
  width: 200px;
  height: 200px;
  background-color: #f5f5f5;
}

.n-qr-code {
  padding: 0 !important;
}
</style>
