module github.com/rshade/finfocus-plugin-aws-public/tools/generate-embeds

go 1.25.8

require github.com/rshade/finfocus-plugin-aws-public v0.1.8

require gopkg.in/yaml.v3 v3.0.1 // indirect

// Use local parent module for internal packages
replace github.com/rshade/finfocus-plugin-aws-public => ../..
