// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Console is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package enum

var PermissionMap = make(map[string][]string)

const (
	UserRead       = "system.user:read"
	UserAll        = "system.user:all"
	RoleRead       = "system.role:read"
	RoleAll        = "system.role:all"
	SecurityRead   = "system.security:read"
	SecurityAll    = "system.security:all"
	ClusterAll     = "system.cluster:all"
	ClusterRead    = "system.cluster:read"
	CommandAll     = "system.command:all"
	CommandRead    = "system.command:read"
	CredentialAll  = "system.credential:all"
	CredentialRead = "system.credential:read"
	SmtpServerAll  = "system.smtp_server:all"
	SmtpServerRead = "system.smtp_server:read"

	InstanceRead = "gateway.instance:read"
	InstanceAll  = "gateway.instance:all"
	EntryAll     = "gateway.entry:all"
	EntryRead    = "gateway.entry:read"
	RouterRead   = "gateway.router:read"
	RouterAll    = "gateway.router:all"
	FlowRead     = "gateway.flow:read"
	FlowAll      = "gateway.flow:all"

	AgentInstanceRead = "agent.instance:read"
	AgentInstanceAll  = "agent.instance:all"

	IndexAll     = "data.index:all"
	IndexRead    = "data.index:read"
	AliasAll     = "data.alias:all"
	AliasRead    = "data.alias:read"
	ViewsAll     = "data.view:all"
	ViewsRead    = "data.view:read"
	DiscoverAll  = "data.discover:all"
	DiscoverRead = "data.discover:read"

	RuleRead         = "alerting.rule:read"
	RuleAll          = "alerting.rule:all"
	AlertRead        = "alerting.alert:read"
	AlertAll         = "alerting.alert:all"
	AlertMessageRead = "alerting.message:read"
	AlertMessageAll  = "alerting.message:all"
	ChannelRead      = "alerting.channel:read"
	ChannelAll       = "alerting.channel:all"

	ClusterOverviewRead = "cluster.overview:read"
	ClusterOverviewAll  = "cluster.overview:all"
	MonitoringRead      = "cluster.monitoring:read"
	MonitoringAll       = "cluster.monitoring:all"
	ActivitiesRead      = "cluster.activities:read"
	ActivitiesAll       = "cluster.activities:all"
	AuditLogsRead       = "system.audit_logs:read"
	AuditLogsAll        = "system.audit_logs:all"
	DataMigrationRead   = "data_tools.migration:read"
	DataMigrationAll    = "data_tools.migration:all"
	DataComparisonRead  = "data_tools.comparison:read"
	DataComparisonAll   = "data_tools.comparison:all"
	DashboardRead       = "insight.dashboard:read"
	DashboardAll        = "insight.dashboard:all"
	DevtoolConsoleAll   = "devtool.console:all"
	DevtoolConsoleRead  = "devtool.console:read"
	WorkbenchAll        = "workbench:all"
	WorkbenchRead       = "workbench:read"

	TenantCustomerRead = "tenant.customer:read"
	TenantCustomerAll  = "tenant.customer:all"

	SubscriptionRead = "tenant.subscription:read"
	SubscriptionAll  = "tenant.subscription:all"
)

