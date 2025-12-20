//go:build !region_use1 && !region_usw1 && !region_usw2 && !region_govw1 && !region_gove1 && !region_euw1 && !region_apse1 && !region_apse2 && !region_apne1 && !region_aps1 && !region_cac1 && !region_sae1

package pricing

// rawPricingJSON contains fallback dummy pricing data for development/testing.
// This is used when no region-specific build tag is provided.
// The format matches the AWS Price List API structure to ensure the client can parse it.
var rawPricingJSON = []byte(`{
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
    },
    "SKU_EKS_CLUSTER": {
      "sku": "SKU_EKS_CLUSTER",
      "productFamily": "Compute",
      "attributes": {
        "servicecode": "AmazonEKS",
        "regionCode": "unknown"
      }
    },
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
        "usagetype": "LCUUsage"
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
      },
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
      },
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
              "unit": "LCU-Hrs",
              "pricePerUnit": { "USD": "0.006" }
            }
          }
        }
      }
    }
  }
}`)
