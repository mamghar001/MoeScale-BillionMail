package v1

import (
	"billionmail-core/utility/types/api_v1"

	"github.com/gogf/gf/v2/frame/g"
)

type ApiTemplates struct {
	Id                int    `json:"id" dc:"id"`
	ApiKey            string `json:"api_key" dc:"api key"`
	ApiName           string `json:"api_name" dc:"api name"`
	TemplateId        int    `json:"template_id" dc:"template id"`
	GroupId           int    `json:"group_id" dc:"group id"`
	Subject           string `json:"subject" dc:"subject"`
	Addresser         string `json:"addresser" dc:"addresser"`
	FullName          string `json:"full_name" dc:"full name"`
	Unsubscribe       int    `json:"unsubscribe" dc:"unsubscribe"`
	TrackOpen         int    `json:"track_open" dc:"track open"`
	TrackClick        int    `json:"track_click" dc:"track click"`
	Active            int    `json:"active" dc:"active"`
	CreateTime        int    `json:"create_time" dc:"create time"`
	UpdateTime        int    `json:"update_time" dc:"update time"`
	ExpireTime        int    `json:"expire_time" dc:"expire time"`
	LastKeyUpdateTime int    `json:"last_key_update_time" dc:"last key update time"`
}

type ApiMailLogs struct {
	Id        int    `json:"id" dc:"id"`
	ApiId     int    `json:"api_id" dc:"api id"`
	Recipient string `json:"recipient" dc:"recipient"`
	MessageId string `json:"message_id" dc:"message id"`
	Addresser string `json:"addresser" dc:"addresser"`
	SendTime  int    `json:"send_time" dc:"send time"`
}

type ApiTemplatesListReq struct {
	g.Meta        `path:"/batch_mail/api/list" method:"get" tags:"ApiMail" summary:"api list"`
	Authorization string `json:"authorization" dc:"Authorization" in:"header"`
	Page          int    `json:"page" dc:"page"`
	PageSize      int    `json:"page_size" dc:"page size"`
	Keyword       string `json:"keyword" dc:"Search Keyword"`
	Active        int    `json:"active" dc:"active"`
	StartTime     int    `json:"start_time" dc:"start time"`
	EndTime       int    `json:"end_time" dc:"end time"`
}

type ApiTemplatesInfo struct {
	ApiTemplates
	GroupId      int     `json:"group_id" dc:"group id"`
	SendCount    int     `json:"send_count" dc:"send count"`
	SuccessCount int     `json:"success_count" dc:"success count"`
	FailCount    int     `json:"fail_count" dc:"fail count"`
	OpenRate     float64 `json:"open_rate" dc:"open rate"`
	ClickRate    float64 `json:"click_rate" dc:"click rate"`
	DeliveryRate float64 `json:"delivery_rate" dc:"delivery rate"`
	BounceRate   float64 `json:"bounce_rate" dc:"bounce rate"`
	//UnsubscribeCount int      `json:"unsubscribe_count" dc:"unsubscribe count"`
	IpWhitelist     []string `json:"ip_whitelist" dc:"IP whitelist"`
	ServerAddresser string   `json:"server_addresser" dc:"server addresser"`
}

type ApiTemplatesListRes struct {
	api_v1.StandardRes
	Data struct {
		Total int                 `json:"total" dc:"total"`
		List  []*ApiTemplatesInfo `json:"list"  dc:"api templates list"`
	} `json:"data"`
}

type ApiSummaryStats struct {
	TotalSend       int     `json:"total_send" dc:"total send count"`
	AvgDeliveryRate float64 `json:"avg_delivery_rate" dc:"average delivery rate"`
	AvgOpenRate     float64 `json:"avg_open_rate" dc:"average open rate"`
	AvgClickRate    float64 `json:"avg_click_rate" dc:"average click rate"`
	AvgBounceRate   float64 `json:"avg_bounce_rate" dc:"average bounce rate"`
	//AvgUnsubRate     float64 `json:"avg_unsub_rate" dc:"average unsubscribe rate"`
	//TotalUnsubscribe int     `json:"total_unsubscribe" dc:"total unsubscribe count"`
}

type ApiOverviewStatsReq struct {
	g.Meta        `path:"/batch_mail/api/overview_stats" method:"get" tags:"ApiMail" summary:"api Overview"`
	Authorization string `json:"authorization" dc:"Authorization" in:"header"`
	StartTime     int    `json:"start_time" dc:"start time"`
	EndTime       int    `json:"end_time" dc:"end time"`
}

