export IAM_GROUPS=$(shell aws iam list-groups-for-user --user-name elopez@chanzuckerberg.com --output text --query "join(',', Groups[].GroupName)")
export LATEST=$$LATEST


config:
	@envsubst <blessconfig.cfg.tpl
ssh:
	@ssh-keygen -f ~/.ssh/blessid -b 4096 -t rsa -C 'Key for Bless certificates' -N ''
	@ssh-keygen -y -f ~/.ssh/blessid > ~/.ssh/blessid.pub
