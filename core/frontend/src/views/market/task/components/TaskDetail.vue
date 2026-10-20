<template>
	<modal :title="modalTitle" :width="820" :footer="false">
		<n-spin :show="loading">
			<div v-if="taskData" class="flex flex-col gap-4">
				<n-descriptions
					label-placement="left"
					:bordered="true"
					:column="2"
					:label-style="{ width: '150px' }">
					<n-descriptions-item :label="$t('market.task.columns.id')" :span="1">
						<span class="font-mono font-bold text-primary">#{{ taskData.id }}</span>
					</n-descriptions-item>
					<n-descriptions-item :label="$t('market.task.columns.status')" :span="1">
						<n-tag v-if="taskData.task_process === 3 || taskData.pause === 1" size="small" :bordered="false" type="error">
							{{ $t('market.task.status.paused') }}
						</n-tag>
						<n-tag v-else-if="taskData.task_process === 0" size="small" :bordered="false" type="warning">
							{{ $t('market.task.status.pending') }}
						</n-tag>
						<n-tag v-else-if="taskData.task_process === 1" size="small" :bordered="false" type="info">
							{{ $t('market.task.status.processing') }}
						</n-tag>
						<n-tag v-else size="small" :bordered="false" type="success">
							{{ $t('market.task.status.done') }}
						</n-tag>
					</n-descriptions-item>

					<n-descriptions-item :label="$t('market.task.columns.time')" :span="1">
						{{ formatTime(taskData.create_time) }}
					</n-descriptions-item>
					<n-descriptions-item :label="$t('market.task.detail.sendTime')" :span="1">
						{{ taskData.start_time && taskData.start_time > 0 ? formatTime(taskData.start_time) : 'Immediate' }}
					</n-descriptions-item>

					<n-descriptions-item :label="$t('market.task.detail.subject')" :span="2">
						<span class="font-medium">{{ taskData.subject }}</span>
					</n-descriptions-item>

					<n-descriptions-item :label="$t('market.task.detail.displayName')" :span="1">
						{{ taskData.full_name?.trim() || '--' }}
					</n-descriptions-item>
					<n-descriptions-item :label="$t('market.task.detail.sender')" :span="1">
						{{ taskData.addresser }}
					</n-descriptions-item>

					<n-descriptions-item :label="$t('market.task.edit.rotateSenders')" :span="2">
						<n-tag v-if="taskData.rotate_senders === 0" size="small" :bordered="false" type="warning">
							Disabled (Single sender)
						</n-tag>
						<n-tag v-else size="small" :bordered="false" type="success">
							Enabled (Rotate across all fleet mailboxes &amp; IPs)
						</n-tag>
					</n-descriptions-item>

					<n-descriptions-item :label="$t('market.task.detail.groups')" :span="1">
						{{ taskData.group_name || '--' }}
					</n-descriptions-item>
					<n-descriptions-item :label="$t('market.task.columns.recipients')" :span="1">
						<span class="font-semibold">{{ taskData.recipient_count?.toLocaleString() || 0 }}</span>
					</n-descriptions-item>

					<n-descriptions-item :label="$t('market.task.edit.tags')" :span="1">
						<n-flex v-if="taskData.tags && taskData.tags.length > 0">
							<n-tag v-for="item of taskData.tags" :key="item.id" size="small">{{ item.name }}</n-tag>
						</n-flex>
						<span v-else>--</span>
					</n-descriptions-item>
					<n-descriptions-item :label="$t('market.task.edit.tagLogic')" :span="1">
						{{ taskData.tags && taskData.tags.length > 0 ? taskData.tag_logic : '--' }}
					</n-descriptions-item>

					<n-descriptions-item :label="$t('market.task.columns.success')" :span="1">
						<span class="text-green-600 font-semibold">{{ (taskData.success_count || taskData.deliveredCount || 0).toLocaleString() }}</span>
					</n-descriptions-item>
					<n-descriptions-item :label="$t('market.task.columns.failed')" :span="1">
						<span :class="(taskData.error_count || taskData.bouncedCount || 0) > 0 ? 'text-red-500 font-semibold' : ''">
							{{ (taskData.error_count || taskData.bouncedCount || 0).toLocaleString() }}
						</span>
					</n-descriptions-item>

					<n-descriptions-item :label="$t('market.task.columns.progress')" :span="2">
						<div class="flex items-center gap-3">
							<div class="flex-1">
								<n-progress
									type="line"
									status="success"
									indicator-placement="inside"
									:processing="progressPercent !== 100 && taskData.task_process === 1"
									:percentage="progressPercent"
									:show-indicator="false"
								/>
							</div>
							<span class="text-xs font-semibold">{{ progressPercent }}%</span>
						</div>
					</n-descriptions-item>

					<n-descriptions-item label="Business Hours" :span="2">
						<n-tag v-if="taskData.sending_window_enabled === 1" size="small" :bordered="false" type="info">
							Active: {{ taskData.sending_window_start }} - {{ taskData.sending_window_end }} ({{ taskData.sending_window_tz }})
						</n-tag>
						<n-tag v-else size="small" :bordered="false">
							Disabled (Sends continuously 24/7)
						</n-tag>
					</n-descriptions-item>

					<n-descriptions-item label="Open Tracking" :span="1">
						<n-tag size="small" :bordered="false" :type="taskData.track_open === 1 ? 'success' : 'default'">
							{{ taskData.track_open === 1 ? 'Enabled' : 'Disabled' }}
						</n-tag>
					</n-descriptions-item>
					<n-descriptions-item label="Click Tracking" :span="1">
						<n-tag size="small" :bordered="false" :type="taskData.track_click === 1 ? 'success' : 'default'">
							{{ taskData.track_click === 1 ? 'Enabled' : 'Disabled' }}
						</n-tag>
					</n-descriptions-item>

					<n-descriptions-item :label="$t('market.task.detail.threads')" :span="1">
						{{ taskData.threads === 0 ? $t('market.task.detail.auto') : taskData.threads }}
					</n-descriptions-item>
					<n-descriptions-item :label="$t('market.task.detail.unsubscribeLink')" :span="1">
						{{ taskData.unsubscribe === 1 ? $t('market.task.detail.yes') : $t('market.task.detail.no') }}
					</n-descriptions-item>

					<n-descriptions-item :label="$t('market.task.detail.remark')" :span="2">
						{{ taskData.remark || '--' }}
					</n-descriptions-item>
				</n-descriptions>

				<!-- Template Preview Section -->
				<div class="mt-2 border rounded-md p-3 bg-gray-50 dark:bg-dark-700">
					<div class="flex flex-wrap items-center justify-between gap-2 mb-2.5">
						<div class="flex items-center gap-2">
							<span class="font-semibold text-sm">
								Template:
								<span class="text-primary font-medium">{{ templateName || ('Template #' + taskData.template_id) }}</span>
							</span>
							<n-tag v-if="taskData.template_id" size="tiny" :bordered="false">
								ID: {{ taskData.template_id }}
							</n-tag>
						</div>

						<!-- Mode Switch & Re-roll Buttons -->
						<div class="flex items-center gap-2">
							<n-button-group size="small">
								<n-button
									:type="previewMode === 'template' ? 'primary' : 'default'"
									:secondary="previewMode !== 'template'"
									@click="previewMode = 'template'"
								>
									Raw Template
								</n-button>
								<n-button
									:type="previewMode === 'sample' ? 'primary' : 'default'"
									:secondary="previewMode !== 'sample'"
									:loading="sampleLoading && previewMode === 'sample'"
									@click="switchPreviewMode('sample')"
								>
									✨ Rendered Sample
								</n-button>
							</n-button-group>

							<n-button
								v-if="previewMode === 'sample'"
								size="small"
								type="info"
								secondary
								:loading="sampleLoading"
								@click="fetchSamplePreview"
							>
								🎲 Re-roll Sample
							</n-button>
						</div>
					</div>

					<!-- Sample Metadata Card (Shown in sample mode) -->
					<div
						v-if="previewMode === 'sample' && sampleData"
						class="mb-2.5 p-2.5 rounded bg-blue-50/80 dark:bg-blue-950/40 border border-blue-200 dark:border-blue-900 text-xs flex flex-col gap-1.5"
					>
						<div class="flex flex-wrap items-center justify-between gap-2">
							<div class="flex items-center gap-1.5">
								<span class="font-bold text-gray-700 dark:text-gray-300">Sender:</span>
								<span class="font-medium text-blue-700 dark:text-blue-300">
									{{ sampleData.sender_name || 'Fleet Sender' }} &lt;{{ sampleData.sender_email }}&gt;
								</span>
								<n-tag v-if="sampleData.rotate_senders === 1" size="tiny" type="success" :bordered="false">Rotated</n-tag>
							</div>
							<div class="flex items-center gap-1.5">
								<span class="font-bold text-gray-700 dark:text-gray-300">Prospect:</span>
								<span class="font-medium text-emerald-700 dark:text-emerald-300">
									{{ sampleData.recipient_name }} &lt;{{ sampleData.recipient_email }}&gt;
								</span>
							</div>
						</div>

						<div class="flex items-start gap-1.5 pt-1 border-t border-blue-100 dark:border-blue-900/50">
							<span class="font-bold text-gray-700 dark:text-gray-300 shrink-0">Subject:</span>
							<span class="font-medium text-gray-900 dark:text-gray-100 font-sans">
								{{ sampleData.rendered_subject || taskData.subject }}
							</span>
						</div>

						<div class="text-[11px] text-gray-500 dark:text-gray-400 italic">
							Simulating live delivery: Random spintax option chosen and template variables (e.g. {{ '{' }}{{ '{' }} .Task.FullName {{ '}' }}{{ '}' }}, {{ '{' }}{{ '{' }} .Subscriber.* {{ '}' }}{{ '}' }}) evaluated dynamically.
						</div>
					</div>

					<div class="border rounded bg-white overflow-hidden shadow-sm">
						<n-spin :show="sampleLoading && previewMode === 'sample'">
							<bt-preview
								v-if="currentPreviewContent"
								:height="320"
								:value="currentPreviewContent"
							/>
							<div v-else class="text-gray-400 text-center py-8 text-sm">
								{{ sampleLoading ? 'Generating sample email preview...' : 'No preview available' }}
							</div>
						</n-spin>
					</div>
				</div>
			</div>
		</n-spin>
	</modal>
