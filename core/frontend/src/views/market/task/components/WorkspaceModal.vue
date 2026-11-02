<template>
	<bt-modal
		v-model:show="visible"
		title="Manage Workspaces / Clients"
		:width="680"
		:footer="false"
		:on-after-leave="handleClose"
	>
		<div class="py-12px">
			<!-- Add/Edit Workspace Section -->
			<div class="mb-20px p-16px bg-gray-50 dark:bg-dark-700 rounded-8px border border-gray-100 dark:border-dark-600">
				<div class="text-sm font-semibold mb-12px text-gray-800 dark:text-gray-100">
					{{ editingId ? 'Edit Workspace' : 'Add New Workspace / Client' }}
				</div>
				<div class="flex items-center gap-12px">
					<n-input
						v-model:value="form.name"
						placeholder="Workspace name (e.g. Acme Corp, Outreach B2B)"
						class="flex-1"
						maxlength="50"
						show-count
						@keyup.enter="handleSave"
					/>
					<div class="flex items-center gap-6px">
						<n-color-picker
							v-model:value="form.color"
							:show-alpha="false"
							:swatches="colorSwatches"
							style="width: 80px"
						/>
					</div>
					<n-button type="primary" :loading="saving" :disabled="!form.name.trim()" @click="handleSave">
						{{ editingId ? 'Update' : 'Add' }}
					</n-button>
					<n-button v-if="editingId" quaternary @click="cancelEdit">
						Cancel
					</n-button>
				</div>
			</div>

			<!-- Workspaces List -->
			<div class="workspace-list">
				<div class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-8px px-4px flex justify-between">
					<span>Existing Workspaces</span>
					<span v-if="workspaces.length">{{ workspaces.length }} total</span>
				</div>

				<n-spin :show="loading">
					<div v-if="workspaces.length === 0 && !loading" class="text-center py-32px text-gray-400">
						No custom workspaces created yet. Create one above to organize campaigns by client or team!
					</div>

					<div v-else class="flex flex-col gap-8px max-h-360px overflow-y-auto pr-4px">
						<div
							v-for="ws in workspaces"
							:key="ws.id"
							class="flex items-center justify-between p-12px rounded-8px border border-gray-100 dark:border-dark-600 bg-white dark:bg-dark-800 hover:shadow-sm transition-all"
						>
							<div class="flex items-center gap-10px min-w-0">
								<span
									class="w-14px h-14px rounded-full flex-shrink-0 shadow-sm"
									:style="{ backgroundColor: ws.color || '#2080f0' }"
								/>
								<span class="font-medium text-gray-900 dark:text-gray-100 truncate text-sm">
									{{ ws.name }}
								</span>
								<n-tag size="tiny" round :bordered="false" type="info">
									{{ ws.task_count || 0 }} {{ (ws.task_count === 1) ? 'campaign' : 'campaigns' }}
								</n-tag>
							</div>

							<div class="flex items-center gap-4px flex-shrink-0">
								<n-button size="tiny" quaternary type="primary" @click="startEdit(ws)">
									Edit
								</n-button>
								<n-button size="tiny" quaternary type="error" @click="handleDelete(ws)">
									Delete
								</n-button>
							</div>
						</div>
					</div>
				</n-spin>
			</div>

			<div class="mt-20px pt-12px border-t border-gray-100 dark:border-dark-600 flex justify-between items-center text-xs text-gray-400">
				<span>Workspaces help organize &amp; filter email marketing campaigns.</span>
				<n-button size="small" secondary @click="visible = false">Close</n-button>
			</div>
		</div>
	</bt-modal>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { confirm, Message } from '@/utils'
import {
	getWorkspaceList,
	createWorkspace,
	updateWorkspace,
	deleteWorkspace,
} from '@/api/modules/market/task'
import type { WorkspaceItem } from '../interface'

const emit = defineEmits<{
	(e: 'change'): void
}>()

const visible = ref(false)
const loading = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const workspaces = ref<WorkspaceItem[]>([])

const colorSwatches = [
	'#2080f0',
	'#18a058',
	'#f0a020',
	'#d03050',
	'#8a2be2',
	'#00ced1',
	'#ff69b4',
	'#3f51b5',
	'#009688',
	'#e91e63',
]

const form = reactive({
	name: '',
	color: '#2080f0',
})

const fetchWorkspaces = async () => {
	loading.value = true
	try {
		const res: any = await getWorkspaceList()
		workspaces.value = Array.isArray(res) ? res : (res?.data || [])
	} catch (e: any) {
		Message.error('Failed to load workspaces')
	} finally {
		loading.value = false
	}
}

const open = () => {
	visible.value = true
	cancelEdit()
	fetchWorkspaces()
}

const handleClose = () => {
	cancelEdit()
}

const startEdit = (ws: WorkspaceItem) => {
	editingId.value = ws.id
	form.name = ws.name
	form.color = ws.color || '#2080f0'
}

const cancelEdit = () => {
	editingId.value = null
	form.name = ''
	form.color = '#2080f0'
}

const handleSave = async () => {
	const trimmed = form.name.trim()
	if (!trimmed) {
		Message.warning('Workspace name is required')
		return
	}

	saving.value = true
	try {
		if (editingId.value) {
			await updateWorkspace({
				id: editingId.value,
				name: trimmed,
				color: form.color,
			})
			Message.success('Workspace updated')
		} else {
			await createWorkspace({
				name: trimmed,
				color: form.color,
			})
			Message.success('Workspace created')
		}
		cancelEdit()
		await fetchWorkspaces()
		emit('change')
	} catch (e: any) {
		// Error handled by request wrapper
	} finally {
		saving.value = false
	}
}

const handleDelete = (ws: WorkspaceItem) => {
	const hasCampaigns = (ws.task_count || 0) > 0
	confirm({
		title: `Delete Workspace "${ws.name}"?`,
		content: hasCampaigns
			? `This workspace has ${ws.task_count} campaign(s). Deleting it will unassign those campaigns back to Default, but will NOT delete the campaigns. Proceed?`
			: `Are you sure you want to delete "${ws.name}"?`,
		confirmText: 'Delete',
		confirmType: 'error',
		onConfirm: async () => {
			await deleteWorkspace({ id: ws.id })
			Message.success('Workspace deleted')
			if (editingId.value === ws.id) {
				cancelEdit()
			}
			await fetchWorkspaces()
			emit('change')
		},
	})
}

defineExpose({
	open,
})
</script>

<style scoped>
.workspace-list {
	user-select: none;
}
</style>
