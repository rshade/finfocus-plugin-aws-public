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
