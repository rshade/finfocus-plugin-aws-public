//go:build !region_use1 && !region_usw1 && !region_usw2 && !region_govw1 && !region_gove1 && !region_euw1 && !region_apse1 && !region_apse2 && !region_apne1 && !region_aps1 && !region_cac1 && !region_sae1

package pricing

// Per-service fallback pricing data for development/testing.
// Used when no region-specific build tag is provided.
// The format matches the AWS Price List API structure to ensure the client can parse it.

// rawEC2JSON contains minimal EC2 pricing data for development/testing.
var rawEC2JSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AmazonEC2",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {
    "SKU_T3MICRO": {
      "sku": "SKU_T3MICRO",
      "productFamily": "Compute Instance",
      "attributes": {
        "instanceType": "t3.micro",
        "operatingSystem": "Linux",
        "tenancy": "Shared",
        "regionCode": "unknown",
        "capacitystatus": "Used",
        "preInstalledSw": "NA"
      }
    },
    "SKU_T3SMALL": {
      "sku": "SKU_T3SMALL",
      "productFamily": "Compute Instance",
      "attributes": {
        "instanceType": "t3.small",
        "operatingSystem": "Linux",
        "tenancy": "Shared",
        "regionCode": "unknown",
        "capacitystatus": "Used",
        "preInstalledSw": "NA"
      }
    },
    "SKU_GP3": {
      "sku": "SKU_GP3",
      "productFamily": "Storage",
      "attributes": {
        "volumeApiName": "gp3",
        "regionCode": "unknown"
      }
    },
    "SKU_GP2": {
      "sku": "SKU_GP2",
      "productFamily": "Storage",
      "attributes": {
        "volumeApiName": "gp2",
        "regionCode": "unknown"
      }
    }
  },
  "terms": {
    "OnDemand": {
      "SKU_T3MICRO": {
        "SKU_T3MICRO.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_T3MICRO",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_T3MICRO.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_T3MICRO.JRTCKXETXF.6YS6EN2CT7",
              "description": "t3.micro hourly rate",
              "unit": "Hrs",
              "pricePerUnit": { "USD": "0.0104" }
            }
          }
        }
      },
      "SKU_T3SMALL": {
        "SKU_T3SMALL.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_T3SMALL",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_T3SMALL.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_T3SMALL.JRTCKXETXF.6YS6EN2CT7",
              "description": "t3.small hourly rate",
              "unit": "Hrs",
              "pricePerUnit": { "USD": "0.0208" }
            }
          }
        }
      },
      "SKU_GP3": {
        "SKU_GP3.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_GP3",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_GP3.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_GP3.JRTCKXETXF.6YS6EN2CT7",
              "description": "gp3 storage rate",
              "unit": "GB-Mo",
              "pricePerUnit": { "USD": "0.08" }
            }
          }
        }
      },
      "SKU_GP2": {
        "SKU_GP2.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_GP2",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_GP2.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_GP2.JRTCKXETXF.6YS6EN2CT7",
              "description": "gp2 storage rate",
              "unit": "GB-Mo",
              "pricePerUnit": { "USD": "0.10" }
            }
          }
        }
      }
    }
  }
}`)

// rawS3JSON contains minimal S3 pricing data for development/testing.
var rawS3JSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AmazonS3",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {},
  "terms": {"OnDemand": {}}
}`)

// rawRDSJSON contains minimal RDS pricing data for development/testing.
var rawRDSJSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AmazonRDS",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {},
  "terms": {"OnDemand": {}}
}`)

// rawEKSJSON contains minimal EKS pricing data for development/testing.
var rawEKSJSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AmazonEKS",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {
    "SKU_EKS_CLUSTER": {
      "sku": "SKU_EKS_CLUSTER",
      "productFamily": "Compute",
      "attributes": {
        "servicecode": "AmazonEKS",
        "regionCode": "unknown"
      }
    }
  },
  "terms": {
    "OnDemand": {
      "SKU_EKS_CLUSTER": {
        "SKU_EKS_CLUSTER.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_EKS_CLUSTER",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_EKS_CLUSTER.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_EKS_CLUSTER.JRTCKXETXF.6YS6EN2CT7",
              "description": "EKS cluster hourly rate",
              "unit": "Hrs",
              "pricePerUnit": { "USD": "0.10" }
            }
          }
        }
      }
    }
  }
}`)

