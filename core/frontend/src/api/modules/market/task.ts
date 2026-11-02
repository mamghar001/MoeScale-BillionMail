import { i18n } from '@/i18n'
import { instance } from '@/api'
import type { TaskParams } from '@/views/market/task/interface'

const { t } = i18n.global

/**
 * 获取任务列表
 */
export function getTaskList(params: TaskParams) {
	return instance.get('/batch_mail/task/list', { params })
}

/**
 * 获取任务统计
 */
export function getTaskOverview(params: { task_id: number; start_time: number; end_time: number }) {
	return instance.get('/batch_mail/task/stat_chart', { params })
}

/**
 * 获取任务详情
 */
export function getTaskDetails(params: { id: number }) {
	return instance.get('/batch_mail/task/find', { params })
}

interface TaskAddParams {
	track_open: number
	track_click: number
	rotate_senders?: number
	sending_window_enabled?: number
	sending_window_start?: string
	sending_window_end?: string
	sending_window_tz?: string
	addresser: string
	full_name: string
	subject: string
	group_id: number
	template_id: number
	is_record: number
	warmup: number
	unsubscribe: number
	threads: number
	start_time: number
	remark: string
	tag_ids: number[]
	tag_logic: string
	workspace_id?: number
}

/**
 * 获取任务列表
 */
export function addTask(params: TaskAddParams) {
	return instance.post('/batch_mail/task/create', params, {
		fetchOptions: {
			loading: t('market.task.loading.creating'),
			successMessage: true,
		},
	})
}

/**
 * 更新任务
 */
export function updateTask(params: TaskAddParams & { task_id: number }) {
	return instance.post('/batch_mail/task/update', params, {
		fetchOptions: {
			loading: t('market.task.loading.updating'),
			successMessage: true,
		},
	})
}

/**
 * 删除任务
 */
export function deleteTask(params: { id: number }) {
	return instance.post('/batch_mail/task/delete', params, {
		fetchOptions: {
			loading: t('market.task.loading.deleting'),
			successMessage: true,
		},
	})
}

/**
 * 暂停任务
 */
export function pauseTask(params: { task_id: number }) {
	return instance.post('/batch_mail/task/pause', params, {
		fetchOptions: {
			loading: t('market.task.loading.pausing'),
			successMessage: true,
		},
	})
}

/**
 * 恢复任务
 */
export function resumeTask(params: { task_id: number }) {
	return instance.post('/batch_mail/task/resume', params, {
		fetchOptions: {
			loading: t('market.task.loading.resuming'),
			successMessage: true,
		},
	})
}

/**
 * 恢复任务
 */
export function sendTestEmail(params: {
	addresser: string
	full_name?: string
	subject: string
	recipient: string
	template_id: number
	unsubscribe?: number
}) {
	return instance.post('/batch_mail/task/send_test', params, {
		fetchOptions: {
			loading: t('market.task.loading.sendingTest'),
			successMessage: true,
		},
	})
}

export function getMailProvider(params: { task_id: number; status: number }) {
	return instance.get('/batch_mail/tracking/mail_provider', { params })
}

export function getMailProviderLogs(params: {
	page: number
	page_size: number
	task_id: number
	domain: string
}) {
	return instance.get('/batch_mail/tracking/logs', { params })
}

/**
 * Generate a sample rendered email preview with dynamic sender rotation, prospect attributes, and spintax
 */
export function getTaskSamplePreview(params: { id: number; seed?: number }) {
	return instance.get('/batch_mail/task/sample_preview', { params })
}

/**
 * 获取工作区列表
 */
export function getWorkspaceList() {
	return instance.get('/batch_mail/workspace/list')
}

/**
 * 创建工作区
 */
export function createWorkspace(data: { name: string; color?: string }) {
	return instance.post('/batch_mail/workspace/create', data, {
		fetchOptions: {
			successMessage: true,
		},
	})
}

/**
 * 更新工作区
 */
export function updateWorkspace(data: { id: number; name: string; color?: string }) {
	return instance.post('/batch_mail/workspace/update', data, {
		fetchOptions: {
			successMessage: true,
		},
	})
}

/**
 * 删除工作区
 */
export function deleteWorkspace(data: { id: number }) {
	return instance.post('/batch_mail/workspace/delete', data, {
		fetchOptions: {
			successMessage: true,
		},
	})
}

/**
 * 批量移动任务到工作区
 */
export function batchMoveTasks(data: { task_ids: number[]; workspace_id: number }) {
	return instance.post('/batch_mail/task/batch_move', data, {
		fetchOptions: {
			loading: 'Moving campaigns...',
			successMessage: true,
		},
	})
}

/**
 * 批量删除任务
 */
export function batchDeleteTasks(data: { task_ids: number[] }) {
	return instance.post('/batch_mail/task/batch_delete', data, {
		fetchOptions: {
			loading: 'Deleting campaigns...',
			successMessage: true,
		},
	})
}