const (
	PermissionUserRead                  string = "user:read"
	PermissionUserWrite                        = "user:write"
	PermissionDisableBuiltinAdmin              = "user:disable_builtin_admin"
	PermissionRoleRead                         = "role:read"
	PermissionRoleWrite                        = "role:write"
	PermissionCommandRead                      = "command:read"
	PermissionCommandWrite                     = "command:write"
	PermissionElasticsearchClusterRead         = "es.cluster:read"
	PermissionElasticsearchClusterWrite        = "es.cluster:write" // es cluster
	PermissionElasticsearchIndexRead           = "es.index:read"
	PermissionElasticsearchIndexWrite          = "es.index:write" // es index metadata
	PermissionElasticsearchNodeRead            = "es.node:read"   //es node metadata
	PermissionActivityRead                     = "activity:read"
	PermissionActivityWrite                    = "activity:write"
	PermissionAuditLogRead                     = "audit_log:read"
	PermissionAuditLogWrite                    = "audit_log:write"
	PermissionAlertRuleRead                    = "alert.rule:read"
	PermissionAlertRuleWrite                   = "alert.rule:write"
	PermissionAlertHistoryRead                 = "alert.history:read"
	PermissionAlertHistoryWrite                = "alert.history:write"
	PermissionAlertMessageRead                 = "alert.message:read"
	PermissionAlertMessageWrite                = "alert.message:write"
	PermissionAlertChannelRead                 = "alert.channel:read"
	PermissionAlertChannelWrite                = "alert.channel:write"
	PermissionViewRead                         = "view:read"
	PermissionViewWrite                        = "view:write"
	PermissionLayoutRead                       = "layout:read"
	PermissionLayoutWrite                      = "layout:write"
	PermissionGatewayInstanceRead              = "gateway.instance:read"
	PermissionGatewayInstanceWrite             = "gateway.instance:write"
	PermissionGatewayEntryRead                 = "gateway.entry:read"
	PermissionGatewayEntryWrite                = "gateway.entry:write"
	PermissionGatewayRouterRead                = "gateway.router:read"
	PermissionGatewayRouterWrite               = "gateway.router:write"
	PermissionGatewayFlowRead                  = "gateway.flow:read"
	PermissionGatewayFlowWrite                 = "gateway.flow:write"
	PermissionElasticsearchMetricRead          = "es.metric:read"

	PermissionAgentInstanceRead   = "agent.instance:read"
	PermissionAgentInstanceWrite  = "agent.instance:write"
	PermissionCredentialRead      = "credential:read"
	PermissionCredentialWrite     = "credential:write"
	PermissionMigrationTaskRead   = "task:read"
	PermissionMigrationTaskWrite  = "task:write"
	PermissionComparisonTaskRead  = "comparison_task:read"
	PermissionComparisonTaskWrite = "comparison_task:write"
	PermissionSmtpServerRead      = "smtp_server:read"
	PermissionSmtpServerWrite     = "smtp_server:write"
)