// rawLambdaJSON contains minimal Lambda pricing data for development/testing.
var rawLambdaJSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AWSLambda",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {},
  "terms": {"OnDemand": {}}
}`)

// rawDynamoDBJSON contains minimal DynamoDB pricing data for development/testing.
var rawDynamoDBJSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AmazonDynamoDB",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {},
  "terms": {"OnDemand": {}}
}`)

// rawELBJSON contains minimal ELB pricing data for development/testing.
var rawELBJSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AWSELB",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {
    "SKU_ALB_HOURLY": {
      "sku": "SKU_ALB_HOURLY",
      "productFamily": "Load Balancer-Application",
      "attributes": {
        "regionCode": "unknown",
        "usagetype": "LoadBalancerUsage"
      }
    },
    "SKU_ALB_LCU": {
      "sku": "SKU_ALB_LCU",
      "productFamily": "Load Balancer-Application",
      "attributes": {
        "regionCode": "unknown",
        "usagetype": "LCUUsage"
      }
    },
    "SKU_NLB_HOURLY": {
      "sku": "SKU_NLB_HOURLY",
      "productFamily": "Load Balancer-Network",
      "attributes": {
        "regionCode": "unknown",
        "usagetype": "LoadBalancerUsage"
      }
    },
    "SKU_NLB_NLCU": {
      "sku": "SKU_NLB_NLCU",
      "productFamily": "Load Balancer-Network",
      "attributes": {
        "regionCode": "unknown",
        "usagetype": "NLCUUsage"
      }
    }
  },
  "terms": {
    "OnDemand": {
      "SKU_ALB_HOURLY": {
        "SKU_ALB_HOURLY.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_ALB_HOURLY",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_ALB_HOURLY.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_ALB_HOURLY.JRTCKXETXF.6YS6EN2CT7",
              "description": "ALB hourly rate",
              "unit": "Hrs",
              "pricePerUnit": { "USD": "0.0225" }
            }
          }
        }
      },
      "SKU_ALB_LCU": {
        "SKU_ALB_LCU.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_ALB_LCU",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_ALB_LCU.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_ALB_LCU.JRTCKXETXF.6YS6EN2CT7",
              "description": "ALB LCU rate",
              "unit": "LCU-Hrs",
              "pricePerUnit": { "USD": "0.008" }
            }
          }
        }
      },
      "SKU_NLB_HOURLY": {
        "SKU_NLB_HOURLY.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_NLB_HOURLY",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_NLB_HOURLY.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_NLB_HOURLY.JRTCKXETXF.6YS6EN2CT7",
              "description": "NLB hourly rate",
              "unit": "Hrs",
              "pricePerUnit": { "USD": "0.0225" }
            }
          }
        }
      },
      "SKU_NLB_NLCU": {
        "SKU_NLB_NLCU.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_NLB_NLCU",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_NLB_NLCU.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_NLB_NLCU.JRTCKXETXF.6YS6EN2CT7",
              "description": "NLB NLCU rate",
              "unit": "NLCU-Hrs",
              "pricePerUnit": { "USD": "0.006" }
            }
          }
        }
      }
    }
  }
}`)

// rawVPCJSON contains minimal VPC pricing data for development/testing.
var rawVPCJSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AmazonVPC",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {
    "SKU_NATGW_HOURLY": {
      "sku": "SKU_NATGW_HOURLY",
      "productFamily": "NAT Gateway",
      "attributes": {
        "regionCode": "unknown",
        "usagetype": "NatGateway-Hours"
      }
    },
    "SKU_NATGW_BYTES": {
      "sku": "SKU_NATGW_BYTES",
      "productFamily": "NAT Gateway",
      "attributes": {
        "regionCode": "unknown",
        "usagetype": "NatGateway-Bytes"
      }
    }
  },
  "terms": {
    "OnDemand": {
      "SKU_NATGW_HOURLY": {
        "SKU_NATGW_HOURLY.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_NATGW_HOURLY",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_NATGW_HOURLY.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_NATGW_HOURLY.JRTCKXETXF.6YS6EN2CT7",
              "description": "NAT Gateway hourly rate",
              "unit": "Hrs",
              "pricePerUnit": { "USD": "0.045" }
            }
          }
        }
      },
      "SKU_NATGW_BYTES": {
        "SKU_NATGW_BYTES.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_NATGW_BYTES",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_NATGW_BYTES.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_NATGW_BYTES.JRTCKXETXF.6YS6EN2CT7",
              "description": "NAT Gateway data processing rate",
              "unit": "Quantity",
              "pricePerUnit": { "USD": "0.045" }
            }
          }
        }
      }
    }
  }
}`)

