module github.com/rshade/pulumicost-plugin-aws-public/tools/generate-goreleaser

go 1.25.5

require github.com/rshade/pulumicost-plugin-aws-public v0.0.17

require gopkg.in/yaml.v3 v3.0.1 // indirect

// Use local parent module for internal packages
replace github.com/rshade/pulumicost-plugin-aws-public => ../..
