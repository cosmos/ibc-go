#!/usr/bin/env bash
formatted_files="$(docker run -v "$(pwd)":/ibc-go --rm -w "/ibc-go" --entrypoint="" cytopia/goimports goimports -l -local 'github.com/cosmos/ibc-go' /ibc-go)"

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

if [ "${exit_code}" == 1 ]; then
    echo "not all files were correctly formated, run the following:"
    echo "make goimports"
fi

exit $exit_code
