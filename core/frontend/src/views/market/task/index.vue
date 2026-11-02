<template>
	<div>
		<bt-table-layout>
			<template #toolsLeft>
				<div class="flex items-center gap-12px flex-wrap">
					<n-button type="primary" @click="handleAdd">
						{{ t('market.task.actions.add') }}
					</n-button>

					<!-- Workspace Filter Dropdown -->
					<div class="flex items-center gap-6px">
						<n-select
							v-model:value="selectedWorkspace"
							:options="workspaceOptions"
							placeholder="Filter by Workspace"
							style="width: 210px"
							size="medium"
							@update:value="handleWorkspaceChange"
						/>
						<n-button
							secondary
							title="Manage Workspaces / Clients"
							@click="openWorkspaceModal"
						>
							Workspaces
						</n-button>
					</div>

					<!-- Status Filter Dropdown -->
					<div class="flex items-center">
						<n-select
							v-model:value="selectedStatus"
							:options="statusOptions"
							placeholder="Filter by Status"
							style="width: 160px"
							size="medium"
							@update:value="handleStatusChange"
						/>
					</div>

					<!-- Bulk Action Buttons -->
					<div v-if="checkedRowKeys.length > 0" class="flex items-center gap-8px border-l border-gray-300 dark:border-dark-500 pl-12px">
						<span class="text-xs text-gray-500 font-medium">{{ checkedRowKeys.length }} selected:</span>
						<n-button type="info" size="small" secondary @click="openBatchMoveModal">
							Move to Workspace
						</n-button>
						<n-button type="error" size="small" secondary @click="handleBatchDelete">
							Delete Selected
						</n-button>
					</div>
				</div>
			</template>
			<template #toolsRight>
				<bt-search
					v-model:value="tableParams.keyword"
					:width="320"
					:placeholder="t('market.task.search.subjectPlaceholder')"
					@search="() => getTableData(true)">
				</bt-search>
			</template>
			<template #table>
				<n-data-table
					v-model:checked-row-keys="checkedRowKeys"
					:row-key="row => row.id"
					:loading="loading"
					:columns="columns"
					:data="tableList"
				>
					<template #empty>
						<bt-table-help> </bt-table-help>
					</template>
				</n-data-table>
			</template>
			<template #pageRight>
				<bt-table-page
					v-model:page="tableParams.page"
					v-model:page-size="tableParams.page_size"
					:item-count="tableTotal"
					@refresh="getTableData">
				</bt-table-page>
			</template>
			<template #modal>
				<status-modal />
				<detail-modal />
				<workspace-modal ref="workspaceModalRef" @change="handleWorkspaceModalChange" />

				<!-- Batch Move Modal -->
				<bt-modal
					v-model:show="batchMoveVisible"
					title="Move Selected Campaigns to Workspace"
					:width="500"
					confirm-text="Move Campaigns"
					@confirm="handleConfirmBatchMove"
				>
					<div class="py-16px">
						<p class="text-sm text-gray-600 dark:text-gray-300 mb-12px">
							Select the target workspace for <b>{{ checkedRowKeys.length }}</b> selected campaign(s):
						</p>
						<n-select
							v-model:value="batchTargetWorkspaceId"
							:options="batchTargetWorkspaceOptions"
							placeholder="Select Target Workspace"
						/>
					</div>
				</bt-modal>
			</template>
		</bt-table-layout>
	</div>
</template>

<script lang="tsx" setup>
import { computed, ref, onMounted } from 'vue'
import {
	DataTableColumns,
	NButton,
	NFlex,
	NProgress,
	NTag,
	NSelect,
} from 'naive-ui'
import { useModal } from '@/hooks/modal/useModal'
import { useTableData } from '@/hooks/useTableData'
import { confirm, formatDurationHighest, Message } from '@/utils'
import {
	deleteTask,
	getTaskList,
	pauseTask,
	resumeTask,
	getWorkspaceList,
	batchMoveTasks,
	batchDeleteTasks,
} from '@/api/modules/market/task'
import type { Task, TaskParams, WorkspaceItem } from './interface'

import TaskDetail from './components/TaskDetail.vue'
import TaskStatus from './components/TaskStatus.vue'
import WorkspaceModal from './components/WorkspaceModal.vue'

const { t } = useI18n()
const router = useRouter()

// Selection and Batch Actions state
const checkedRowKeys = ref<number[]>([])
const batchMoveVisible = ref(false)
const batchTargetWorkspaceId = ref<number>(0)

// Workspaces state
const selectedWorkspace = ref<number>(0)
const workspaces = ref<WorkspaceItem[]>([])
const workspaceModalRef = ref<any>(null)

// Status filter state
const selectedStatus = ref<number>(-1)
const statusOptions = computed(() => [
	{ label: '📊 All Statuses', value: -1 },
	{ label: '⏳ Pending', value: 0 },
	{ label: '🚀 In Progress', value: 1 },
	{ label: '✅ Completed', value: 2 },
	{ label: '⏸️ Paused', value: 3 },
])

