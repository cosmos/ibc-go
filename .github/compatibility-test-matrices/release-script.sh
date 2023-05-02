#!/bin/bash

# Get the directory path where JSON files are located
directory_path=$(dirname "$0")

# Create a backup directory with the current date and time
backup_dir="${directory_path}_backup_$(date +%Y-%m-%d-%H-%M-%S)"
mkdir "$backup_dir"

# Copy the contents of the directory to the backup directory
cp -R "$directory_path"/* "$backup_dir"

# Step 1: Replace the release version in JSON files
echo "Enter the release version to replace (leave empty to skip):"
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

# Step 2: Add a release version to the compatibility matrix
echo "Enter the release version to add to the compatibility matrix:"
read new_compatibility_version
echo "Enter the latest release version:"
read latest_release_version

find "$directory_path" -name "*.json" -type f -print0 | while IFS= read -r -d '' file; do
    # Use jq to filter JSON objects that contain arrays of strings
    jq -c 'select(type == "object" and any(.[]; type == "array" and all(.[]; type == "string")))' "$file" | while read -r object; do
        # Use jq to find the array containing the latest release version and add the new compatibility version to it
        updated_object=$(echo "$object" | jq ".[] |= map(if . == \"$latest_release_version\" then . + [\"$new_compatibility_version\"] else . end)")
        # Use jq to update the object in the JSON file
        jq --argjson updated_object "$updated_object" '. |= $updated_object' "$file" > tmp.json && mv tmp.json "$file"
        echo "Added ${new_compatibility_version} to the compatibility matrix in ${file}"
    done
done

echo "Done."