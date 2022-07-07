#!/usr/bin/env sh
formatted_files="$(goimports -l -local 'github.com/cosmos/ibc-go' .)"

exit_code=0
for f in $formatted_files
do
  # we don't care about formatting in pb.go files.
  if [ "${f: -5}" == "pb.go" ]; then
    continue
  fi
  exit_code=1
  echo "docker run  -v "$(pwd)":/ibc-go --rm  -w "/ibc-go"  cytopia/goimports -local 'github.com/cosmos/ibc-go' -w /ibc-go/${f}"
done

if [[ ${exit_code} == 1 ]]; then
  echo "Files were not formatted correctly, run the above commands to format them."
fi

exit $exit_code