const handleStatusChange = (val: number) => {
	selectedStatus.value = val
	tableParams.value.status = val
	checkedRowKeys.value = []
	getTableData(true)
}

const loadWorkspaces = async () => {
	try {
		const res: any = await getWorkspaceList()
		workspaces.value = Array.isArray(res) ? res : (res?.data || [])
	} catch (e) {
		// Silent error
	}
}

onMounted(() => {
	loadWorkspaces()
})

const handleWorkspaceModalChange = () => {
	loadWorkspaces()
	getTableData()
}

const openWorkspaceModal = () => {
	workspaceModalRef.value?.open()
}

// Workspace filter options for top select
const workspaceOptions = computed(() => {
	const opts: Array<{ label: string; value: number }> = [
		{ label: '📁 All Workspaces', value: 0 },
		{ label: '⚪ Default / Unassigned', value: -1 },
	]
	for (const ws of workspaces.value) {
		const countStr = ws.task_count ? ` (${ws.task_count})` : ''
		opts.push({
			label: `${ws.name}${countStr}`,
			value: ws.id,
		})
	}
	return opts
})

// Workspace options for batch move dialog
const batchTargetWorkspaceOptions = computed(() => {
	const opts: Array<{ label: string; value: number }> = [
		{ label: '⚪ Default / Unassigned (0)', value: 0 },
	]
	for (const ws of workspaces.value) {
		opts.push({
			label: ws.name,
			value: ws.id,
		})
	}
	return opts
})

const handleWorkspaceChange = (val: number) => {
	selectedWorkspace.value = val
	tableParams.value.workspace_id = val
	checkedRowKeys.value = []
	getTableData(true)
}

const openBatchMoveModal = () => {
	batchTargetWorkspaceId.value = selectedWorkspace.value > 0 ? selectedWorkspace.value : 0
	batchMoveVisible.value = true
}

const handleConfirmBatchMove = async () => {
	if (!checkedRowKeys.value.length) return
	try {
		await batchMoveTasks({
			task_ids: checkedRowKeys.value,
			workspace_id: batchTargetWorkspaceId.value,
		})
		Message.success('Campaigns moved successfully')
		batchMoveVisible.value = false
		checkedRowKeys.value = []
		getTableData()
		loadWorkspaces()
	} catch (e: any) {
		// Handled by request wrapper
	}
}

const handleBatchDelete = () => {
	if (!checkedRowKeys.value.length) return
	confirm({
		title: 'Delete Selected Campaigns',
		content: `Are you sure you want to permanently delete ${checkedRowKeys.value.length} selected campaign(s)? All recipient progress and tracking logs for these campaigns will be removed.`,
		confirmText: t('common.actions.delete'),
		confirmType: 'error',
		onConfirm: async () => {
			await batchDeleteTasks({ task_ids: checkedRowKeys.value })
			Message.success('Selected campaigns deleted')
			checkedRowKeys.value = []
			getTableData()
			loadWorkspaces()
		},
	})
}

const { loading, tableParams, tableList, tableTotal, getTableData } = useTableData<
	Task,
	TaskParams
>({
	loading: true,
	immediate: true,
	params: {
		page: 1,
		page_size: 10,
		keyword: '',
		status: -1,
		workspace_id: 0,
	},
	fetchFn: getTaskList,
})

