module github.com/rshade/finfocus-plugin-aws-public/tools/generate-goreleaser

go 1.25.5

require github.com/rshade/finfocus-plugin-aws-public v0.0.0

require gopkg.in/yaml.v3 v3.0.1 // indirect

// Use local parent module for internal packages
replace github.com/rshade/finfocus-plugin-aws-public => ../..
