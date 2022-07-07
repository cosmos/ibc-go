#!/usr/bin/env sh
formatted_files="$(goimports -l -local -w 'github.com/cosmos/ibc-go' .)"

exit_code=0
for f in $formatted_files
do
  # we don't care about formatting in pb.go files.
  if [ "${f: -5}" == "pb.go" ]; then
    continue
  fi
  exit_code=1
  echo "formatted file ${f}..."
done

exit $exit_code
