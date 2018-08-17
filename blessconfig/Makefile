export IAM_GROUPS=$(shell aws iam list-groups-for-user --user-name elopez@chanzuckerberg.com --output text --query "join(',', Groups[].GroupName)")
export LATEST=$$LATEST


config:
	@envsubst <blessconfig.cfg.tpl
