<template>
	<div>
		<label class="block text-sm font-medium text-basic mb-3">
			Campaign Circuit Breaker (Auto-Pause)
		</label>
		<div class="grid items-start grid-cols-1 md:grid-cols-2 gap-4">
			<div>
				<div class="flex items-center mb-2">
					<n-switch :value="enabled" :loading="loading" @update:value="handleToggleSwitch">
						<template #checked>Enabled</template>
						<template #unchecked>Disabled</template>
					</n-switch>
					<span class="ml-3 text-sm font-medium text-basic">
						{{ enabled ? 'Active (Auto-pause enabled)' : 'Disabled (No auto-pause)' }}
					</span>
				</div>
				<p class="text-xs text-desc">
					Automatically pauses active campaigns if genuine bad email bounces (unknown user / non-existent mailbox) exceed the threshold in a 15-minute sliding window. External IP blocks (e.g. Microsoft S3150) and spam filters are excluded.
				</p>
			</div>

			<div>
				<div class="flex items-center gap-2">
					<n-input-group>
						<n-input-group-label>Pause Threshold</n-input-group-label>
						<n-select
							v-model:value="thresholdPercent"
							:options="thresholdOptions"
							:disabled="!enabled"
							class="flex-1"
							@update:value="handleUpdateThreshold"
						/>
					</n-input-group>
				</div>
				<p class="mt-1 text-xs text-desc">
					Current threshold: <strong>{{ thresholdPercent }}%</strong> hard bounces before pausing. Recommended: 15%.
				</p>
			</div>
		</div>
	</div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { getCircuitBreaker, setCircuitBreaker } from '@/api/modules/settings/common'

const enabled = ref(true)
const thresholdPercent = ref(15)
const loading = ref(false)

const thresholdOptions = [
	{ label: '10% (Strict)', value: 10 },
	{ label: '15% (Standard - Recommended)', value: 15 },
	{ label: '20% (Permissive)', value: 20 },
	{ label: '25% (High Tolerance)', value: 25 },
	{ label: '30% (Very High Tolerance)', value: 30 },
]

const fetchSettings = async () => {
	try {
		loading.value = true
		const res: any = await getCircuitBreaker()
		if (res && res.data) {
			enabled.value = res.data.enabled !== false
			const th = res.data.threshold || 0.15
			thresholdPercent.value = th > 1 ? Math.round(th) : Math.round(th * 100)
		}
	} catch (e) {
		console.error('Failed to get circuit breaker settings:', e)
	} finally {
		loading.value = false
	}
}

const handleToggleSwitch = async (val: boolean) => {
	try {
		loading.value = true
		await setCircuitBreaker({
			enabled: val,
			threshold: thresholdPercent.value / 100,
		})
		enabled.value = val
	} catch (e) {
		console.error(e)
	} finally {
		loading.value = false
	}
}

const handleUpdateThreshold = async (val: number) => {
	try {
		loading.value = true
		await setCircuitBreaker({
			enabled: enabled.value,
			threshold: val / 100,
		})
		thresholdPercent.value = val
	} catch (e) {
		console.error(e)
	} finally {
		loading.value = false
	}
}

onMounted(() => {
	fetchSettings()
})
</script>

<style lang="scss" scoped></style>
