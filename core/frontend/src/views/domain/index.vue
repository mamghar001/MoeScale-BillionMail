<template>
	<div class="p-24px">
		<div class="bt-title">{{ t('layout.menu.domain') }}</div>
		<bt-table-layout>
			<template #toolsLeft>
				<n-button type="primary" @click="handleAddDomain">{{ t('domain.addDomain') }}</n-button>
				<bt-help href="https://www.billionmail.com/start/domain.html" :text="t('domain.help')">
				</bt-help>
			</template>
			<template #table>
				<n-data-table :loading="loading" :columns="columns" :data="tableList">
					<template #empty>
						<bt-table-help></bt-table-help>
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
				<form-modal />
				<ssl-modal />
				<dns-modal />
				<domain-ip-set-modal />
				<blacklist-modal />
				<check-logs-modal />
			</template>
		</bt-table-layout>
	</div>
</template>

<script lang="tsx" setup>
import { DataTableColumns, NFlex, NButton, NTag, NSwitch } from 'naive-ui'
import { confirm, getByteUnit } from '@/utils'
import { useModal } from '@/hooks/modal/useModal'
import { useTableData } from '@/hooks/useTableData'
import { deleteDomain, getDomainList, setDefaultDomain, updateDomain } from '@/api/modules/domain'
import type { MailDomain, MailDomainParams } from './interface'

import DomainForm from './components/DomainForm.vue'
import DomainSsl from './components/DomainSsl/index.vue'
import DomainDns from './components/DomainDns.vue'
import DomainIpSet from './components/DomainIpSet.vue'
import IpStatus from './components/IpStatus.vue'
import NotSpam from './components/NotSpam.vue'
import BlacklistDetection from './components/BlacklistDetection.vue'
import CheckLogs from './components/CheckLogs.vue'

const { t } = useI18n()

const { tableParams, tableList, loading, tableTotal, getTableData } = useTableData<
	MailDomain,
	MailDomainParams
>({
	immediate: true,
	params: {
		page: 1,
		page_size: 10,
		keyword: '',
	},
	fetchFn: getDomainList,
})

const route = useRoute()
const router = useRouter()

const getDnsSummary = (row: MailDomain) => {
	const dns = row.dns_records
	if (!dns) return { valid: false, count: 0, invalidNames: 'All' }
	const items = [
		{ name: 'A', valid: Boolean(dns.a?.valid) },
		{ name: 'MX', valid: Boolean(dns.mx?.valid) },
		{ name: 'SPF', valid: Boolean(dns.spf?.valid) },
		{ name: 'DKIM', valid: Boolean(dns.dkim?.valid || dns.dkim_short?.valid) },
		{ name: 'DMARC', valid: Boolean(dns.dmarc?.valid) },
	]
	const validCount = items.filter(i => i.valid).length
	const allValid = validCount === items.length
	const invalidNames = items.filter(i => !i.valid).map(i => i.name).join(', ')
	return {
		valid: allValid,
		count: validCount,
		invalidNames,
	}
}

const handleToggleActive = async (row: MailDomain, active: boolean) => {
	try {
		await updateDomain({
			domain: row.domain,
			quota: row.quota,
			mailboxes: row.mailboxes,
			email: row.email,
			urls: row.urls || [],
			hostname: row.a_record,
			outbound_ip: row.multi_ip_domains?.outbound_ip || '',
			active: active ? 1 : 0,
		} as any)
		row.active = active ? 1 : 0
		getTableData()
	} catch (e) {
		console.error(e)
	}
}

