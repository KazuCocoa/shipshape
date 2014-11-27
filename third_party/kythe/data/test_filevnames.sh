#!/bin/bash -e

# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Ensures that vnames.json can be read by the //kythe/storage/go/filevnames library

CAMPFIRE_ROOT="$(readlink -e "$PWD")"
DIRECTORY_INDEXER="${CAMPFIRE_ROOT}/campfire-out/bin/third_party/kythe/go/storage/tools/directory_indexer"
CONFIG="${CAMPFIRE_ROOT}/third_party/kythe/data/vnames.json"
OUT="${CAMPFIRE_ROOT}/campfire-out/gen/third_party/kythe/data/file_entries"

# Directory tree with some (but not many) files
DIR="${CAMPFIRE_ROOT}/third_party/kythe/go/platform"

mkdir -p "$(dirname "$OUT")"
cd "$DIR"
"$DIRECTORY_INDEXER" --vnames "$CONFIG" >"$OUT"

test -s "$OUT" || {
  echo "$OUT is empty" >&2
  exit 1
}