type ApiOverviewStatsRes struct {
	api_v1.StandardRes
	Data ApiSummaryStats `json:"data"`
}

type ApiTemplatesCreateReq struct {
	g.Meta        `path:"/batch_mail/api/create" method:"post" tags:"ApiMail" summary:"api create"`
	Authorization string `json:"authorization" dc:"Authorization" in:"header"`
	ApiName       string `json:"api_name" dc:"api name"`
	TemplateId    int    `json:"template_id" dc:"template id"`
	GroupId       int    `json:"group_id" dc:"Associated group ID"`
	Subject       string `json:"subject" dc:"subject"`
	Addresser     string `json:"addresser" dc:"addresser"`
	FullName      string `json:"full_name" dc:"full name"`
	Unsubscribe   int    `json:"unsubscribe" dc:"unsubscribe"`
	Active        int    `json:"active" dc:"active"`
	ExpireTime    int    `json:"expire_time" dc:"expire time"` // 0 is a permanently valid unit of seconds
	//IpWhitelistEnabled int      `json:"ip_whitelist_enabled" dc:"ip whitelist enabled"`
	IpWhitelist []string `json:"ip_whitelist" dc:"ip whitelist"`
	TrackOpen   int      `json:"track_open" v:"in:0,1" dc:"track open" default:"1"`
	TrackClick  int      `json:"track_click" v:"in:0,1" dc:"track click" default:"1"`
}

type ApiTemplatesCreateRes struct {
	api_v1.StandardRes
}

type ApiTemplatesUpdateReq struct {
	g.Meta        `path:"/batch_mail/api/update" method:"post" tags:"ApiMail" summary:"api update"`
	Authorization string `json:"authorization" dc:"Authorization" in:"header"`
	ID            int    `json:"id" dc:"id"`
	ApiName       string `json:"api_name" dc:"api name"`
	TemplateId    int    `json:"template_id" dc:"template id"`
	GroupId       int    `json:"group_id" dc:"Associated group ID"`
	Subject       string `json:"subject" dc:"subject"`
	Addresser     string `json:"addresser" dc:"addresser"`
	FullName      string `json:"full_name" dc:"full name"`
	Unsubscribe   int    `json:"unsubscribe" dc:"unsubscribe"`
	TrackOpen     int    `json:"track_open" dc:"track open"`
	TrackClick    int    `json:"track_click" dc:"track click"`
	Active        int    `json:"active" dc:"active"`
	ExpireTime    int    `json:"expire_time" dc:"Key expiration time (0 is permanent)"`
	ResetKey      bool   `json:"reset_key" dc:"reset key"`
	//IpWhitelistEnabled int      `json:"ip_whitelist_enabled" dc:"ip whitelist enabled"`
	IpWhitelist []string `json:"ip_whitelist" dc:"ip whitelist"`
}

type ApiTemplatesUpdateRes struct {
	api_v1.StandardRes
}

type ApiTemplatesDeleteReq struct {
	g.Meta        `path:"/batch_mail/api/delete" method:"post" tags:"ApiMail" summary:"api delete"`
	Authorization string `json:"authorization" dc:"Authorization" in:"header"`
	ID            int    `json:"id" dc:"id"`
}

type ApiTemplatesDeleteRes struct {
	api_v1.StandardRes
}

type ApiMailSendReq struct {
	g.Meta        `path:"/batch_mail/api/send" method:"post" tags:"ApiMail" summary:"Send single or multiple emails" dc:"Queue single or multiple emails for delivery. Authenticate using Master Admin API Token in Authorization header (Bearer <token>) or x-api-key header, or use a dedicated Template API Key."`
	Authorization string            `json:"authorization" dc:"Master Admin API Token ('Bearer <token>')" in:"header"`
	ApiKey        string            `json:"x-api-key" dc:"Master Admin API Token or Template API Key" in:"header"`
	TemplateId    int               `json:"template_id" dc:"Email template ID (optional, defaults to primary template or template bound to API key)"`
	Subject       string            `json:"subject" dc:"Custom email subject (optional, overrides template subject)"`
	Addresser     string            `json:"addresser" dc:"Sender email address (optional, e.g. all_replies@b2bprosperity.com)"`
	Recipient     string            `json:"recipient" dc:"Recipient email address (supports single email or multiple separated by comma, semicolon, space, or newline)"`
	Recipients    []string          `json:"recipients" dc:"Recipient email addresses (optional array format)"`
	Attribs       map[string]string `json:"attribs" dc:"Custom merge tags and variables (e.g. {'name': 'John', 'company': 'Acme'})"`
}