// Table columns
const columns = ref<DataTableColumns<MailDomain>>([
	{
		key: 'domain',
		title: t('domain.columns.domain'),
		minWidth: 130,
		ellipsis: {
			tooltip: true,
		},
		render: row => {
			return (
				<div class="inline-flex items-center">
					{row.hasbrandinfo == 1 && <i class="i-domain:brand-info w-4 h-4 mr-4px"></i>}
					<NButton text type="primary" onClick={() => handleEdit(row)}>
						{row.domain}
					</NButton>
					{row.default === 1 && (
						<NTag size="small" class="ml-8px" bordered={false}>
							{t('domain.status.default')}
						</NTag>
					)}
				</div>
			)
		},
	},
	{
		key: 'not_in_spam_list',
		title: 'Not in Spam List',
		render: row => (
			<NotSpam
				record={row.a_record}
				check-result={row.black_check_result}
				onShowCheck={() => {
					blacklistModalApi.setState({ row })
					blacklistModalApi.open()
				}}
				onShowCheckLogs={() => {
					checkLogsModalApi.setState({ row })
					checkLogsModalApi.open()
				}}
			/>
		),
	},
	{
		key: 'dns_status',
		title: 'DNS Status',
		minWidth: 120,
		render: row => {
			const summary = getDnsSummary(row)
			return (
				<div
					class="inline-flex items-center gap-6px cursor-pointer"
					onClick={() => handleDNSRecord(row)}
					title={
						summary.valid
							? 'All DNS records (A, MX, SPF, DKIM, DMARC) are valid. Click to view.'
							: `Missing/Invalid: ${summary.invalidNames}. Click to view details.`
					}>
					<i
						class={[
							'text-16px',
							summary.valid ? 'i-ooui:success text-primary' : 'i-ooui:clear text-error',
						]}></i>
					<span class={summary.valid ? 'text-primary font-medium' : 'text-error font-medium'}>
						{summary.valid ? 'DNS OK' : `${summary.count}/5 Valid`}
					</span>
				</div>
			)
		},
	},
	{
		key: 'quota',
		title: t('domain.columns.quota'),
		render: row => getByteUnit(row.quota),
	},
	{
		key: 'mailboxes',
		title: t('domain.columns.mailboxes'),
	},
	{
		key: 'current_quota',
		title: 'Current usage',
		render: row => getByteUnit(row.current_usage),
	},
	{
		key: 'mailbox_quota',
		title: t('domain.columns.defaultMailboxSize'),
		render: row => getByteUnit(row.mailbox_quota),
	},
	// {
	// 	key: 'catch_all',
	// 	title: 'Catch All',
	// 	render: row => (
	// 		<NSwitch
	// 			value={row.catch_all}
	// 			size="small"
	// 			onUpdateValue={(val: boolean) => {
	// 				if (val) {
	// 					handleOpenCatch(row)
	// 				} else {
	// 					handleCloseCatch(row)
	// 				}
	// 			}}
	// 		/>
	// 	),
	// },
	{
		key: 'multi_ip_domains',
		title: t('domain.columns.dedicatedIp'),
		render: row => {
			if (row.multi_ip_domains) {
				return (
					<div
						class="flex items-center cursor-pointer"
						onClick={() => {
							handleMultiIpDomains(row)
						}}>
						<span class="leading-14px mr-6px">{row.multi_ip_domains.outbound_ip}</span>
						<IpStatus status={row.multi_ip_domains.status}></IpStatus>
					</div>
				)
			}
			return '--'
		},
	},
	{
		key: 'ssl',
		title: 'SSL',
		render: row => {
			if (row.cert_info && row.cert_info.endtime) {
				const ssl = row.cert_info
				const day = Math.floor((ssl.endtime * 1000 - Date.now()) / 1000 / 60 / 60 / 24)
				return (
					<NButton
						type={day < 0 ? 'error' : 'primary'}
						text
						onClick={() => {
							handleShowSsl(row)
						}}>
						{day < 0 ? t('domain.ssl.expired') : t('domain.ssl.daysLeft', { days: day })}
					</NButton>
				)
			}
			return (
				<NButton
					type="warning"
					text
					onClick={() => {
						handleShowSsl(row)
					}}>
					{t('domain.ssl.notSet')}
				</NButton>
			)
		},
	},
	{
		key: 'active',
		title: 'Status',
		width: 110,
		render: row => (
			<div class="inline-flex items-center gap-6px">
				<NSwitch
					value={row.active === 1}
					size="small"
					disabled={row.default === 1}
					onUpdateValue={async (val: boolean) => {
						await handleToggleActive(row, val)
					}}
				/>
				<span class={row.active === 1 ? 'text-primary text-12px font-medium' : 'text-gray-400 text-12px'}>
					{row.active === 1 ? 'Active' : 'Inactive'}
				</span>
			</div>
		),
	},
	{
		title: t('domain.columns.actions'),
		key: 'actions',
		align: 'right',
		width: 280,
		render: row => (
			<NFlex inline={true}>
				<NButton
					type="primary"
					text={true}
					onClick={() => {
						handleDNSRecord(row)
					}}>
					<span class="inline-flex items-center gap-4px">
						{t('domain.actions.dnsRecord')}
						<i
							class={[
								'text-14px',
								getDnsSummary(row).valid ? 'i-ooui:success text-primary' : 'i-ooui:clear text-error',
							]}></i>
					</span>
				</NButton>
				<NButton
					type="primary"
					text={true}
					onClick={() => {
						handleEdit(row)
					}}>
					{t('common.actions.edit')}
				</NButton>
				<NButton
					type="primary"
					text={true}
					disabled={row.default === 1}
					onClick={() => {
						handleSetDefault(row)
					}}>
					{t('domain.actions.setDefault')}
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

const [FormModal, formModalApi] = useModal({
	component: DomainForm,
	state: {
		isEdit: false,
		refresh: getTableData,
	},
})

// Handle add domain
const handleAddDomain = () => {
	formModalApi.setState({
		row: null,
		isEdit: false,
	})
	formModalApi.open()
}

// Handle open catch all
// const handleOpenCatch = (row: MailDomain) => {
// 	catchModalApi.setState({ row })
// 	catchModalApi.open()
// }

// Handle close catch all
// const handleCloseCatch = (row: MailDomain) => {
// 	console.log(row)
// }

const [SslModal, sslModalApi] = useModal({
	component: DomainSsl,
	state: {
		refresh: getTableData,
	},
})

// Handle show ssl
const handleShowSsl = (row: MailDomain) => {
	sslModalApi.setState({ row })
	sslModalApi.open()
}

const [DnsModal, dnsModalApi] = useModal({
	component: DomainDns,
	state: {
		refresh: getTableData,
	},
})

// Handle DNS records
const handleDNSRecord = (row: MailDomain) => {
	dnsModalApi.setState({ row })
	dnsModalApi.open()
}

const handleSetDefault = (row: MailDomain) => {
	confirm({
		title: t('domain.setDefault.title', { domain: row.domain }),
		content: t('domain.setDefault.confirm'),
		onConfirm: async () => {
			await setDefaultDomain({ domain: row.domain })
			getTableData()
		},
	})
}

// Handle edit
const handleEdit = (row: MailDomain) => {
	router.push({
		name: 'EditDomain',
		params: {
			domain: row.domain,
		},
	})
}

const [DomainIpSetModal, domainIpSetModalApi] = useModal({
	component: DomainIpSet,
})

// Handle multi ip domains
const handleMultiIpDomains = (row: MailDomain) => {
	domainIpSetModalApi.setState({ row })
	domainIpSetModalApi.open()
}

// Handle delete
const handleDelete = (row: MailDomain) => {
	confirm({
		title: t('domain.delete.title'),
		content: t('domain.delete.confirm', { domain: row.domain }),
		confirmText: t('common.actions.delete'),
		confirmType: 'error',
		onConfirm: async () => {
			await deleteDomain({ domain: row.domain })
			getTableData()
		},
	})
}

const [BlacklistModal, blacklistModalApi] = useModal({
	component: BlacklistDetection,
})

const [CheckLogsModal, checkLogsModalApi] = useModal({
	component: CheckLogs,
})

// Whether should open create modal automic
const initDomainFlag = ref('')

onMounted(() => {
	initDomainFlag.value = route.query.init as string
	if (initDomainFlag.value === 'init-domain') {
		handleAddDomain()
	}
})
</script>

<style lang="scss" scoped>
.domain-container {
	padding: 16px;
	background-color: #fff;
	border-radius: 3px;

	.action-bar {
		display: flex;
		justify-content: space-between;
		margin-bottom: 16px;
	}
}
</style>
