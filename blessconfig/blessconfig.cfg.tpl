[MAIN]
region_aliases: WEST,EAST
kms_service_name: shared-infra-prod-bless
bastion_ips: 10.0.0.0/8
remote_user: ${IAM_GROUPS}
ca_backend: bless

[CLIENT]
domain_regex: .*
cache_dir: .bless/session
cache_file: bless_cache.json
mfa_cache_dir: .aws/session
mfa_cache_file: token_cache.json
ip_urls: http://api.ipify.org, http://canihazip.com
update_script: update_blessclient.sh

[LAMBDA]
user_role: blessclient
account_id: 416703108729
functionname: shared-infra-prod-bless
certlifetime: 1800
functionversion: $LATEST
ipcachelifetime: 30
timeout_connect: 5
timeout_read: 10

[REGION_WEST]
awsregion: us-west-2
kmsauthkey: arn:aws:kms:us-west-2:416703108729:key/fe4c9d09-5006-4cb3-bb48-8b98476d3600

[REGION_EAST]
awsregion: us-east-2
kmsauthkey: arn:aws:kms:us-east-2:416703108729:key/3e3781fa-3d67-451e-8ea7-396c9120034f

