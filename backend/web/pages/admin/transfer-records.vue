<template>
  <AdminPageLayout>
    <template #page-header>
      <div class="min-w-0">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">转存链路记录</h1>
        <p class="text-sm text-gray-600 dark:text-gray-400">转存、分享与云盘文件清理审计</p>
      </div>
      <n-button type="primary" :loading="loading" @click="refreshData">
        <template #icon><i class="fas fa-refresh" aria-hidden="true"></i></template>
        刷新
      </n-button>
    </template>

    <template #notice-section>
      <div class="space-y-3">
        <n-alert type="info" title="固定清理策略" :bordered="false">
          云盘文件在最后一次成功转存或分享 10 分钟后删除，每分钟扫描一次；链路记录永久保留。
        </n-alert>
        <div class="summary-grid" :aria-busy="summaryLoading">
          <div v-for="item in summaryItems" :key="item.key" class="summary-item">
            <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
              <i :class="item.icon" aria-hidden="true"></i>
              <span>{{ item.label }}</span>
            </div>
            <strong :class="item.valueClass">{{ formatNumber(item.value) }}</strong>
          </div>
        </div>
      </div>
    </template>

    <template #filter-bar>
      <div class="rounded-md border border-gray-200 bg-white p-4 dark:border-gray-700 dark:bg-gray-800">
        <n-button class="mb-3 w-full md:hidden" secondary @click="mobileFiltersExpanded = !mobileFiltersExpanded">
          <template #icon><i class="fas fa-filter" aria-hidden="true"></i></template>
          {{ mobileFiltersExpanded ? '收起筛选条件' : '展开筛选条件' }}
          <i :class="[mobileFiltersExpanded ? 'fas fa-chevron-up' : 'fas fa-chevron-down', 'ml-2']" aria-hidden="true"></i>
        </n-button>
        <div class="filter-grid" :class="{ 'filter-grid-collapsed': !mobileFiltersExpanded }">
          <div class="filter-field filter-keyword">
            <label for="transfer-record-query">关键词</label>
            <n-input
              id="transfer-record-query"
              v-model:value="filters.query"
              clearable
              placeholder="标题、链接、账号、Trace ID"
              @keyup.enter="applyFilters"
            >
              <template #prefix><i class="fas fa-search" aria-hidden="true"></i></template>
            </n-input>
          </div>
          <div class="filter-field">
            <label>操作类型</label>
            <n-select v-model:value="filters.operation" clearable :options="operationOptions" placeholder="全部操作" />
          </div>
          <div class="filter-field">
            <label>执行状态</label>
            <n-select v-model:value="filters.status" clearable :options="statusOptions" placeholder="全部状态" />
          </div>
          <div class="filter-field">
            <label>清理状态</label>
            <n-select v-model:value="filters.cleanupStatus" clearable :options="cleanupOptions" placeholder="全部状态" />
          </div>
          <div class="filter-field">
            <label>网盘类型</label>
            <n-select v-model:value="filters.panType" clearable :options="panOptions" placeholder="全部网盘" />
          </div>
          <div class="filter-field">
            <label>开始日期</label>
            <n-date-picker v-model:value="filters.startDate" type="date" clearable class="w-full" />
          </div>
          <div class="filter-field">
            <label>结束日期</label>
            <n-date-picker v-model:value="filters.endDate" type="date" clearable class="w-full" />
          </div>
          <div class="filter-actions">
            <n-button type="primary" @click="applyFilters">
              <template #icon><i class="fas fa-filter" aria-hidden="true"></i></template>
              筛选
            </n-button>
            <n-button secondary @click="resetFilters">
              <template #icon><i class="fas fa-undo" aria-hidden="true"></i></template>
              重置
            </n-button>
          </div>
        </div>
      </div>
    </template>

    <template #content-header>
      <div class="flex min-w-0 items-center justify-between gap-3">
        <div class="min-w-0">
          <span class="font-semibold text-gray-900 dark:text-white">链路明细</span>
          <span class="ml-2 text-sm text-gray-500 dark:text-gray-400">共 {{ formatNumber(total) }} 条</span>
        </div>
        <span class="hidden text-xs text-gray-500 dark:text-gray-400 sm:inline">记录只读，云盘文件清理不影响审计数据</span>
      </div>
    </template>

    <template #content>
      <div v-if="loading" class="flex h-full min-h-64 items-center justify-center" aria-live="polite">
        <n-spin size="large" description="正在加载链路记录" />
      </div>
      <AdminEmptyState
        v-else-if="records.length === 0"
        icon="fas fa-route"
        title="暂无链路记录"
      />
      <div v-else class="h-full overflow-y-auto">
        <div class="hidden min-w-[1120px] lg:block">
          <table class="audit-table">
            <thead>
              <tr>
                <th>操作</th>
                <th>资源</th>
                <th>网盘与账号</th>
                <th>原分享链接</th>
                <th>结果链接</th>
                <th>操作时间</th>
                <th>清理状态</th>
                <th class="w-20 text-right">详情</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="record in records" :key="record.id">
                <td>
                  <div class="space-y-1">
                    <n-tag :type="operationTagType(record.operation)" size="small">{{ operationLabel(record.operation) }}</n-tag>
                    <div class="text-xs text-gray-500 dark:text-gray-400">{{ triggerSourceLabel(record.trigger_source) }}</div>
                  </div>
                </td>
                <td>
                  <button type="button" class="title-button" @click="openDetails(record)">
                    {{ record.resource_title || `资源 #${record.resource_id || '-'}` }}
                  </button>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    #{{ record.resource_id || '-' }} · {{ record.resource_source || '系统资源' }}
                  </div>
                </td>
                <td>
                  <div class="font-medium text-gray-800 dark:text-gray-200">{{ record.pan_name || panLabel(record.pan_type) }}</div>
                  <div class="mt-1 max-w-40 truncate text-xs text-gray-500 dark:text-gray-400">
                    {{ accountLabel(record) }}
                  </div>
                </td>
                <td><LinkActions :url="record.source_url" @copy="copyText" @open="openURL" /></td>
                <td><LinkActions :url="record.result_url" @copy="copyText" @open="openURL" /></td>
                <td>
                  <div class="whitespace-nowrap text-sm text-gray-700 dark:text-gray-300">{{ formatDate(record.occurred_at) }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ record.duration_ms || 0 }} ms</div>
                </td>
                <td>
                  <n-tag :type="cleanupTagType(record.cleanup_status)" size="small">
                    {{ cleanupLabel(record.cleanup_status) }}
                  </n-tag>
                  <div class="mt-1 whitespace-nowrap text-xs text-gray-500 dark:text-gray-400">
                    {{ cleanupTimeLabel(record) }}
                  </div>
                </td>
                <td class="text-right">
                  <n-tooltip>
                    <template #trigger>
                      <n-button quaternary circle size="small" aria-label="查看链路详情" @click="openDetails(record)">
                        <template #icon><i class="fas fa-eye" aria-hidden="true"></i></template>
                      </n-button>
                    </template>
                    查看详情
                  </n-tooltip>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="space-y-3 p-3 lg:hidden">
          <article v-for="record in records" :key="record.id" class="mobile-record">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="mb-2 flex flex-wrap items-center gap-2">
                  <n-tag :type="operationTagType(record.operation)" size="small">{{ operationLabel(record.operation) }}</n-tag>
                  <n-tag :type="cleanupTagType(record.cleanup_status)" size="small">{{ cleanupLabel(record.cleanup_status) }}</n-tag>
                </div>
                <button type="button" class="title-button" @click="openDetails(record)">
                  {{ record.resource_title || `资源 #${record.resource_id || '-'}` }}
                </button>
              </div>
              <n-button quaternary circle aria-label="查看链路详情" @click="openDetails(record)">
                <template #icon><i class="fas fa-chevron-right" aria-hidden="true"></i></template>
              </n-button>
            </div>
            <dl class="mt-3 grid grid-cols-2 gap-x-3 gap-y-2 text-sm">
              <div><dt>网盘</dt><dd>{{ record.pan_name || panLabel(record.pan_type) }}</dd></div>
              <div><dt>账号</dt><dd>{{ accountLabel(record) }}</dd></div>
              <div class="col-span-2"><dt>操作时间</dt><dd>{{ formatDate(record.occurred_at) }}</dd></div>
            </dl>
            <div class="mt-3 flex items-center justify-between border-t border-gray-100 pt-3 dark:border-gray-700">
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ triggerSourceLabel(record.trigger_source) }}</span>
              <div class="flex items-center gap-2">
                <LinkActions :url="record.source_url" compact @copy="copyText" @open="openURL" />
                <LinkActions :url="record.result_url" compact @copy="copyText" @open="openURL" />
              </div>
            </div>
          </article>
        </div>
      </div>
    </template>

    <template #content-footer>
      <div class="flex justify-center p-3">
        <n-pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :item-count="total"
          :page-sizes="[20, 50, 100]"
          show-size-picker
          @update:page="fetchRecords"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </template>
  </AdminPageLayout>

  <n-drawer v-model:show="drawerOpen" :width="drawerWidth" placement="right">
    <n-drawer-content title="链路记录详情" closable>
      <div v-if="detailLoading" class="flex min-h-48 items-center justify-center"><n-spin /></div>
      <div v-else-if="selectedRecord" class="space-y-6">
        <section class="detail-section">
          <h2>操作状态</h2>
          <div class="flex flex-wrap gap-2">
            <n-tag :type="operationTagType(selectedRecord.operation)">{{ operationLabel(selectedRecord.operation) }}</n-tag>
            <n-tag :type="statusTagType(selectedRecord.status)">{{ statusLabel(selectedRecord.status) }}</n-tag>
            <n-tag :type="cleanupTagType(selectedRecord.cleanup_status)">{{ cleanupLabel(selectedRecord.cleanup_status) }}</n-tag>
          </div>
        </section>

        <section class="detail-section">
          <h2>链路标识</h2>
          <DetailRow label="记录 ID" :value="String(selectedRecord.id)" />
          <DetailRow label="Trace ID" :value="selectedRecord.trace_id" copyable @copy="copyText" />
          <DetailRow label="父记录 ID" :value="displayValue(selectedRecord.parent_id)" />
          <DetailRow label="触发源" :value="triggerSourceLabel(selectedRecord.trigger_source)" />
        </section>

        <section class="detail-section">
          <h2>资源关联</h2>
          <DetailRow label="资源标题" :value="selectedRecord.resource_title" />
          <DetailRow label="资源 ID" :value="displayValue(selectedRecord.resource_id)" />
          <DetailRow label="资源 Key" :value="selectedRecord.resource_key" copyable @copy="copyText" />
          <DetailRow label="资源源头" :value="selectedRecord.resource_source" />
          <DetailRow label="外部资源 ID" :value="selectedRecord.external_id" copyable @copy="copyText" />
          <DetailRow label="任务 ID" :value="displayValue(selectedRecord.task_id)" />
          <DetailRow label="任务项 ID" :value="displayValue(selectedRecord.task_item_id)" />
        </section>

        <section class="detail-section">
          <h2>链接链路</h2>
          <DetailRow label="原分享链接" :value="selectedRecord.source_url" link copyable @copy="copyText" @open="openURL" />
          <DetailRow label="上次分享链接" :value="selectedRecord.previous_share_url" link copyable @copy="copyText" @open="openURL" />
          <DetailRow label="本次结果链接" :value="selectedRecord.result_url" link copyable @copy="copyText" @open="openURL" />
        </section>

        <section class="detail-section">
          <h2>网盘文件</h2>
          <DetailRow label="网盘类型" :value="panLabel(selectedRecord.pan_type)" />
          <DetailRow label="网盘名称" :value="selectedRecord.pan_name" />
          <DetailRow label="账号 ID" :value="displayValue(selectedRecord.account_id)" />
          <DetailRow label="账号用户名" :value="selectedRecord.account_username" />
          <DetailRow label="账号备注" :value="selectedRecord.account_remark" />
          <DetailRow label="云盘文件 ID" :value="selectedRecord.file_id" copyable @copy="copyText" />
        </section>

        <section class="detail-section">
          <h2>时间与清理</h2>
          <DetailRow label="转存/分享时间" :value="formatDate(selectedRecord.occurred_at)" />
          <DetailRow label="应清理时间" :value="formatDate(selectedRecord.cleanup_due_at)" />
          <DetailRow label="实际清理时间" :value="formatDate(selectedRecord.cleaned_at)" />
          <DetailRow label="最后清理尝试" :value="formatDate(selectedRecord.last_cleanup_attempt_at)" />
          <DetailRow label="清理尝试次数" :value="String(selectedRecord.cleanup_attempts || 0)" />
          <DetailRow label="操作耗时" :value="`${selectedRecord.duration_ms || 0} ms`" />
          <DetailRow label="清理错误" :value="selectedRecord.cleanup_error" tone="danger" />
          <DetailRow label="操作错误" :value="selectedRecord.error_message" tone="danger" />
        </section>

        <section class="detail-section">
          <h2>扩展数据</h2>
          <n-code :code="formatMetadata(selectedRecord.metadata)" language="json" word-wrap />
        </section>
      </div>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue'
