<template>
  <section
    ref="sectionRef"
    aria-labelledby="melost-search-title"
    class="mt-5 sm:mt-8"
  >
    <div class="mb-3 flex flex-col gap-1 px-1 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h2 id="melost-search-title" class="text-base font-semibold text-gray-900 dark:text-slate-100">
          搜索结果
        </h2>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-slate-400">
          由 melost.cn 提供
          <template v-if="!loading && !errorMessage"> · {{ total.toLocaleString() }} 条结果 · {{ took }} ms</template>
        </p>
      </div>
    </div>

    <div class="mb-3 flex flex-wrap gap-1.5" role="group" aria-label="站外资源平台筛选">
      <button
        v-for="filter in platformFilters"
        :key="filter.value"
        type="button"
        class="min-h-11 rounded border px-3 text-xs font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-slate-900"
        :class="selectedType === filter.value
          ? 'border-slate-800 bg-slate-800 text-white dark:border-slate-200 dark:bg-slate-200 dark:text-slate-900'
          : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700'"
        :aria-pressed="selectedType === filter.value"
        @click="changeType(filter.value)"
      >
        {{ filter.label }}
      </button>
    </div>

    <div class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm dark:border-slate-700 dark:bg-slate-800">
      <div v-if="loading" class="divide-y divide-gray-100 dark:divide-slate-700" aria-live="polite" aria-busy="true">
        <div v-for="index in 4" :key="index" class="p-4 sm:p-5">
          <div class="animate-pulse space-y-3">
            <div class="h-4 w-2/3 rounded bg-gray-200 dark:bg-slate-700"></div>
            <div class="h-3 w-full rounded bg-gray-100 dark:bg-slate-700/60"></div>
            <div class="h-3 w-24 rounded bg-gray-100 dark:bg-slate-700/60"></div>
          </div>
        </div>
        <span class="sr-only">正在搜索站外资源</span>
      </div>

      <div v-else-if="errorMessage" class="px-5 py-10 text-center" role="alert">
        <i class="fas fa-circle-exclamation text-xl text-red-500" aria-hidden="true"></i>
        <p class="mt-2 text-sm font-medium text-gray-800 dark:text-slate-100">站外搜索暂时不可用</p>
        <p class="mt-1 text-xs text-gray-500 dark:text-slate-400">{{ errorMessage }}</p>
        <n-button class="mt-4 min-h-11" secondary @click="loadResults">
          <template #icon><i class="fas fa-rotate-right" aria-hidden="true"></i></template>
          重试
        </n-button>
      </div>

      <div v-else-if="items.length === 0" class="px-5 py-10 text-center text-sm text-gray-500 dark:text-slate-400">
        <i class="fas fa-magnifying-glass text-xl text-gray-400" aria-hidden="true"></i>
        <p class="mt-2">站外也没有找到“{{ query }}”相关资源</p>
        <p class="mt-1 text-xs">可以缩短关键词后再次搜索</p>
      </div>

      <div v-else class="divide-y divide-gray-100 dark:divide-slate-700">
        <button
          v-for="item in items"
          :key="itemKey(item)"
          type="button"
          class="group block min-h-[112px] w-full p-4 text-left transition-colors focus:outline-none focus:ring-2 focus:ring-inset focus:ring-blue-500 sm:p-5"
          :class="item.can_stage
            ? 'hover:bg-gray-50 disabled:cursor-wait disabled:bg-gray-50 dark:hover:bg-slate-700/50 dark:disabled:bg-slate-700/40'
            : 'cursor-not-allowed opacity-60'"
          :disabled="!item.can_stage || stateFor(item).status === 'opening'"
          :aria-label="`打开${item.disk_name}的资源详情`"
          @click="stageAndOpen(item)"
        >
          <span class="flex items-start gap-3 sm:gap-4">
            <span class="min-w-0 flex-1">
              <span class="flex flex-wrap items-center gap-2">
                <span class="inline-flex items-center gap-1 rounded border border-gray-200 bg-gray-50 px-1.5 py-0.5 text-xs font-medium text-gray-700 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-200">
                  <i class="fas fa-cloud" aria-hidden="true"></i>
                  {{ platformName(item.disk_type) }}
                </span>
                <span v-if="item.shared_time" class="text-xs text-gray-400 dark:text-slate-500">{{ item.shared_time }}</span>
              </span>
              <span class="mt-2 block break-words text-sm font-semibold leading-6 text-gray-900 dark:text-slate-100 sm:text-base">
                {{ item.disk_name }}
              </span>
              <span v-if="item.files" class="mt-1 line-clamp-2 block break-words text-xs leading-5 text-gray-500 dark:text-slate-400">
                {{ item.files }}
              </span>
              <span v-if="item.share_user" class="mt-2 block text-xs text-gray-400 dark:text-slate-500">
                分享者：{{ item.share_user }}
              </span>
              <span v-if="stateFor(item).status === 'failed'" class="mt-2 block break-words text-xs leading-5 text-red-600 dark:text-red-400" role="alert">
                {{ stateFor(item).error }}
              </span>
              <span v-else-if="!item.can_stage" class="mt-2 block text-xs leading-5 text-amber-700 dark:text-amber-400">
                {{ item.stage_message || '该资源类型暂不支持' }}
              </span>
            </span>

            <span class="flex h-11 w-11 shrink-0 items-center justify-center text-gray-400 dark:text-slate-500">
              <i
                class="fas text-sm"
                :class="stateFor(item).status === 'opening'
                  ? 'fa-spinner fa-spin text-blue-600 dark:text-blue-400'
                  : 'fa-chevron-right transition-transform group-hover:translate-x-0.5'"
                aria-hidden="true"
              ></i>
              <span v-if="stateFor(item).status === 'opening'" class="sr-only">正在打开详情</span>
            </span>
          </span>
        </button>
      </div>

      <div
        v-if="!loading && !errorMessage && items.length > 0 && totalPages > 1"
        class="flex items-center justify-between border-t border-gray-200 px-4 py-3 dark:border-slate-700"
        aria-label="站外搜索分页"
      >
        <n-button class="min-h-11" secondary :disabled="page <= 1" @click="changePage(page - 1)">
          <template #icon><i class="fas fa-chevron-left" aria-hidden="true"></i></template>
          上一页
        </n-button>
        <span class="text-xs text-gray-500 dark:text-slate-400">第 {{ page }} / {{ totalPages }} 页</span>
        <n-button class="min-h-11" secondary :disabled="page >= totalPages" @click="changePage(page + 1)">
          下一页
          <template #icon><i class="fas fa-chevron-right" aria-hidden="true"></i></template>
        </n-button>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { MelostSearchItem } from '~/composables/useMelostApi'
