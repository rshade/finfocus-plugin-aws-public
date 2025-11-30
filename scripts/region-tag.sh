#!/bin/bash
# Maps AWS region names to Go build tags
# Usage: ./scripts/region-tag.sh <region>
# Example: ./scripts/region-tag.sh us-east-1 → use1

case "$1" in
    us-east-1)      echo "use1" ;;
    us-west-1)      echo "usw1" ;;
    us-west-2)      echo "usw2" ;;
    us-gov-west-1)  echo "govw1" ;;
    us-gov-east-1)  echo "gove1" ;;
    eu-west-1)      echo "euw1" ;;
    ap-southeast-1) echo "apse1" ;;
    ap-southeast-2) echo "apse2" ;;
    ap-northeast-1) echo "apne1" ;;
    ap-south-1)     echo "aps1" ;;
    ca-central-1)   echo "cac1" ;;
    sa-east-1)      echo "sae1" ;;
    *)
        echo "Unknown region: $1" >&2
        echo "Supported regions:" >&2
        echo "  us-east-1, us-west-1, us-west-2, us-gov-west-1, us-gov-east-1" >&2
        echo "  eu-west-1" >&2
        echo "  ap-southeast-1, ap-southeast-2, ap-northeast-1, ap-south-1" >&2
        echo "  ca-central-1, sa-east-1" >&2
        exit 1
        ;;
esac