import { NButton, NTooltip, useNotification } from 'naive-ui'
import { useWindowSize } from '@vueuse/core'
import AdminPageLayout from '~/components/AdminPageLayout.vue'
import { useTransferRecordApi } from '~/composables/useApi'

definePageMeta({ layout: 'admin', middleware: ['auth'], ssr: false })

interface TransferRecord {
  id: number
  trace_id: string
  parent_id: number | null
  resource_id: number | null
  resource_key: string
  resource_source: string
  external_id: string
  task_id: number | null
  task_item_id: number | null
  operation: string
  trigger_source: string
  status: string
  source_url: string
  previous_share_url: string
  result_url: string
  resource_title: string
  pan_id: number | null
  pan_type: string
  pan_name: string
  account_id: number | null
  account_username: string
  account_remark: string
  file_id: string
  occurred_at: string
  cleanup_due_at: string | null
  cleanup_status: string
  cleanup_attempts: number
  last_cleanup_attempt_at: string | null
  cleaned_at: string | null
  cleanup_error: string
  error_message: string
  duration_ms: number
  metadata: string
}

interface TransferSummary {
  total_records: number
  today_records: number
  transfer_count: number
  share_count: number
  pending_cleanup: number
  cleaned_count: number
  cleanup_failed: number
}

const api = useTransferRecordApi()
const notification = useNotification()
const records = ref<TransferRecord[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const summaryLoading = ref(false)
const drawerOpen = ref(false)
const detailLoading = ref(false)
const mobileFiltersExpanded = ref(false)
const selectedRecord = ref<TransferRecord | null>(null)
const summary = ref<TransferSummary>({
  total_records: 0,
  today_records: 0,
  transfer_count: 0,
  share_count: 0,
  pending_cleanup: 0,
  cleaned_count: 0,
  cleanup_failed: 0,
})

const filters = reactive({
  query: '',
  operation: null as string | null,
  status: null as string | null,
  cleanupStatus: null as string | null,
  panType: null as string | null,
  startDate: null as number | null,
  endDate: null as number | null,
})

const operationOptions = [
  { label: '转存', value: 'transfer' },
  { label: '重新分享', value: 'share' },
]
const statusOptions = [
  { label: '成功', value: 'succeeded' },
  { label: '失败', value: 'failed' },
]
const cleanupOptions = [
  { label: '待清理', value: 'pending' },
  { label: '已清理', value: 'cleaned' },
  { label: '清理失败', value: 'failed' },
]
const panOptions = [
  { label: '夸克网盘', value: 'quark' },
  { label: '迅雷网盘', value: 'xunlei' },
  { label: '百度网盘', value: 'baidu' },
  { label: 'UC 网盘', value: 'uc' },
  { label: '阿里云盘', value: 'alipan' },
]

const summaryItems = computed(() => [
  { key: 'total', label: '全部记录', value: summary.value.total_records, icon: 'fas fa-list', valueClass: 'text-gray-900 dark:text-white' },
  { key: 'today', label: '今日新增', value: summary.value.today_records, icon: 'fas fa-calendar-day', valueClass: 'text-blue-600 dark:text-blue-400' },
  { key: 'transfer', label: '转存', value: summary.value.transfer_count, icon: 'fas fa-cloud-download-alt', valueClass: 'text-cyan-700 dark:text-cyan-400' },
  { key: 'share', label: '分享', value: summary.value.share_count, icon: 'fas fa-share-alt', valueClass: 'text-indigo-600 dark:text-indigo-400' },
  { key: 'pending', label: '待清理', value: summary.value.pending_cleanup, icon: 'fas fa-clock', valueClass: 'text-amber-600 dark:text-amber-400' },
  { key: 'cleaned', label: '已清理', value: summary.value.cleaned_count, icon: 'fas fa-check-circle', valueClass: 'text-green-600 dark:text-green-400' },
  { key: 'failed', label: '清理失败', value: summary.value.cleanup_failed, icon: 'fas fa-exclamation-circle', valueClass: 'text-red-600 dark:text-red-400' },
])

const { width } = useWindowSize()
const drawerWidth = computed(() => width.value < 640 ? Math.max(width.value, 320) : 620)

const LinkActions = defineComponent({
  props: { url: { type: String, default: '' }, compact: { type: Boolean, default: false } },
  emits: ['copy', 'open'],
  setup(props, { emit }) {
    return () => props.url
      ? h('div', { class: props.compact ? 'flex items-center gap-1' : 'flex max-w-48 items-center gap-1' }, [
          props.compact ? null : h('span', { class: 'min-w-0 flex-1 truncate text-xs text-gray-600 dark:text-gray-400', title: props.url }, displayURL(props.url)),
          h(NTooltip, null, {
            trigger: () => h(NButton, { quaternary: true, circle: true, size: 'small', 'aria-label': '复制链接', onClick: () => emit('copy', props.url) }, { icon: () => h('i', { class: 'fas fa-copy', 'aria-hidden': 'true' }) }),
            default: () => '复制链接',
          }),
          h(NTooltip, null, {
            trigger: () => h(NButton, { quaternary: true, circle: true, size: 'small', 'aria-label': '打开链接', onClick: () => emit('open', props.url) }, { icon: () => h('i', { class: 'fas fa-external-link-alt', 'aria-hidden': 'true' }) }),
            default: () => '打开链接',
          }),
        ])
      : h('span', { class: 'text-xs text-gray-400' }, '-')
  },
})

const DetailRow = defineComponent({
  props: {
    label: { type: String, required: true }, value: { type: String, default: '' },
    copyable: { type: Boolean, default: false }, link: { type: Boolean, default: false }, tone: { type: String, default: '' },
  },
  emits: ['copy', 'open'],
  setup(props, { emit }) {
    return () => h('div', { class: 'detail-row' }, [
      h('dt', props.label),
      h('dd', { class: props.tone === 'danger' && props.value ? 'text-red-600 dark:text-red-400' : '' }, [
        h('span', { class: 'break-all' }, props.value || '-'),
        props.copyable && props.value ? h(NButton, { quaternary: true, circle: true, size: 'tiny', 'aria-label': `复制${props.label}`, onClick: () => emit('copy', props.value) }, { icon: () => h('i', { class: 'fas fa-copy', 'aria-hidden': 'true' }) }) : null,
        props.link && props.value ? h(NButton, { quaternary: true, circle: true, size: 'tiny', 'aria-label': `打开${props.label}`, onClick: () => emit('open', props.value) }, { icon: () => h('i', { class: 'fas fa-external-link-alt', 'aria-hidden': 'true' }) }) : null,
      ]),
    ])
  },
})

function buildParams() {
  return {
    page: page.value,
    page_size: pageSize.value,
    query: filters.query.trim() || undefined,
    operation: filters.operation || undefined,
    status: filters.status || undefined,
    cleanup_status: filters.cleanupStatus || undefined,
    pan_type: filters.panType || undefined,
    start_date: toDateString(filters.startDate),
    end_date: toDateString(filters.endDate),
  }
}

async function fetchRecords() {
  loading.value = true
  try {
    const response = await api.getTransferRecords(buildParams()) as any
    records.value = Array.isArray(response?.records) ? response.records : []
    total.value = Number(response?.total || 0)
  } catch (error: any) {
    notification.error({ content: error?.message || '获取转存链路记录失败', duration: 3000 })
  } finally {
    loading.value = false
  }
}

async function fetchSummary() {
  summaryLoading.value = true
  try {
    const response = await api.getTransferRecordSummary() as TransferSummary
    summary.value = { ...summary.value, ...response }
  } catch (error: any) {
    notification.error({ content: error?.message || '获取链路统计失败', duration: 3000 })
  } finally {
    summaryLoading.value = false
  }
}

async function refreshData() {
  await Promise.all([fetchRecords(), fetchSummary()])
}

function applyFilters() {
  page.value = 1
  mobileFiltersExpanded.value = false
  fetchRecords()
}

function resetFilters() {
  filters.query = ''
  filters.operation = null
  filters.status = null
  filters.cleanupStatus = null
  filters.panType = null
  filters.startDate = null
  filters.endDate = null
  page.value = 1
  mobileFiltersExpanded.value = false
  fetchRecords()
}

function handlePageSizeChange() {
  page.value = 1
  fetchRecords()
}

async function openDetails(record: TransferRecord) {
  selectedRecord.value = record
  drawerOpen.value = true
  detailLoading.value = true
  try {
    selectedRecord.value = await api.getTransferRecord(record.id) as TransferRecord
  } catch (error: any) {
    notification.error({ content: error?.message || '获取链路详情失败', duration: 3000 })
  } finally {
    detailLoading.value = false
  }
}

async function copyText(value: string) {
  if (!value) return
  try {
    await navigator.clipboard.writeText(value)
    notification.success({ content: '已复制', duration: 1500 })
  } catch {
    notification.error({ content: '复制失败', duration: 2000 })
  }
}

function openURL(value: string) {
  if (!/^https?:\/\//i.test(value)) {
    notification.warning({ content: '链接格式无效', duration: 2000 })
    return
  }
  window.open(value, '_blank', 'noopener,noreferrer')
}

function operationLabel(value: string) { return value === 'share' ? '重新分享' : '转存' }
function operationTagType(value: string) { return value === 'share' ? 'info' : 'success' }
function statusLabel(value: string) { return value === 'failed' ? '失败' : '成功' }
function statusTagType(value: string) { return value === 'failed' ? 'error' : 'success' }
function cleanupLabel(value: string) { return ({ pending: '待清理', cleaned: '已清理', failed: '清理失败' } as Record<string, string>)[value] || value || '-' }
function cleanupTagType(value: string) { return ({ pending: 'warning', cleaned: 'success', failed: 'error' } as Record<string, any>)[value] || 'default' }
function panLabel(value: string) { return ({ quark: '夸克网盘', xunlei: '迅雷网盘', baidu: '百度网盘', uc: 'UC 网盘', alipan: '阿里云盘', aliyun: '阿里云盘' } as Record<string, string>)[value] || value || '-' }
function triggerSourceLabel(value: string) { return ({ resource_link: '智能取链', auto_transfer: '自动转存', reshare: '重新分享', admin_transfer_task: '后台转存任务', melost: '聚合搜索', web_resource_detail: '网页详情' } as Record<string, string>)[value] || value || '系统' }
function accountLabel(record: TransferRecord) { return record.account_remark || record.account_username || (record.account_id ? `账号 #${record.account_id}` : '-') }
function displayValue(value: unknown) { return value === null || value === undefined || value === '' ? '-' : String(value) }
function formatNumber(value: number) { return new Intl.NumberFormat('zh-CN').format(value || 0) }

function formatDate(value: string | null | undefined) {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? '-' : date.toLocaleString('zh-CN', { hour12: false })
}

function toDateString(timestamp: number | null) {
  if (!timestamp) return undefined
  const date = new Date(timestamp)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function displayURL(value: string) {
  try {
    const url = new URL(value)
    return `${url.hostname}${url.pathname}`
  } catch {
    return value
  }
}

function cleanupTimeLabel(record: TransferRecord) {
  if (record.cleanup_status === 'cleaned') return formatDate(record.cleaned_at)
  return `到期 ${formatDate(record.cleanup_due_at)}`
}

function formatMetadata(value: string) {
  if (!value) return '{}'
  try { return JSON.stringify(JSON.parse(value), null, 2) } catch { return value }
}

onMounted(refreshData)
</script>

<style scoped>
.summary-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  overflow: hidden;
  border: 1px solid rgb(229 231 235);
  border-radius: 6px;
  background: white;
}
.summary-item { min-width: 0; padding: 10px 12px; border-right: 1px solid rgb(229 231 235); border-bottom: 1px solid rgb(229 231 235); }
.summary-item strong { display: block; margin-top: 3px; font-size: 20px; line-height: 1.25; }
.filter-grid { display: grid; grid-template-columns: repeat(1, minmax(0, 1fr)); gap: 12px; align-items: end; }
.filter-grid-collapsed { display: none; }
.filter-field { min-width: 0; }
.filter-field label { display: block; margin-bottom: 5px; font-size: 12px; font-weight: 600; color: rgb(75 85 99); }
.filter-actions { display: flex; gap: 8px; align-items: center; }
.audit-table { width: 100%; table-layout: fixed; border-collapse: collapse; font-size: 13px; }
.audit-table th { position: sticky; top: 0; z-index: 1; padding: 10px 12px; background: rgb(249 250 251); color: rgb(75 85 99); font-size: 12px; font-weight: 600; text-align: left; border-bottom: 1px solid rgb(229 231 235); }
.audit-table td { padding: 11px 12px; vertical-align: middle; border-bottom: 1px solid rgb(243 244 246); }
.audit-table tbody tr:hover { background: rgb(249 250 251); }
.audit-table th:nth-child(1) { width: 100px; }
.audit-table th:nth-child(2) { width: 180px; }
.audit-table th:nth-child(3) { width: 150px; }
.audit-table th:nth-child(4), .audit-table th:nth-child(5) { width: 150px; }
.audit-table th:nth-child(6) { width: 140px; }
.audit-table th:nth-child(7) { width: 160px; }
.audit-table th:nth-child(8) { width: 64px; }
.title-button { display: block; max-width: 100%; overflow: hidden; color: rgb(17 24 39); font-weight: 600; text-align: left; text-overflow: ellipsis; white-space: nowrap; cursor: pointer; }
.title-button:hover { color: rgb(37 99 235); }
.title-button:focus-visible { outline: 2px solid rgb(37 99 235); outline-offset: 2px; }
.mobile-record { padding: 14px; border: 1px solid rgb(229 231 235); border-radius: 6px; background: white; }
.mobile-record dt { margin-bottom: 2px; color: rgb(107 114 128); font-size: 12px; }
.mobile-record dd { overflow: hidden; color: rgb(31 41 55); text-overflow: ellipsis; white-space: nowrap; }
.detail-section { padding-bottom: 20px; border-bottom: 1px solid rgb(229 231 235); }
.detail-section:last-child { padding-bottom: 0; border-bottom: 0; }
.detail-section h2 { margin-bottom: 10px; color: rgb(17 24 39); font-size: 14px; font-weight: 700; }
:deep(.detail-row) { display: grid; grid-template-columns: 112px minmax(0, 1fr); gap: 12px; padding: 7px 0; font-size: 13px; }
:deep(.detail-row dt) { color: rgb(107 114 128); }
:deep(.detail-row dd) { display: flex; min-width: 0; align-items: flex-start; gap: 4px; color: rgb(31 41 55); }
.dark .summary-grid, .dark .mobile-record { border-color: rgb(55 65 81); background: rgb(31 41 55); }
.dark .summary-item { border-color: rgb(55 65 81); }
.dark .filter-field label { color: rgb(209 213 219); }
.dark .audit-table th { border-color: rgb(55 65 81); background: rgb(55 65 81); color: rgb(209 213 219); }
.dark .audit-table td { border-color: rgb(55 65 81); }
.dark .audit-table tbody tr:hover { background: rgb(55 65 81 / 0.45); }
.dark .title-button, .dark .mobile-record dd, .dark .detail-section h2, .dark :deep(.detail-row dd) { color: rgb(243 244 246); }
.dark .detail-section { border-color: rgb(55 65 81); }
@media (min-width: 640px) { .summary-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); } .filter-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .filter-keyword { grid-column: span 2; } }
@media (min-width: 768px) { .filter-grid-collapsed { display: grid; } }
@media (min-width: 1024px) { .filter-grid { grid-template-columns: repeat(4, minmax(0, 1fr)); } }
@media (min-width: 1280px) { .summary-grid { grid-template-columns: repeat(7, minmax(0, 1fr)); } .summary-item { border-bottom: 0; } }
@media (min-width: 1536px) { .filter-grid { grid-template-columns: minmax(220px, 1.5fr) repeat(6, minmax(128px, 1fr)) auto; } .filter-keyword { grid-column: span 1; } }
@media (prefers-reduced-motion: reduce) { * { scroll-behavior: auto !important; transition-duration: 0.01ms !important; } }
</style>
