// Package pluginid holds the name this plugin reports to FinFocus core.
package pluginid

// Name is the plugin's registry name. The region binaries and the router
// report it from Name() and GetPluginInfo, and core matches routing rules and
// --adapter against it, so it must equal the name the FinFocus registry
// installs the plugin under. Binary and release asset names are separate and
// keep the finfocus-plugin-aws-public prefix.
const Name = "aws-public"