</template>

<script lang="ts" setup>
import { NButton, NButtonGroup, NDescriptions, NDescriptionsItem, NFlex, NProgress, NSpin, NTag } from 'naive-ui'
import { formatTime, isObject } from '@/utils'
import { useModal } from '@/hooks/modal/useModal'
import { getTaskDetails, getTaskSamplePreview } from '@/api/modules/market/task'
import { getTemplateDetails } from '@/api/modules/market/template'
import type { Task } from '../interface'
import type { Template } from '@/views/market/template/interface'

interface SamplePreviewData {
	task_id: number
	task_name: string
	template_id: number
	template_name: string
	sender_email: string
	sender_name: string
	rotate_senders: number
	recipient_email: string
	recipient_name: string
	recipient_attribs: Record<string, any>
	raw_subject: string
	rendered_subject: string
	raw_content: string
	rendered_content: string
}

const { t } = useI18n()

const loading = ref(false)
const taskData = ref<Task | null>(null)
const templateName = ref('')
const templateContent = ref('')

const previewMode = ref<'template' | 'sample'>('template')
const sampleLoading = ref(false)
const sampleData = ref<SamplePreviewData | null>(null)

const currentPreviewContent = computed(() => {
	if (previewMode.value === 'sample') {
		return sampleData.value?.rendered_content || ''
	}
	return templateContent.value || ''
})

