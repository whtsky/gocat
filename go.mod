module github.com/whtsky/gocat

require (
	github.com/kelseyhightower/envconfig v1.3.0
	github.com/magefile/mage v1.8.0
	github.com/palantir/stacktrace v0.0.0-20161112013806-78658fd2d177
	github.com/spf13/cobra v0.0.3
	github.com/stretchr/testify v1.4.0
	github.com/sumup-oss/go-pkgs v0.0.0-20200803091251-631821eeafd6
	github.com/whtsky/gocat/relay v0.0.0-00010101000000-000000000000
)

replace github.com/whtsky/gocat/relay => ./relay

go 1.13
