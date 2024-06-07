#!/bin/bash

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <yaml-file> <n>"
  exit 1
fi

FILE=$1
N=$2

for ((i=1; i<=N; i++))
do
  NEW_UUID=$(uuid)
  REPLICAS=$((RANDOM % 2 + 1))
  
  yq eval ".metadata.name = \"${NEW_UUID}\"" -i "$FILE"
  yq eval ".spec.replicas = ${REPLICAS}" -i "$FILE"

  kubectl create namespace "$NEW_UUID"
  kubectl apply -f "$FILE" -n "$NEW_UUID"
done