import { useMelostApi } from '~/composables/useMelostApi'

const props = defineProps<{
  query: string
}>()

type LocalStageState = {
  status: 'idle' | 'opening' | 'failed'
  error?: string
}

const api = useMelostApi()
const router = useRouter()
const sectionRef = ref<HTMLElement | null>(null)
const items = ref<MelostSearchItem[]>([])
const total = ref(0)
const took = ref(0)
const page = ref(1)
const selectedType = ref('')
const pageSize = 20
const loading = ref(true)
const errorMessage = ref('')
const stageStates = reactive<Record<string, LocalStageState>>({})
let searchSequence = 0
let disposed = false

const totalPages = computed(() => Math.max(1, Math.min(500, Math.ceil(total.value / pageSize))))
const platformFilters = [
  { value: '', label: '全部' },
  { value: 'ALY', label: '阿里云盘' },
  { value: 'QUARK', label: '夸克网盘' },
  { value: 'BDY', label: '百度网盘' },
  { value: 'XUNLEI', label: '迅雷网盘' },
  { value: 'UC', label: 'UC网盘' }
]

const itemKey = (item: MelostSearchItem) => item.doc_id || item.link
const stateFor = (item: MelostSearchItem): LocalStageState => stageStates[itemKey(item)] || { status: 'idle' }

const getErrorMessage = (error: any, fallback: string) => {
  return error?.data?.message || error?.data?.error || error?.message || fallback
}

const loadResults = async () => {
  const sequence = ++searchSequence
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await api.search(props.query, page.value, pageSize, selectedType.value)
    if (sequence !== searchSequence || disposed) return
    items.value = response.items || []
    total.value = response.total || 0
    took.value = response.took || 0
  } catch (error: any) {
    if (sequence !== searchSequence || disposed) return
    items.value = []
    total.value = 0
    errorMessage.value = getErrorMessage(error, '请稍后重试')
  } finally {
    if (sequence === searchSequence && !disposed) loading.value = false
  }
}

const stageAndOpen = async (item: MelostSearchItem) => {
  if (!item.can_stage || stateFor(item).status === 'opening') return

  stageStates[itemKey(item)] = { status: 'opening' }
  try {
    const result = await api.stageResource(item)
    if (!result.resource_key) throw new Error('资源详情地址生成失败')
    await router.push(`/r/${result.resource_key}`)
  } catch (error: any) {
    stageStates[itemKey(item)] = {
      status: 'failed',
      error: getErrorMessage(error, '保存资源失败，请重试')
    }
  }
}

const changePage = async (nextPage: number) => {
  if (nextPage < 1 || nextPage > totalPages.value || nextPage === page.value) return
  page.value = nextPage
  await loadResults()
  sectionRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const changeType = async (type: string) => {
  if (selectedType.value === type) return
  selectedType.value = type
  page.value = 1
  await loadResults()
}

const platformName = (type: string) => {
  const names: Record<string, string> = {
    BDY: '百度网盘',
    ALY: '阿里云盘',
    QUARK: '夸克网盘',
    XUNLEI: '迅雷网盘',
    UC: 'UC网盘',
    '115': '115网盘',
    LZY: '蓝奏云',
    MAGNET: '磁力链接',
    ED2K: '电驴链接'
  }
  return names[type?.toUpperCase()] || type || '未知平台'
}

onMounted(loadResults)

watch(() => props.query, async () => {
  page.value = 1
  await loadResults()
})

onBeforeUnmount(() => {
  disposed = true
  searchSequence++
})
</script>