// rawCloudWatchJSON contains minimal CloudWatch pricing data for development/testing.
var rawCloudWatchJSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AmazonCloudWatch",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {},
  "terms": {"OnDemand": {}}
}`)

// rawElastiCacheJSON contains minimal ElastiCache pricing data for development/testing.
// Includes test data for Redis, Memcached, and Valkey engines with cache.t3.micro and cache.m5.large instance types.
var rawElastiCacheJSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AmazonElastiCache",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {
    "SKU_CACHE_T3MICRO_REDIS": {
      "sku": "SKU_CACHE_T3MICRO_REDIS",
      "productFamily": "Cache Instance",
      "attributes": {
        "instanceType": "cache.t3.micro",
        "cacheEngine": "Redis",
        "regionCode": "unknown"
      }
    },
    "SKU_CACHE_T3MICRO_MEMCACHED": {
      "sku": "SKU_CACHE_T3MICRO_MEMCACHED",
      "productFamily": "Cache Instance",
      "attributes": {
        "instanceType": "cache.t3.micro",
        "cacheEngine": "Memcached",
        "regionCode": "unknown"
      }
    },
    "SKU_CACHE_T3MICRO_VALKEY": {
      "sku": "SKU_CACHE_T3MICRO_VALKEY",
      "productFamily": "Cache Instance",
      "attributes": {
        "instanceType": "cache.t3.micro",
        "cacheEngine": "Valkey",
        "regionCode": "unknown"
      }
    },
    "SKU_CACHE_M5LARGE_REDIS": {
      "sku": "SKU_CACHE_M5LARGE_REDIS",
      "productFamily": "Cache Instance",
      "attributes": {
        "instanceType": "cache.m5.large",
        "cacheEngine": "Redis",
        "regionCode": "unknown"
      }
    }
  },
  "terms": {
    "OnDemand": {
      "SKU_CACHE_T3MICRO_REDIS": {
        "SKU_CACHE_T3MICRO_REDIS.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_CACHE_T3MICRO_REDIS",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_CACHE_T3MICRO_REDIS.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_CACHE_T3MICRO_REDIS.JRTCKXETXF.6YS6EN2CT7",
              "description": "cache.t3.micro Redis hourly rate",
              "unit": "Hrs",
              "pricePerUnit": { "USD": "0.017" }
            }
          }
        }
      },
      "SKU_CACHE_T3MICRO_MEMCACHED": {
        "SKU_CACHE_T3MICRO_MEMCACHED.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_CACHE_T3MICRO_MEMCACHED",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_CACHE_T3MICRO_MEMCACHED.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_CACHE_T3MICRO_MEMCACHED.JRTCKXETXF.6YS6EN2CT7",
              "description": "cache.t3.micro Memcached hourly rate",
              "unit": "Hrs",
              "pricePerUnit": { "USD": "0.017" }
            }
          }
        }
      },
      "SKU_CACHE_T3MICRO_VALKEY": {
        "SKU_CACHE_T3MICRO_VALKEY.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_CACHE_T3MICRO_VALKEY",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_CACHE_T3MICRO_VALKEY.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_CACHE_T3MICRO_VALKEY.JRTCKXETXF.6YS6EN2CT7",
              "description": "cache.t3.micro Valkey hourly rate",
              "unit": "Hrs",
              "pricePerUnit": { "USD": "0.017" }
            }
          }
        }
      },
      "SKU_CACHE_M5LARGE_REDIS": {
        "SKU_CACHE_M5LARGE_REDIS.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_CACHE_M5LARGE_REDIS",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_CACHE_M5LARGE_REDIS.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_CACHE_M5LARGE_REDIS.JRTCKXETXF.6YS6EN2CT7",
              "description": "cache.m5.large Redis hourly rate",
              "unit": "Hrs",
              "pricePerUnit": { "USD": "0.156" }
            }
          }
        }
      }
    }
  }
}`)

