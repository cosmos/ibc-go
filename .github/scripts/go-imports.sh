#!/usr/bin/env sh

modified_files="$(git diff --name-only | grep .go$ | grep -v pb.go)"
if [ "${modified_files// /}" ]; then
  echo "No go files changed"
  exit 0
fi

formatted_files="$(docker run -v "$(pwd)":/ibc-go --rm -w "/ibc-go" cytopia/goimports -l -local 'github.com/cosmos/ibc-go' $modified_files)"
echo $formatted_files

if [[ ${formatted_files} ]]; then
  echo "Files were not formatted correctly to format them run the following command to format them:"
  echo "make goimports"
  exit 1
fi

echo "All files correctly formatted!"
exit 0