// Table columns
const columns = ref<DataTableColumns<Task>>([
	{
		type: 'selection',
		width: 45,
	},
	{
		key: 'id',
		title: t('market.task.columns.id'),
		width: '6%',
		minWidth: 70,
		render: row => <span class="font-mono font-semibold">#{row.id}</span>,
	},
	{
		key: 'workspace_name',
		title: 'Workspace',
		width: '10%',
		minWidth: 110,
		render: row => {
			const ws = row.workspace_name
				? { name: row.workspace_name, color: row.workspace_color }
				: workspaces.value.find(w => w.id === row.workspace_id)
			if (ws && (row.workspace_id || 0) > 0) {
				return (
					<NTag
						size="small"
						round
						style={{
							backgroundColor: ws.color || '#2080f0',
							color: '#ffffff',
							border: 'none',
							fontWeight: '500',
							fontSize: '11px',
						}}
					>
						{ws.name}
					</NTag>
				)
			}
			return <span class="text-gray-400 text-xs">Default</span>
		},
	},
	{
		key: 'subject',
		title: t('market.task.columns.subject'),
		minWidth: 150,
		ellipsis: {
			tooltip: true,
		},
	},
	{
		key: 'recipient_count',
		title: t('market.task.columns.recipients'),
		width: '6%',
		minWidth: 80,
	},
	{
		key: 'success_count',
		title: t('market.task.columns.success'),
		width: '6%',
		minWidth: 80,
	},
	{
		key: 'error_count',
		title: t('market.task.columns.failed'),
		width: '6%',
		minWidth: 80,
		render: row => (
			<NButton
				type="error"
				text
				onClick={() => {
					handleShowError(row)
				}}>
				{row.error_count}
			</NButton>
		),
	},
	{
		key: 'active',
		title: t('market.task.columns.status'),
		width: '8%',
		minWidth: 90,
		render: row => {
			if (row.task_process === 3 || row.pause === 1)
				return (
					<NTag size="small" bordered={false} type="error">
						{t('market.task.status.paused')}
					</NTag>
				)
			if (row.task_process === 0)
				return (
					<NTag size="small" bordered={false} type="warning">
						{t('market.task.status.pending')}
					</NTag>
				)
			if (row.task_process === 1)
				return (
					<NTag size="small" bordered={false} type="info">
						{t('market.task.status.processing')}
					</NTag>
				)
			return (
				<NTag size="small" bordered={false} type="success">
					{t('market.task.status.done')}
				</NTag>
			)
		},
	},
	{
		key: 'remark',
		title: t('market.task.columns.remark'),
		width: '8%',
		minWidth: 80,
		ellipsis: {
			tooltip: true,
		},
		render: row => row.remark || '--',
	},
	{
		key: 'estimated_time_with_warmup',
		title: t('market.task.columns.estimatedTime'),
		width: '11%',
		minWidth: 110,
		ellipsis: {
			tooltip: true,
		},
		render: row => {
			if (row.task_process === 2) {
				return '--'
			}
			if (row.estimated_time_with_warmup >= 0) {
				return formatDurationHighest(row.estimated_time_with_warmup)
			}
			return t('market.task.status.noIpWarmup')
		},
	},
	{
		key: 'progress',
		title: t('market.task.columns.progress'),
		width: '8%',
		minWidth: 130,
		render: row => {
			return (
				<div class="max-w-160px">
					<NProgress
						type="line"
						status="success"
						indicator-placement="inside"
						processing={row.progress !== 100}
						percentage={row.progress}
						show-indicator={false}
					/>
				</div>
			)
		},
	},
	{
		title: t('common.columns.actions'),
		key: 'actions',
		align: 'right',
		width: 290,
		render: row => (
			<NFlex inline={true}>
				{row.task_process !== 2 && (
					<NButton
						type="primary"
						text={true}
						onClick={() => {
							handleSetStatus(row)
						}}>
						{row.task_process === 3
							? t('market.task.actions.send')
							: t('market.task.actions.pause')}
					</NButton>
				)}
				{row.task_process === 3 && (
					<NButton
						type="primary"
						text={true}
						onClick={() => {
							handleEdit(row)
						}}>
						{t('common.actions.edit')}
					</NButton>
				)}
				<NButton
					type="primary"
					text={true}
					onClick={() => {
						handleGoAnalytics(row)
					}}>
					{t('market.task.actions.analytics')}
				</NButton>
				<NButton
					type="primary"
					text={true}
					onClick={() => {
						handleDetail(row)
					}}>
					{t('market.task.actions.detail')}
				</NButton>
				<NButton
					type="primary"
					text={true}
					onClick={() => {
						handleCopy(row)
					}}>
					{t('common.actions.copy')}
				</NButton>
				<NButton
					type="error"
					text={true}
					onClick={() => {
						handleDelete(row)
					}}>
					{t('common.actions.delete')}
				</NButton>
			</NFlex>
		),
	},
])

// 添加任务
const handleAdd = () => {
	router.push('/market/task/edit')
}

// 前往分析
const handleGoAnalytics = (row: Task) => {
	router.push(`/market/task/analytics/${row.id}`)
}

// 复制任务
const handleCopy = (row: Task) => {
	router.push(`/market/task/edit?task_id=${row.id}`)
}

// 暂停/发送任务
const handleSetStatus = (row: Task) => {
	confirm({
		title: t('market.task.changeStatus.title', { subject: row.subject }),
		content:
			row.task_process === 3
				? t('market.task.changeStatus.startConfirm')
				: t('market.task.changeStatus.pauseConfirm'),
		onConfirm: async () => {
			if (row.task_process === 3) {
				await resumeTask({ task_id: row.id })
			} else {
				await pauseTask({ task_id: row.id })
			}
			getTableData()
		},
	})
}

const [StatusModal, statusModalApi] = useModal({
	component: TaskStatus,
})

// 查看任务详情
const handleShowError = (row: Task) => {
	statusModalApi.setState({ row })
	statusModalApi.open()
}

const [DetailModal, detailModalApi] = useModal({
	component: TaskDetail,
})

// 查看任务详情
const handleDetail = (row: Task) => {
	detailModalApi.setState({ row })
	detailModalApi.open()
}

// 删除任务
const handleDelete = (row: Task) => {
	confirm({
		title: t('market.task.delete.title'),
		content: t('market.task.delete.confirm', { subject: row.subject }),
		confirmText: t('common.actions.delete'),
		confirmType: 'error',
		onConfirm: async () => {
			await deleteTask({ id: row.id })
			getTableData()
			loadWorkspaces()
		},
	})
}

const handleEdit = (row: Task) => {
	router.push(`/market/task/edit?task_id=${row.id}&type=edit`)
}
</script>

<style lang="scss" scoped></style>