type ApiMailSendRes struct {
	api_v1.StandardRes
}

type ApiMailBatchSendReq struct {
	g.Meta        `path:"/batch_mail/api/batch_send" method:"post" tags:"ApiMail" summary:"Batch send emails to multiple recipients" dc:"Queue batch emails to a list of recipients. Authenticate using Master Admin API Token in Authorization header (Bearer <token>) or x-api-key header, or use a dedicated Template API Key."`
	Authorization string            `json:"authorization" dc:"Master Admin API Token ('Bearer <token>')" in:"header"`
	ApiKey        string            `json:"x-api-key" dc:"Master Admin API Token or Template API Key" in:"header"`
	TemplateId    int               `json:"template_id" dc:"Email template ID (optional, defaults to primary template or template bound to API key)"`
	Subject       string            `json:"subject" dc:"Custom email subject (optional, overrides template subject)"`
	Addresser     string            `json:"addresser" dc:"Sender email address (optional, e.g. all_replies@b2bprosperity.com)"`
	Recipients    []string          `json:"recipients" dc:"List of recipient email addresses"`
	Recipient     string            `json:"recipient" dc:"Recipient email address (optional comma/semicolon-separated fallback)"`
	Attribs       map[string]string `json:"attribs" dc:"Custom merge tags and variables (e.g. {'name': 'John', 'company': 'Acme'})"`
}

type ApiMailBatchSendRes struct {
	api_v1.StandardRes
}

type CampaignStatsReq struct {
	g.Meta        `path:"/batch_mail/api/campaign/stats" method:"get" tags:"ApiMail" summary:"Get comprehensive campaign stats, daily delivery/bounce breakdown, bounce reasons, domain stats, and failed samples" dc:"Detailed campaign stats. Authenticate using x-api-key header or Authorization: Bearer <token>."`
	Authorization string `p:"header:authorization" json:"authorization" dc:"Master Admin Token ('Bearer <token>') or API key" in:"header"`
	ApiKey        string `p:"header:x-api-key" json:"x-api-key" dc:"API Key" in:"header"`
	CampaignId    int64  `p:"campaign_id" json:"campaign_id" dc:"Campaign ID (e.g. 478). Optional - defaults to 0 for all campaigns" in:"query"`
	SendingDomain string `p:"sending_domain" json:"sending_domain" dc:"Filter by outbound sending domain (e.g. 'b2bbox.shop')" in:"query"`
	Domain        string `p:"domain" json:"domain" dc:"Alias for sending_domain" in:"query"`
	StartTime     int64  `p:"start_time" json:"start_time" dc:"Start time (Unix timestamp in seconds)" in:"query"`
	EndTime       int64  `p:"end_time" json:"end_time" dc:"End time (Unix timestamp in seconds)" in:"query"`
	StartDate     string `p:"start_date" json:"start_date" dc:"Start date in 'YYYY-MM-DD' format" in:"query"`
	EndDate       string `p:"end_date" json:"end_date" dc:"End date in 'YYYY-MM-DD' format" in:"query"`
	FailedLimit   int    `p:"failed_limit" json:"failed_limit" dc:"Number of recent failed emails to include (default: 50, max: 200)" in:"query"`
}

type CampaignStatsAliasReq struct {
	g.Meta        `path:"/batch_mail/api/campaign_stats" method:"get" tags:"ApiMail" summary:"Alias for /batch_mail/api/campaign/stats" dc:"Detailed campaign stats."`
	Authorization string `p:"header:authorization" json:"authorization" dc:"Master Admin Token ('Bearer <token>') or API key" in:"header"`
	ApiKey        string `p:"header:x-api-key" json:"x-api-key" dc:"API Key" in:"header"`
	CampaignId    int64  `p:"campaign_id" json:"campaign_id" dc:"Campaign ID (e.g. 478). Optional - defaults to 0 for all campaigns" in:"query"`
	SendingDomain string `p:"sending_domain" json:"sending_domain" dc:"Filter by outbound sending domain (e.g. 'b2bbox.shop')" in:"query"`
	Domain        string `p:"domain" json:"domain" dc:"Alias for sending_domain" in:"query"`
	StartTime     int64  `p:"start_time" json:"start_time" dc:"Start time (Unix timestamp in seconds)" in:"query"`
	EndTime       int64  `p:"end_time" json:"end_time" dc:"End time (Unix timestamp in seconds)" in:"query"`
	StartDate     string `p:"start_date" json:"start_date" dc:"Start date in 'YYYY-MM-DD' format" in:"query"`
	EndDate       string `p:"end_date" json:"end_date" dc:"End date in 'YYYY-MM-DD' format" in:"query"`
	FailedLimit   int    `p:"failed_limit" json:"failed_limit" dc:"Number of recent failed emails to include (default: 50, max: 200)" in:"query"`
}

