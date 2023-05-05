#!/bin/bash

# Get the directory path where JSON files are located
directory_path=$(dirname "$0")

# Create a backup directory with the current date and time
backup_dir="${directory_path}_backup_$(date +%Y-%m-%d-%H-%M-%S)"
mkdir "$backup_dir"

# Copy the contents of the directory to the backup directory
cp -R "$directory_path"/* "$backup_dir"

# Step 1: Replace the release version in JSON files
echo "Enter the release version to replace (ie: v4.4.0, leave empty to skip):"
read old_release_version

if [ -n "$old_release_version" ]; then
  echo "Enter the current release version:"
  read new_release_version
  find "$directory_path" -name "*.json" -type f -print0 | while IFS= read -r -d '' file; do
      # Use jq to filter JSON objects that contain arrays of strings
      jq -c 'select(type == "object" and any(.[]; type == "array" and all(.[]; type == "string")))' "$file" | while read -r object; do
          # Use jq to replace the old release version with the new release version
          updated_object=$(echo "$object" | jq ".[] |= map(if . == \"$old_release_version\" then \"$new_release_version\" else . end)")
          # Use jq to update the object in the JSON file
          jq --argjson updated_object "$updated_object" '. |= $updated_object' "$file" > tmp.json && mv tmp.json "$file"
          echo "Replaced ${old_release_version} with ${new_release_version} in ${file}"
      done
  done
fi

# Step 2: Add new release version to testing configuration files
read -p "Enter the new release version that you would like to insert (ie: v4.5.0): " new_version
read -p "Enter the most recent release version (ie: v4.4.0): " recent_version

# loop through all json files in directory and its subdirectories
for file in $(find "$directory_path" -name "*.json"); do

  # parse the json file and search for the recent_version string
  json=$(cat $file)
  if [[ $json == *"$recent_version"* ]]; then

    # add new_version string to the json array containing recent_version string
    updated_json=$(echo $json | jq ".[] |= if(index(\"$recent_version\")) then . + [\"$new_version\"] else . end")

    # write the updated json to file
    echo $updated_json > $file
    echo "Updated $file with new release version."
  fi
done