// rawECSJSON contains minimal ECS pricing data for development/testing.
// Includes the Linux x86_64 and arm64 Fargate vCPU-hour and GiB-hour rates.
var rawECSJSON = []byte(`{
  "formatVersion": "v1.0",
  "disclaimer": "Fallback data for development/testing only",
  "offerCode": "AmazonECS",
  "version": "fallback",
  "publicationDate": "2024-01-01T00:00:00Z",
  "products": {
    "SKU_FARGATE_X86_VCPU": {
      "sku": "SKU_FARGATE_X86_VCPU",
      "productFamily": "Compute",
      "attributes": {
        "usagetype": "USE1-Fargate-vCPU-Hours:perCPU",
        "locationType": "AWS Region",
        "regionCode": "unknown"
      }
    },
    "SKU_FARGATE_X86_GIB": {
      "sku": "SKU_FARGATE_X86_GIB",
      "productFamily": "Compute",
      "attributes": {
        "usagetype": "USE1-Fargate-GB-Hours",
        "locationType": "AWS Region",
        "regionCode": "unknown"
      }
    },
    "SKU_FARGATE_ARM_VCPU": {
      "sku": "SKU_FARGATE_ARM_VCPU",
      "productFamily": "Compute",
      "attributes": {
        "usagetype": "USE1-Fargate-ARM-vCPU-Hours:perCPU",
        "locationType": "AWS Region",
        "regionCode": "unknown"
      }
    },
    "SKU_FARGATE_ARM_GIB": {
      "sku": "SKU_FARGATE_ARM_GIB",
      "productFamily": "Compute",
      "attributes": {
        "usagetype": "USE1-Fargate-ARM-GB-Hours",
        "locationType": "AWS Region",
        "regionCode": "unknown"
      }
    },
    "SKU_FARGATE_WIN_VCPU": {
      "sku": "SKU_FARGATE_WIN_VCPU",
      "productFamily": "Compute",
      "attributes": {
        "usagetype": "USE1-Fargate-Windows-vCPU-Hours:perCPU",
        "locationType": "AWS Region",
        "regionCode": "unknown"
      }
    },
    "SKU_FARGATE_WIN_GIB": {
      "sku": "SKU_FARGATE_WIN_GIB",
      "productFamily": "Compute",
      "attributes": {
        "usagetype": "USE1-Fargate-Windows-GB-Hours",
        "locationType": "AWS Region",
        "regionCode": "unknown"
      }
    },
    "SKU_FARGATE_WIN_OS": {
      "sku": "SKU_FARGATE_WIN_OS",
      "productFamily": "Compute",
      "attributes": {
        "usagetype": "USE1-Fargate-Windows-OS-Hours:perCPU",
        "locationType": "AWS Region",
        "regionCode": "unknown"
      }
    },
    "SKU_FARGATE_EPHEMERAL": {
      "sku": "SKU_FARGATE_EPHEMERAL",
      "productFamily": "Compute",
      "attributes": {
        "usagetype": "USE1-Fargate-EphemeralStorage-GB-Hours",
        "locationType": "AWS Region",
        "regionCode": "unknown"
      }
    }
  },
  "terms": {
    "OnDemand": {
      "SKU_FARGATE_X86_VCPU": {
        "SKU_FARGATE_X86_VCPU.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_FARGATE_X86_VCPU",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_FARGATE_X86_VCPU.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_FARGATE_X86_VCPU.JRTCKXETXF.6YS6EN2CT7",
              "description": "AWS Fargate - vCPU",
              "unit": "hours",
              "pricePerUnit": { "USD": "0.04048" }
            }
          }
        }
      },
      "SKU_FARGATE_X86_GIB": {
        "SKU_FARGATE_X86_GIB.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_FARGATE_X86_GIB",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_FARGATE_X86_GIB.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_FARGATE_X86_GIB.JRTCKXETXF.6YS6EN2CT7",
              "description": "AWS Fargate - Memory",
              "unit": "hours",
              "pricePerUnit": { "USD": "0.004445" }
            }
          }
        }
      },
      "SKU_FARGATE_ARM_VCPU": {
        "SKU_FARGATE_ARM_VCPU.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_FARGATE_ARM_VCPU",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_FARGATE_ARM_VCPU.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_FARGATE_ARM_VCPU.JRTCKXETXF.6YS6EN2CT7",
              "description": "AWS Fargate - ARM - vCPU",
              "unit": "hours",
              "pricePerUnit": { "USD": "0.03238" }
            }
          }
        }
      },
      "SKU_FARGATE_ARM_GIB": {
        "SKU_FARGATE_ARM_GIB.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_FARGATE_ARM_GIB",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_FARGATE_ARM_GIB.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_FARGATE_ARM_GIB.JRTCKXETXF.6YS6EN2CT7",
              "description": "AWS Fargate - ARM - Memory",
              "unit": "hours",
              "pricePerUnit": { "USD": "0.003560" }
            }
          }
        }
      },
      "SKU_FARGATE_WIN_VCPU": {
        "SKU_FARGATE_WIN_VCPU.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_FARGATE_WIN_VCPU",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_FARGATE_WIN_VCPU.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_FARGATE_WIN_VCPU.JRTCKXETXF.6YS6EN2CT7",
              "description": "AWS Fargate - Windows - vCPU",
              "unit": "hours",
              "pricePerUnit": { "USD": "0.046552" }
            }
          }
        }
      },
      "SKU_FARGATE_WIN_GIB": {
        "SKU_FARGATE_WIN_GIB.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_FARGATE_WIN_GIB",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_FARGATE_WIN_GIB.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_FARGATE_WIN_GIB.JRTCKXETXF.6YS6EN2CT7",
              "description": "AWS Fargate - Windows - Memory",
              "unit": "hours",
              "pricePerUnit": { "USD": "0.00511175" }
            }
          }
        }
      },
      "SKU_FARGATE_WIN_OS": {
        "SKU_FARGATE_WIN_OS.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_FARGATE_WIN_OS",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_FARGATE_WIN_OS.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_FARGATE_WIN_OS.JRTCKXETXF.6YS6EN2CT7",
              "description": "AWS Fargate - Windows - OS license",
              "unit": "hours",
              "pricePerUnit": { "USD": "0.046" }
            }
          }
        }
      },
      "SKU_FARGATE_EPHEMERAL": {
        "SKU_FARGATE_EPHEMERAL.JRTCKXETXF": {
          "offerTermCode": "JRTCKXETXF",
          "sku": "SKU_FARGATE_EPHEMERAL",
          "effectiveDate": "2024-01-01T00:00:00Z",
          "priceDimensions": {
            "SKU_FARGATE_EPHEMERAL.JRTCKXETXF.6YS6EN2CT7": {
              "rateCode": "SKU_FARGATE_EPHEMERAL.JRTCKXETXF.6YS6EN2CT7",
              "description": "AWS Fargate - Ephemeral Storage",
              "unit": "GB-Hours",
              "pricePerUnit": { "USD": "0.000111" }
            }
          }
        }
      }
    }
  }
}`)