async function fetchSamplePreview() {
	if (!taskData.value?.id) return
	try {
		sampleLoading.value = true
		const res = await getTaskSamplePreview({ id: taskData.value.id, seed: Date.now() })
		if (isObject<SamplePreviewData>(res)) {
			sampleData.value = res
		}
	} catch (err) {
		console.error('Failed to load sample preview:', err)
	} finally {
		sampleLoading.value = false
	}
}

function switchPreviewMode(mode: 'template' | 'sample') {
	previewMode.value = mode
	if (mode === 'sample' && !sampleData.value) {
		fetchSamplePreview()
	}
}

const modalTitle = computed(() => {
	if (taskData.value?.id) {
		return `${t('market.task.detail.title')} [#${taskData.value.id}]`
	}
	return t('market.task.detail.title')
})

const progressPercent = computed(() => {
	if (!taskData.value) return 0
	if (taskData.value.progress !== undefined && taskData.value.progress !== null) {
		return taskData.value.progress
	}
	const total = taskData.value.recipient_count || 0
	const sent = taskData.value.success_count || taskData.value.sent_count || 0
	if (total <= 0) return 0
	const p = Math.round((sent / total) * 100)
	return p > 100 ? 100 : p
})

const [Modal, modalApi] = useModal({
	onChangeState: async isOpen => {
		if (isOpen) {
			const state = modalApi.getState<{ row: Task }>()
			taskData.value = { ...state.row }
			templateName.value = ''
			templateContent.value = ''
			previewMode.value = 'template'
			sampleData.value = null

			if (!state.row?.id) return

			try {
				loading.value = true

				// 1. Fetch fresh task details from backend
				const taskRes = await getTaskDetails({ id: state.row.id })
				if (isObject<Task>(taskRes)) {
					// Merge fresh fields from backend (rotate_senders, sending_window_enabled, etc.)
					taskData.value = {
						...taskData.value,
						...taskRes,
						// Preserve group_name and tags if backend find didn't populate them
						group_name: state.row.group_name || taskRes.group_name,
						tags: state.row.tags?.length ? state.row.tags : taskRes.tags,
					}
				}

				// 2. Fetch template content if template_id exists
				const tplId = taskData.value?.template_id
				if (tplId) {
					const tplRes = await getTemplateDetails({ id: String(tplId) })
					if (isObject<Template>(tplRes)) {
						templateName.value = tplRes.temp_name || ''
						templateContent.value = tplRes.html_content || ''
					}
				}
			} catch (err) {
				console.error('Failed to load full task or template details:', err)
			} finally {
				loading.value = false
			}
		} else {
			taskData.value = null
			templateName.value = ''
			templateContent.value = ''
			previewMode.value = 'template'
			sampleData.value = null
		}
	},
})
</script>

<style lang="scss" scoped></style>