var (
	UserReadPermission = []string{PermissionUserRead}
	UserAllPermission  = []string{PermissionUserRead, PermissionUserWrite, PermissionRoleRead}

	RoleReadPermission     = []string{PermissionRoleRead}
	RoleAllPermission      = []string{PermissionRoleRead, PermissionRoleWrite}
	SecurityReadPermission = []string{PermissionUserRead, PermissionRoleRead}
	SecurityAllPermission  = []string{PermissionUserRead, PermissionUserWrite, PermissionRoleRead, PermissionRoleWrite, PermissionDisableBuiltinAdmin}

	ClusterReadPermission = []string{PermissionElasticsearchClusterRead}
	ClusterAllPermission  = []string{PermissionElasticsearchClusterRead, PermissionElasticsearchClusterWrite}

	CommandReadPermission = []string{PermissionCommandRead}
	CommandAllPermission  = []string{PermissionCommandRead, PermissionCommandWrite}

	InstanceReadPermission = []string{PermissionGatewayInstanceRead}
	InstanceAllPermission  = []string{PermissionGatewayInstanceRead, PermissionGatewayInstanceWrite}

	EntryReadPermission = []string{PermissionGatewayEntryRead}
	EntryAllPermission  = []string{PermissionGatewayEntryRead, PermissionGatewayEntryWrite}

	RouterReadPermission = []string{PermissionGatewayRouterRead}
	RouterAllPermission  = []string{PermissionGatewayRouterRead, PermissionGatewayRouterWrite}

	FlowReadPermission = []string{PermissionGatewayFlowRead}
	FlowAllPermission  = []string{PermissionGatewayFlowRead, PermissionGatewayFlowWrite}

	IndexAllPermission     = []string{"index:read"}
	IndexReadPermission    = []string{"index:read", "alias:write"}
	AliasAllPermission     = []string{"alias:read"}
	AliasReadPermission    = []string{"alias:read", "alias:write"}
	ViewsAllPermission     = []string{PermissionViewRead, PermissionViewWrite, PermissionLayoutRead, PermissionLayoutWrite}
	ViewsReadPermission    = []string{PermissionViewRead, PermissionLayoutRead}
	DiscoverReadPermission = []string{PermissionViewRead}
	DiscoverAllPermission  = []string{PermissionViewRead}

	RuleReadPermission         = []string{PermissionAlertRuleRead, PermissionAlertHistoryRead}
	RuleAllPermission          = []string{PermissionAlertRuleRead, PermissionAlertRuleWrite, PermissionAlertHistoryRead, PermissionElasticsearchClusterRead}
	AlertReadPermission        = []string{PermissionAlertHistoryRead}
	AlertAllPermission         = []string{PermissionAlertHistoryRead, PermissionAlertHistoryWrite}
	AlertMessageReadPermission = []string{PermissionAlertMessageRead, PermissionAlertHistoryRead}
	AlertMessageAllPermission  = []string{PermissionAlertMessageRead, PermissionAlertMessageWrite, PermissionAlertHistoryRead}
	ChannelReadPermission      = []string{PermissionAlertChannelRead}
	ChannelAllPermission       = []string{PermissionAlertChannelRead, PermissionAlertChannelWrite}

	ClusterOverviewReadPermission = []string{PermissionElasticsearchClusterRead, PermissionElasticsearchIndexRead, PermissionElasticsearchNodeRead, PermissionElasticsearchMetricRead}
	ClusterOverviewAllPermission  = ClusterOverviewReadPermission
	MonitoringReadPermission      = ClusterOverviewAllPermission

	ActivitiesReadPermission = []string{PermissionActivityRead}
	ActivitiesAllPermission  = []string{PermissionActivityRead, PermissionActivityWrite}

	AuditLogsReadPermission = []string{PermissionAuditLogRead}
	AuditLogsAllPermission  = []string{PermissionAuditLogRead, PermissionAuditLogWrite}

	TenantCustomerReadPermission = []string{TenantCustomerRead}
	TenantCustomerAllPermission  = []string{TenantCustomerRead, TenantCustomerAll}

	SubscriptionReadPermission = []string{SubscriptionRead}
	SubscriptionAllPermission  = []string{SubscriptionRead, SubscriptionAll}

	AgentInstanceReadPermission  = []string{PermissionAgentInstanceRead}
	AgentInstanceAllPermission   = []string{PermissionAgentInstanceRead, PermissionAgentInstanceWrite}
	CredentialReadPermission     = []string{PermissionCredentialRead}
	CredentialAllPermission      = []string{PermissionCredentialRead, PermissionCredentialWrite}
	DataMigrationReadPermission  = []string{PermissionMigrationTaskRead}
	DataMigrationAllPermission   = []string{PermissionMigrationTaskRead, PermissionMigrationTaskWrite}
	DataComparisonReadPermission = []string{PermissionComparisonTaskRead}
	DataComparisonAllPermission  = []string{PermissionComparisonTaskRead, PermissionComparisonTaskWrite}
	DashboardReadPermission      = []string{PermissionLayoutRead}
	DashboardAllPermission       = []string{PermissionLayoutRead, PermissionLayoutWrite}
	WorkbenchReadPermission      = []string{PermissionElasticsearchClusterRead, PermissionActivityRead, PermissionAlertMessageRead, PermissionElasticsearchMetricRead}
	WorkbenchAllPermission       = WorkbenchReadPermission
	SmtpServerReadPermission     = []string{PermissionSmtpServerRead}
	SmtpServerAllPermission      = []string{PermissionSmtpServerRead, PermissionSmtpServerWrite}
)

