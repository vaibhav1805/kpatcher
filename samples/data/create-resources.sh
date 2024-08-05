#!/bin/bash

# Use this script to create sample data
if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <yaml-file> <n>"
  exit 1
fi

FILE=$1
N=$2

for ((i=1; i<=N; i++))
do
  NAME="deployment-${i}"
  REPLICAS=$((RANDOM % 2 + 1))
  
  yq eval ".metadata.name = \"${NAME}\"" -i "$FILE"
  yq eval ".spec.replicas = ${REPLICAS}" -i "$FILE"

  kubectl create namespace "$NAME"
  kubectl apply -f "$FILE" -n "$NAME"
done
