export AWS_USERNAME=$(shell aws iam get-user --output text --query "User.UserName")
export IAM_GROUPS=$(shell aws iam list-groups-for-user --user-name ${AWS_USERNAME} --output text --query "join(',', Groups[].GroupName)")
export LATEST=$$LATEST


config:
	@envsubst <blessconfig.cfg.tpl
ssh:
	@ssh-keygen -f ~/.ssh/blessid -b 4096 -t rsa -C 'Key for Bless certificates' -N ''
	@ssh-keygen -y -f ~/.ssh/blessid > ~/.ssh/blessid.pub

install:
	go install main.go

.PHONY: config ssh install