var AdminPrivilege = []string{
	SecurityAll, ClusterAll, CommandAll,
	InstanceAll, EntryAll, RouterAll, FlowAll,
	IndexAll, ViewsAll, DiscoverAll,
	RuleAll, AlertAll, ChannelAll,
	AlertMessageAll,
	ClusterOverviewAll, MonitoringAll, ActivitiesAll,
	AliasAll, AgentInstanceAll, CredentialAll,
	DataMigrationAll, DataComparisonAll, DashboardAll, DevtoolConsoleAll,
	WorkbenchAll, TenantCustomerAll, SubscriptionAll, AuditLogsAll, SmtpServerAll,
}

func init() {

	PermissionMap = map[string][]string{
		UserRead:     UserReadPermission,
		UserAll:      UserAllPermission,
		RoleRead:     RoleReadPermission,
		RoleAll:      RoleAllPermission,
		SecurityAll:  SecurityAllPermission,
		SecurityRead: SecurityReadPermission,

		ClusterRead: ClusterReadPermission,
		ClusterAll:  ClusterAllPermission,
		CommandRead: CommandReadPermission,
		CommandAll:  CommandAllPermission,

		InstanceRead: InstanceReadPermission,
		InstanceAll:  InstanceAllPermission,
		EntryRead:    EntryReadPermission,
		EntryAll:     EntryAllPermission,
		RouterRead:   RouterReadPermission,
		RouterAll:    RouterAllPermission,
		FlowRead:     FlowReadPermission,
		FlowAll:      FlowAllPermission,

		IndexAll:     IndexAllPermission,
		IndexRead:    IndexReadPermission,
		AliasAll:     AliasAllPermission,
		AliasRead:    AliasReadPermission,
		ViewsAll:     ViewsAllPermission,
		ViewsRead:    ViewsReadPermission,
		DiscoverRead: DiscoverReadPermission,
		DiscoverAll:  DiscoverAllPermission,

		RuleRead:         RuleReadPermission,
		RuleAll:          RuleAllPermission,
		AlertRead:        AlertReadPermission,
		AlertAll:         AlertAllPermission,
		ChannelRead:      ChannelReadPermission,
		ChannelAll:       ChannelAllPermission,
		AlertMessageRead: AlertMessageReadPermission,
		AlertMessageAll:  AlertMessageAllPermission,

		ClusterOverviewRead: ClusterOverviewReadPermission,
		ClusterOverviewAll:  ClusterOverviewAllPermission,
		MonitoringAll:       MonitoringReadPermission,
		MonitoringRead:      MonitoringReadPermission,
		ActivitiesAll:       ActivitiesAllPermission,
		ActivitiesRead:      ActivitiesReadPermission,
		AuditLogsAll:        AuditLogsAllPermission,
		AuditLogsRead:       AuditLogsReadPermission,
		AgentInstanceAll:    AgentInstanceAllPermission,
		AgentInstanceRead:   AgentInstanceReadPermission,
		CredentialAll:       CredentialAllPermission,
		CredentialRead:      CredentialReadPermission,
		DataMigrationRead:   DataMigrationReadPermission,
		DataMigrationAll:    DataMigrationAllPermission,
		DataComparisonRead:  DataComparisonReadPermission,
		DataComparisonAll:   DataComparisonAllPermission,
		DashboardRead:       DashboardReadPermission,
		DashboardAll:        DashboardAllPermission,
		WorkbenchAll:        WorkbenchAllPermission,
		WorkbenchRead:       WorkbenchReadPermission,
		TenantCustomerRead:  TenantCustomerReadPermission,
		TenantCustomerAll:   TenantCustomerAllPermission,

		SubscriptionRead: SubscriptionReadPermission,
		SubscriptionAll:  SubscriptionAllPermission,
		SmtpServerRead:   SmtpServerReadPermission,
		SmtpServerAll:    SmtpServerAllPermission,
	}

}
