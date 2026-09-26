<template>
	<div>
		<label class="block text-sm font-medium text-basic mb-3">
			Business Hours Sending Window (Pause Campaigns at Night)
		</label>
		<div class="grid items-start grid-cols-1 md:grid-cols-2 gap-4">
			<div>
				<div class="flex items-center gap-12px mb-3">
					<n-switch v-model:value="form.enabled" :checked-value="1" :unchecked-value="0" />
					<span v-if="form.enabled" class="text-xs text-primary font-medium">Active: Night pause enabled for all campaigns</span>
					<span v-else class="text-xs text-desc">Disabled: Campaigns run 24/7 continuously</span>
				</div>
				<div class="space-y-3">
					<div class="flex items-center gap-12px">
						<div class="flex-1">
							<span class="block text-xs text-desc mb-1">Morning Resume Time</span>
							<n-input v-model:value="form.start" placeholder="08:00" />
						</div>
						<div class="flex-1">
							<span class="block text-xs text-desc mb-1">Evening Pause Time</span>
							<n-input v-model:value="form.end" placeholder="18:00" />
						</div>
					</div>
					<div>
						<span class="block text-xs text-desc mb-1">Target Prospect Timezone</span>
						<n-select v-model:value="form.timezone" :options="timezoneOptions" filterable />
					</div>
				</div>
			</div>
			<div>
				<n-alert type="info" class="w-full">
					<template #icon>
						<n-icon>
							<i class="i-mdi-clock-outline"></i>
						</n-icon>
					</template>
					When enabled, any campaign created via Web UI or API will automatically pause in the evening and sleep until morning. At {{ form.start || '08:00' }} sharp, sending automatically resumes so emails land in inboxes during business hours.
				</n-alert>
			</div>
		</div>
		<div class="mt-4">
			<n-button type="primary" :loading="loading" @click="handleSave">
				Save Business Hours
			</n-button>
		</div>
	</div>
</template>

<script lang="ts" setup>
import { instance } from '@/api'
import { Message, isObject } from '@/utils'

const loading = ref(false)

const form = reactive({
	enabled: 1,
	start: '08:00',
	end: '18:00',
	timezone: 'America/New_York',
})

const timezoneOptions = [
	{ label: 'Eastern Time (US / New York) [EST/EDT]', value: 'America/New_York' },
	{ label: 'Central Time (US / Chicago) [CST/CDT]', value: 'America/Chicago' },
	{ label: 'Mountain Time (US / Denver) [MST/MDT]', value: 'America/Denver' },
	{ label: 'Pacific Time (US / Los Angeles) [PST/PDT]', value: 'America/Los_Angeles' },
	{ label: 'London / GMT (Europe)', value: 'Europe/London' },
	{ label: 'Central European Time (Paris/Berlin)', value: 'Europe/Paris' },
	{ label: 'UTC', value: 'UTC' },
]

const fetchConfig = async () => {
	try {
		const res: any = await instance.get('/batch_mail/business_hours')
		if (res && res.data) {
			form.enabled = res.data.enabled !== undefined ? res.data.enabled : 1
			form.start = res.data.start || '08:00'
			form.end = res.data.end || '18:00'
			form.timezone = res.data.timezone || 'America/New_York'
		}
	} catch (e) {
		console.error('Failed to load business hours:', e)
	}
}

const handleSave = async () => {
	loading.value = true
	try {
		await instance.post('/batch_mail/business_hours', {
			enabled: form.enabled,
			start: form.start,
			end: form.end,
			timezone: form.timezone,
		})
		Message.success('Business hours saved successfully!')
	} catch (e) {
		Message.error('Failed to save business hours')
	} finally {
		loading.value = false
	}
}

onMounted(() => {
	fetchConfig()
})
</script>
