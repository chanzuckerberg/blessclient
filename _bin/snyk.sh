# /bin/bash -ex

if [ -z "$SNYK_TOKEN" ]; then
  echo "SNYK_TOKEN not set, skipping"
  exit 0
fi

snyk monitor --org=czi
snyk test