type CampaignDetailItem struct {
	Id                   int    `json:"id"`
	TaskName             string `json:"task_name"`
	Subject              string `json:"subject"`
	Addresser            string `json:"addresser"`
	RecipientCount       int    `json:"recipient_count"`
	SendsCount           int    `json:"sends_count"`
	DeliveredCount       int    `json:"delivered_count"`
	BouncedCount         int    `json:"bounced_count"`
	DeferredCount        int    `json:"deferred_count"`
	TaskProcess          int    `json:"task_process"`
	StatusText           string `json:"status_text"`
	IsPaused             bool   `json:"is_paused"`
	StartTime            int64  `json:"start_time"`
	StartTimeFormatted   string `json:"start_time_formatted"`
	SendingWindowEnabled int    `json:"sending_window_enabled"`
	SendingWindowStart   string `json:"sending_window_start"`
	SendingWindowEnd     string `json:"sending_window_end"`
	SendingWindowTz      string `json:"sending_window_tz"`
}

type CampaignStatsFilter struct {
	CampaignId    int64  `json:"campaign_id"`
	SendingDomain string `json:"sending_domain"`
	StartTime     int64  `json:"start_time"`
	EndTime       int64  `json:"end_time"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
}

type CampaignSummaryMetric struct {
	TotalRecipients int     `json:"total_recipients"`
	TotalSent       int     `json:"total_sent"`
	Delivered       int     `json:"delivered"`
	Bounced         int     `json:"bounced"`
	Deferred        int     `json:"deferred"`
	Opened          int     `json:"opened"`
	Clicked         int     `json:"clicked"`
	DeliveryRate    float64 `json:"delivery_rate"`
	BounceRate      float64 `json:"bounce_rate"`
	OpenRate        float64 `json:"open_rate"`
	ClickRate       float64 `json:"click_rate"`
}

type CampaignDailyMetric struct {
	Date         string  `json:"date"`
	Sends        int     `json:"sends"`
	Delivered    int     `json:"delivered"`
	Bounced      int     `json:"bounced"`
	Deferred     int     `json:"deferred"`
	DeliveryRate float64 `json:"delivery_rate"`
	BounceRate   float64 `json:"bounce_rate"`
	Opened       int     `json:"opened"`
	Clicked      int     `json:"clicked"`
}

type BounceReasonSummaryItem struct {
	ReasonCategory string  `json:"reason_category"`
	Dsn            string  `json:"dsn"`
	Count          int     `json:"count"`
	Percentage     float64 `json:"percentage"`
	SampleError    string  `json:"sample_error"`
}

type DomainMetricItem struct {
	Domain       string  `json:"domain"`
	Sends        int     `json:"sends"`
	Delivered    int     `json:"delivered"`
	Bounced      int     `json:"bounced"`
	DeliveryRate float64 `json:"delivery_rate"`
	BounceRate   float64 `json:"bounce_rate"`
}

type ProviderMetricItem struct {
	MailProvider string  `json:"mail_provider"`
	Sends        int     `json:"sends"`
	Delivered    int     `json:"delivered"`
	Bounced      int     `json:"bounced"`
	DeliveryRate float64 `json:"delivery_rate"`
	BounceRate   float64 `json:"bounce_rate"`
}

type FailedEmailSampleItem struct {
	Recipient     string `json:"recipient"`
	Sender        string `json:"sender"`
	SendingDomain string `json:"sending_domain"`
	Status        string `json:"status"`
	Dsn           string `json:"dsn"`
	Relay         string `json:"relay"`
	Reason        string `json:"reason"`
	LogTime       int64  `json:"log_time"`
	Date          string `json:"date"`
}

type CampaignStatsData struct {
	Campaign             *CampaignDetailItem       `json:"campaign,omitempty"`
	Filter               CampaignStatsFilter       `json:"filter"`
	Summary              CampaignSummaryMetric     `json:"summary"`
	DailyStats           []CampaignDailyMetric     `json:"daily_stats"`
	BounceReasonsSummary []BounceReasonSummaryItem `json:"bounce_reasons_summary"`
	SendingDomains       []DomainMetricItem        `json:"sending_domains"`
	MailProviders        []ProviderMetricItem      `json:"mail_providers"`
	RecentFailedEmails   []FailedEmailSampleItem   `json:"recent_failed_emails"`
}

type CampaignStatsRes struct {
	api_v1.StandardRes
	Data CampaignStatsData `json:"data"`
}

type CampaignsListReq struct {
	g.Meta        `path:"/batch_mail/api/campaigns" method:"get" tags:"ApiMail" summary:"List marketing campaigns with delivery and bounce metrics" dc:"List campaigns. Authenticate using x-api-key header or Authorization: Bearer <token>."`
	Authorization string `p:"header:authorization" json:"authorization" dc:"Master Admin Token or API key" in:"header"`
	ApiKey        string `p:"header:x-api-key" json:"x-api-key" dc:"API Key" in:"header"`
	Status        string `p:"status" json:"status" dc:"Filter by status ('all', 'running', 'paused', 'completed', 'draft')" in:"query"`
	Page          int    `p:"page" json:"page" dc:"Page number (default: 1)" in:"query"`
	PageSize      int    `p:"page_size" json:"page_size" dc:"Page size (default: 50)" in:"query"`
}

type CampaignListItem struct {
	Id                   int     `json:"id"`
	TaskName             string  `json:"task_name"`
	Subject              string  `json:"subject"`
	Addresser            string  `json:"addresser"`
	RecipientCount       int     `json:"recipient_count"`
	SendsCount           int     `json:"sends_count"`
	DeliveredCount       int     `json:"delivered_count"`
	BouncedCount         int     `json:"bounced_count"`
	DeferredCount        int     `json:"deferred_count"`
	DeliveryRate         float64 `json:"delivery_rate"`
	BounceRate           float64 `json:"bounce_rate"`
	TaskProcess          int     `json:"task_process"`
	StatusText           string  `json:"status_text"`
	IsPaused             bool    `json:"is_paused"`
	StartTime            int64   `json:"start_time"`
	StartTimeFormatted   string  `json:"start_time_formatted"`
	CreateTime           int64   `json:"create_time"`
	SendingWindowEnabled int     `json:"sending_window_enabled"`
	SendingWindowStart   string  `json:"sending_window_start"`
	SendingWindowEnd     string  `json:"sending_window_end"`
	SendingWindowTz      string  `json:"sending_window_tz"`
}

type CampaignsListData struct {
	Total int                `json:"total"`
	Page  int                `json:"page"`
	List  []CampaignListItem `json:"list"`
}

type CampaignsListRes struct {
	api_v1.StandardRes
	Data CampaignsListData `json:"data"`
}

type CampaignFailedListReq struct {
	g.Meta        `path:"/batch_mail/api/campaign/failed_list" method:"get" tags:"ApiMail" summary:"Get paginated list of failed and bounced emails with exact SMTP reasons" dc:"Query failed emails. Authenticate using x-api-key header or Authorization: Bearer <token>."`
	Authorization string `p:"header:authorization" json:"authorization" dc:"Master Admin Token or API key" in:"header"`
	ApiKey        string `p:"header:x-api-key" json:"x-api-key" dc:"API Key" in:"header"`
	CampaignId    int64  `p:"campaign_id" json:"campaign_id" dc:"Campaign ID" in:"query"`
	SendingDomain string `p:"sending_domain" json:"sending_domain" dc:"Filter by sending domain" in:"query"`
	Domain        string `p:"domain" json:"domain" dc:"Alias for sending_domain" in:"query"`
	Dsn           string `p:"dsn" json:"dsn" dc:"Filter by DSN status (e.g. '5.1.1', '5.7.1')" in:"query"`
	StartTime     int64  `p:"start_time" json:"start_time" dc:"Start time (Unix timestamp)" in:"query"`
	EndTime       int64  `p:"end_time" json:"end_time" dc:"End time (Unix timestamp)" in:"query"`
	StartDate     string `p:"start_date" json:"start_date" dc:"Start date 'YYYY-MM-DD'" in:"query"`
	EndDate       string `p:"end_date" json:"end_date" dc:"End date 'YYYY-MM-DD'" in:"query"`
	Page          int    `p:"page" json:"page" dc:"Page number (default: 1)" in:"query"`
	PageSize      int    `p:"page_size" json:"page_size" dc:"Page size (default: 50)" in:"query"`
}

type CampaignFailedListData struct {
	Total int                     `json:"total"`
	Page  int                     `json:"page"`
	List  []FailedEmailSampleItem `json:"list"`
}

type CampaignFailedListRes struct {
	api_v1.StandardRes
	Data CampaignFailedListData `json:"data"`
}
