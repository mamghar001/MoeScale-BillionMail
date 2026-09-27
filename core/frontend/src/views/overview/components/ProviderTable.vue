<template>
	<div class="provider-table">
		<n-data-table :columns="columns" :data="tableData" :max-height="320"> </n-data-table>
	</div>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { DataTableColumns, NDataTable } from 'naive-ui'
import { MailProvider } from '../types'

const { t } = useI18n()

const tableData = defineModel<MailProvider[]>('value')

const totalSends = computed(() => {
	return (tableData.value || []).reduce((sum, item) => sum + (item.sends || 0), 0)
})

const getRate = (val: number) => {
	return val >= 0 ? `${val}%` : '--'
}

const getShare = (sends: number) => {
	const total = totalSends.value
	if (!total || total <= 0 || !sends || sends <= 0) {
		return '0%'
	}
	const pct = (sends / total) * 100
	if (pct > 0 && pct < 0.01) {
		return '<0.01%'
	}
	if (pct < 1) {
		return `${Number(pct.toFixed(2))}%`
	}
	return `${Number(pct.toFixed(1))}%`
}

const columns = computed<DataTableColumns<MailProvider>>(() => [
	{
		key: 'mail_provider',
		title: t('overview.provider.mailProvider'),
		ellipsis: {
			tooltip: true,
		},
	},
	{
		key: 'share',
		title: t('overview.provider.share'),
		render: row => {
			const shareText = getShare(row.sends)
			const sendsText = (row.sends || 0).toLocaleString()
			return h(
				'span',
				{
					title: `${sendsText} / ${totalSends.value.toLocaleString()} sent`,
				},
				shareText
			)
		},
	},
	{
		key: 'delivery_rate',
		title: t('overview.provider.delivered'),
		render: row => {
			return getRate(row.delivery_rate)
		},
	},
	{
		key: 'open_rate',
		title: t('overview.provider.open'),
		render: row => {
			return getRate(row.open_rate)
		},
	},
	{
		key: 'click_rate',
		title: t('overview.provider.click'),
		render: row => {
			return getRate(row.click_rate)
		},
	},
	{
		key: 'bounce_rate',
		title: t('overview.provider.bounce'),
		render: row => {
			return getRate(row.bounce_rate)
		},
	},
])
</script>
